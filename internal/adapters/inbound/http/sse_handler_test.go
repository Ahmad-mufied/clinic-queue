package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type mockNATSSubscriber struct {
	subscribeFunc func(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
}

func (m *mockNATSSubscriber) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	if m.subscribeFunc != nil {
		return m.subscribeFunc(subj, cb)
	}
	return nil, nil
}

type errorResponseWriter struct {
	http.ResponseWriter
	errOnWrite atomic.Bool
}

func (w *errorResponseWriter) Write(p []byte) (int, error) {
	if w.errOnWrite.Load() {
		return 0, errors.New("simulated write error")
	}
	return w.ResponseWriter.Write(p)
}

func (w *errorResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestSSEHandler_HubOperations(t *testing.T) {
	handler := NewSSEHandler()

	// 1. Subscribe client 1 & 2
	ch1 := handler.Subscribe()
	ch2 := handler.Subscribe()

	if len(handler.clients) != 2 {
		t.Fatalf("expected 2 subscribers, got %d", len(handler.clients))
	}

	// 2. Broadcast message
	testPayload := []byte(`{"action":"QUEUE_JOINED"}`)
	handler.Broadcast(testPayload)

	select {
	case msg := <-ch1:
		if string(msg) != string(testPayload) {
			t.Errorf("ch1 received %s, want %s", string(msg), string(testPayload))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message on ch1")
	}

	select {
	case msg := <-ch2:
		if string(msg) != string(testPayload) {
			t.Errorf("ch2 received %s, want %s", string(msg), string(testPayload))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for message on ch2")
	}

	// 3. Test Broadcast drop when buffer is full (capacity 32)
	for i := 0; i < 35; i++ {
		handler.Broadcast([]byte("overflow"))
	}

	// 4. Unsubscribe
	handler.Unsubscribe(ch1)
	handler.Unsubscribe(ch2)

	if len(handler.clients) != 0 {
		t.Fatalf("expected 0 subscribers after unsubscribe, got %d", len(handler.clients))
	}

	// Unsubscribe nonexistent channel does not panic
	dummyCh := make(chan []byte)
	handler.Unsubscribe(dummyCh)
}

func TestSSEHandler_HandleEvents_Flow(t *testing.T) {
	e := echo.New()
	handler := NewSSEHandler()
	handler.pingInterval = 10 * time.Millisecond // Fast ping for testing

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = handler.HandleEvents(c)
	}()

	// Wait briefly for connection and initial handshake
	time.Sleep(20 * time.Millisecond)

	// Broadcast an event
	handler.Broadcast([]byte(`{"ticket":"A-01"}`))

	// Wait for keepalive ping
	time.Sleep(25 * time.Millisecond)

	// Cancel context to simulate client disconnect
	cancel()
	wg.Wait()

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"connected"`) {
		t.Errorf("expected response to contain connected status, got: %s", body)
	}
	if !strings.Contains(body, `{"ticket":"A-01"}`) {
		t.Errorf("expected response to contain broadcast event, got: %s", body)
	}
	if !strings.Contains(body, ":keepalive") {
		t.Errorf("expected response to contain keepalive ping, got: %s", body)
	}
	if rec.Header().Get(echo.HeaderContentType) != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", rec.Header().Get(echo.HeaderContentType))
	}
}

func TestSSEHandler_HandleEvents_ErrorBranches(t *testing.T) {
	e := echo.New()
	handler := NewSSEHandler()
	handler.pingInterval = 10 * time.Millisecond

	// 1. Initial write failure
	{
		rec := httptest.NewRecorder()
		errWriter := &errorResponseWriter{ResponseWriter: rec}
		errWriter.errOnWrite.Store(true)
		req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		c := e.NewContext(req, errWriter)

		err := handler.HandleEvents(c)
		if err != nil {
			t.Errorf("expected nil from HandleEvents on write error, got %v", err)
		}
	}

	// 2. Closed clientChan
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
		c := e.NewContext(req, rec)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handler.HandleEvents(c)
		}()

		time.Sleep(15 * time.Millisecond)

		// Find registered channel and close it
		handler.mu.Lock()
		for ch := range handler.clients {
			close(ch)
			delete(handler.clients, ch)
			break
		}
		handler.mu.Unlock()

		wg.Wait()
	}

	// 3. Event write error
	{
		rec := httptest.NewRecorder()
		errWriter := &errorResponseWriter{ResponseWriter: rec}
		req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		c := e.NewContext(req, errWriter)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handler.HandleEvents(c)
		}()

		time.Sleep(15 * time.Millisecond)
		errWriter.errOnWrite.Store(true)
		handler.Broadcast([]byte(`{"test":true}`))

		wg.Wait()
	}

	// 4. Keepalive write error
	{
		rec := httptest.NewRecorder()
		errWriter := &errorResponseWriter{ResponseWriter: rec}
		req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		c := e.NewContext(req, errWriter)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handler.HandleEvents(c)
		}()

		time.Sleep(5 * time.Millisecond)
		errWriter.errOnWrite.Store(true)
		time.Sleep(20 * time.Millisecond)

		wg.Wait()
	}
}

func TestSSEHandler_ListenToNATS(t *testing.T) {
	handler := NewSSEHandler()

	// 1. Nil subscriber error
	_, err := handler.ListenToNATS(context.Background(), nil, "clinic.>")
	if err == nil {
		t.Error("expected error with nil nats subscriber, got nil")
	}

	// 2. Subscribe error
	mockSub := &mockNATSSubscriber{
		subscribeFunc: func(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
			return nil, errors.New("nats subscribe failure")
		},
	}
	_, err = handler.ListenToNATS(context.Background(), mockSub, "clinic.>")
	if err == nil {
		t.Error("expected error from failed Subscribe, got nil")
	}

	// 3. Successful subscription and message reception
	var registeredCB nats.MsgHandler
	ctx, cancel := context.WithCancel(context.Background())

	mockSubSuccess := &mockNATSSubscriber{
		subscribeFunc: func(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
			registeredCB = cb
			return &nats.Subscription{}, nil
		},
	}

	clientCh := handler.Subscribe()
	defer handler.Unsubscribe(clientCh)

	_, err = handler.ListenToNATS(ctx, mockSubSuccess, "clinic.>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registeredCB == nil {
		t.Fatal("expected MsgHandler callback to be registered")
	}

	// Simulate incoming NATS message
	registeredCB(&nats.Msg{Data: []byte(`{"event":"QUEUE_JOINED"}`)})

	select {
	case msg := <-clientCh:
		if !strings.Contains(string(msg), "QUEUE_JOINED") {
			t.Errorf("expected received message to contain QUEUE_JOINED, got %s", string(msg))
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcasted NATS message")
	}

	cancel()
	time.Sleep(10 * time.Millisecond)
}

func TestSSEHandler_RegisterRoutes(t *testing.T) {
	e := echo.New()
	handler := NewSSEHandler()

	dummyMW := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(c)
		}
	}

	handler.RegisterRoutes(e, dummyMW)

	routes := e.Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/api/events" && r.Method == http.MethodGet {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GET /api/events route to be registered")
	}
}
