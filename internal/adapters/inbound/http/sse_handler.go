package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

// SSEBroadcaster defines the interface for managing SSE subscribers and broadcasting events.
type SSEBroadcaster interface {
	Subscribe() chan []byte
	Unsubscribe(ch chan []byte)
	Broadcast(event []byte)
}

// SSEHandler manages Server-Sent Events client streaming and NATS subscriptions.
type SSEHandler struct {
	mu           sync.RWMutex
	clients      map[chan []byte]struct{}
	pingInterval time.Duration
}

// NewSSEHandler constructs a new SSEHandler.
func NewSSEHandler() *SSEHandler {
	return &SSEHandler{
		clients:      make(map[chan []byte]struct{}),
		pingInterval: 15 * time.Second,
	}
}

// RegisterRoutes registers the SSE endpoint on Echo router.
func (h *SSEHandler) RegisterRoutes(e *echo.Echo, rbacMW echo.MiddlewareFunc) {
	e.GET("/api/events", h.HandleEvents, rbacMW)
}

// Subscribe adds a new client channel to the broadcast hub.
func (h *SSEHandler) Subscribe() chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan []byte, 32)
	h.clients[ch] = struct{}{}
	return ch
}

// Unsubscribe removes a client channel from the broadcast hub.
func (h *SSEHandler) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[ch]; ok {
		delete(h.clients, ch)
		close(ch)
	}
}

// Broadcast sends a message to all currently connected SSE clients.
func (h *SSEHandler) Broadcast(event []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			// Client buffer full; drop to avoid blocking other clients
		}
	}
}

// NATSSubscriber defines the minimal interface for subscribing to NATS messages.
type NATSSubscriber interface {
	Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error)
}

// ListenToNATS subscribes to NATS subject and forwards messages to the SSE broadcast hub.
func (h *SSEHandler) ListenToNATS(ctx context.Context, nc NATSSubscriber, subject string) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats subscriber is nil")
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		h.Broadcast(msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe to nats subject %s: %w", subject, err)
	}

	go func() {
		<-ctx.Done()
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}()

	return sub, nil
}

// HandleEvents handles GET /api/events, streaming SSE messages and heartbeat pings.
func (h *SSEHandler) HandleEvents(c echo.Context) error {
	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set(echo.HeaderConnection, "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no")
	res.WriteHeader(http.StatusOK)
	res.Flush()

	clientChan := h.Subscribe()
	defer h.Unsubscribe(clientChan)

	// Send initial connection event
	initialMsg := fmt.Sprintf("event: CONNECTED\ndata: {\"status\":\"connected\",\"timestamp\":%q}\n\n", time.Now().UTC().Format(time.RFC3339))
	if _, err := fmt.Fprint(res, initialMsg); err != nil {
		return nil
	}
	res.Flush()

	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	reqCtx := c.Request().Context()

	for {
		select {
		case <-reqCtx.Done():
			return nil
		case eventData, ok := <-clientChan:
			if !ok {
				return nil
			}
			sseMsg := fmt.Sprintf("event: QUEUE_UPDATED\ndata: %s\n\n", string(eventData))
			if _, err := fmt.Fprint(res, sseMsg); err != nil {
				log.Printf("SSE write error: %v", err)
				return nil
			}
			res.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(res, ":keepalive\n\n"); err != nil {
				return nil
			}
			res.Flush()
		}
	}
}
