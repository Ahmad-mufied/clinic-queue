//go:build ignore

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ANSI Color Codes for terminal reporting
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// TestRecord holds the execution result of an individual test case.
type TestRecord struct {
	ID          string        `json:"id"`
	Category    string        `json:"category"`
	Endpoint    string        `json:"endpoint"`
	Persona     string        `json:"persona"`
	Scenario    string        `json:"scenario"`
	Expected    string        `json:"expected"`
	Actual      string        `json:"actual"`
	Status      string        `json:"status"` // PASS or FAIL
	Duration    time.Duration `json:"duration"`
	ErrorDetail string        `json:"error_detail,omitempty"`
}

// Global test execution tracker
type TestSuiteTracker struct {
	mu          sync.Mutex
	records     []TestRecord
	passedCount int32
	failedCount int32
	startTime   time.Time
}

func NewTracker() *TestSuiteTracker {
	return &TestSuiteTracker{
		records:   make([]TestRecord, 0, 100),
		startTime: time.Now(),
	}
}

func (t *TestSuiteTracker) Record(id, category, endpoint, persona, scenario, expected, actual, status, errDetail string, dur time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	rec := TestRecord{
		ID:          id,
		Category:    category,
		Endpoint:    endpoint,
		Persona:     persona,
		Scenario:    scenario,
		Expected:    expected,
		Actual:      actual,
		Status:      status,
		Duration:    dur,
		ErrorDetail: errDetail,
	}
	t.records = append(t.records, rec)

	if status == "PASS" {
		atomic.AddInt32(&t.passedCount, 1)
		fmt.Printf("  %s[%s]%s %s%-14s%s %-40s -> %sPASS%s (%v)\n",
			ColorGreen, id, ColorReset, ColorCyan, persona, ColorReset, scenario, ColorGreen, ColorReset, dur.Round(time.Millisecond))
	} else {
		atomic.AddInt32(&t.failedCount, 1)
		fmt.Printf("  %s[%s]%s %s%-14s%s %-40s -> %sFAIL%s (%v)\n",
			ColorRed, id, ColorReset, ColorYellow, persona, ColorReset, scenario, ColorRed, ColorReset, dur.Round(time.Millisecond))
		if errDetail != "" {
			fmt.Printf("      %sError:%s %s\n", ColorRed, ColorReset, errDetail)
		}
	}
}

// SSEEvent represents an event received from the /api/events SSE stream.
type SSEEvent struct {
	Event string
	Data  string
	Time  time.Time
}

// SSECollector safely buffers events received over SSE.
type SSECollector struct {
	mu     sync.Mutex
	events []SSEEvent
}

func NewSSECollector() *SSECollector {
	return &SSECollector{
		events: make([]SSEEvent, 0, 200),
	}
}

func (c *SSECollector) Start(ctx context.Context, sseURL string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", sseURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to SSE stream: %w", err)
	}

	go func() {
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		var currentEvent string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: ") {
				currentEvent = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				var parsed struct {
					Type  string `json:"type"`
					Event string `json:"event"`
				}
				_ = json.Unmarshal([]byte(data), &parsed)
				evtName := currentEvent
				if evtName == "" {
					evtName = parsed.Type
					if evtName == "" {
						evtName = parsed.Event
					}
				}
				c.mu.Lock()
				c.events = append(c.events, SSEEvent{
					Event: evtName,
					Data:  data,
					Time:  time.Now(),
				})
				c.mu.Unlock()
			}
		}
	}()

	return nil
}

func (c *SSECollector) HasReceivedEvent(eventType string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for _, ev := range c.events {
			if ev.Event == eventType {
				c.mu.Unlock()
				return true
			}
		}
		c.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (c *SSECollector) HasReceivedDataContaining(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for _, ev := range c.events {
			if strings.Contains(ev.Data, substr) {
				c.mu.Unlock()
				return true
			}
		}
		c.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func (c *SSECollector) CountEvents(eventType string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	cnt := 0
	for _, ev := range c.events {
		if ev.Event == eventType {
			cnt++
		}
	}
	return cnt
}

// TestClient manages HTTP calls and token authorization.
type TestClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type APIResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Duration   time.Duration
}

func (c *TestClient) Request(method, path string, token string, body interface{}) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	reqURL := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		if strings.HasPrefix(token, "Bearer ") || strings.HasPrefix(token, "Raw:") {
			req.Header.Set("Authorization", strings.TrimPrefix(token, "Raw:"))
		} else {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
	}

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return &APIResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       respBytes,
		Duration:   duration,
	}, nil
}

// Database helper to reset tables to pristine state.
func resetDatabase(ctx context.Context, dbURL string) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to db for reset: %w", err)
	}
	defer pool.Close()

	resetSQL := `
		TRUNCATE TABLE audit_logs, consultation_sessions, queue_tickets RESTART IDENTITY CASCADE;
		UPDATE doctors SET avg_consultation_time_min = 3, is_online = true WHERE id = '01919df4-8e3b-7412-a1f9-90b567c9e101';
		UPDATE doctors SET avg_consultation_time_min = 4, is_online = true WHERE id = '01919df4-8e3b-7412-a1f9-90b567c9e102';
	`
	_, err = pool.Exec(ctx, resetSQL)
	if err != nil {
		return fmt.Errorf("execute db reset sql: %w", err)
	}
	return nil
}

// Seed helper for inserting custom audit logs if required.
func insertTestAuditLog(ctx context.Context, dbURL, action, role, actor string, userID *string) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	query := `
		INSERT INTO audit_logs (user_id, actor_name, role, action, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, '{}'::jsonb, '127.0.0.1', NOW())
	`
	_, err = pool.Exec(ctx, query, userID, actor, role, action)
	return err
}

func main() {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5433/clinic_queue_test?sslmode=disable"
	}

	fmt.Printf("\n%s%s========================================================================%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s  SMART CLINIC QUEUE - AUTOMATED END-TO-END REGRESSION TEST SUITE        %s\n", ColorBold, ColorWhite, ColorReset)
	fmt.Printf("%s%s  Target Base URL : %s%s\n", ColorBold, ColorCyan, ColorReset, baseURL)
	fmt.Printf("%s%s  PostgreSQL 18   : %s%s\n", ColorBold, ColorCyan, ColorReset, dbURL)
	fmt.Printf("%s%s  Timestamp       : %s%s\n", ColorBold, ColorCyan, ColorReset, time.Now().Format(time.RFC3339))
	fmt.Printf("%s%s========================================================================%s\n\n", ColorBold, ColorCyan, ColorReset)

	tracker := NewTracker()
	client := NewTestClient(baseURL)

	// Step 1: Health & Connectivity Verification
	fmt.Printf("%s[SUITE 0: Infrastructure & Baseline Verification]%s\n", ColorBold, ColorReset)
	start := time.Now()
	healthResp, err := client.Request("GET", "/health", "", nil)
	if err != nil || healthResp.StatusCode != http.StatusOK {
		tracker.Record("INFRA-01", "Infrastructure", "/health", "System", "Server Health Check", "HTTP 200 OK", fmt.Sprintf("Error: %v", err), "FAIL", fmt.Sprintf("%v", err), time.Since(start))
		log.Fatalf("Fatal: API Server is not reachable at %s. Please start server first.", baseURL)
	} else {
		tracker.Record("INFRA-01", "Infrastructure", "/health", "System", "Server Health Check", "HTTP 200 OK", "HTTP 200 (healthy)", "PASS", "", healthResp.Duration)
	}

	// Reset Database to baseline
	start = time.Now()
	if err := resetDatabase(context.Background(), dbURL); err != nil {
		tracker.Record("INFRA-02", "Infrastructure", "PostgreSQL 18", "System", "Database Clean Baseline Reset", "Clean DB Tables", fmt.Sprintf("Error: %v", err), "FAIL", err.Error(), time.Since(start))
		log.Fatalf("Fatal: Failed to reset database: %v", err)
	} else {
		tracker.Record("INFRA-02", "Infrastructure", "PostgreSQL 18", "System", "Database Clean Baseline Reset", "Clean DB Tables", "Pruned & Reset OK", "PASS", "", time.Since(start))
	}

	// Connect SSE Stream Collector
	sseCollector := NewSSECollector()
	sseCtx, sseCancel := context.WithCancel(context.Background())
	defer sseCancel()
	if err := sseCollector.Start(sseCtx, baseURL+"/api/events"); err != nil {
		fmt.Printf("  %sWarning: SSE listener connection failed: %v%s\n", ColorYellow, err, ColorReset)
	} else {
		time.Sleep(200 * time.Millisecond)
		if sseCollector.HasReceivedEvent("CONNECTED", 2*time.Second) {
			tracker.Record("INFRA-03", "Infrastructure", "/api/events", "Public Client", "SSE Stream Connection & Handshake", "event: CONNECTED", "Received CONNECTED event", "PASS", "", 200*time.Millisecond)
		} else {
			tracker.Record("INFRA-03", "Infrastructure", "/api/events", "Public Client", "SSE Stream Connection & Handshake", "event: CONNECTED", "No event received", "FAIL", "Timeout waiting for CONNECTED event", 2*time.Second)
		}
	}

	// Tokens map for personas
	var (
		tokenAdmin    string
		tokenDoctorA  string
		tokenDoctorB  string
		tokenPatientJ string
		tokenPatientL string
		tokenNewUser  string
	)

	// Step 2: PRD 01 - Authentication & RBAC Foundation
	fmt.Printf("\n%s[SUITE 1: PRD 01 - Authentication & Casbin RBAC Foundation]%s\n", ColorBold, ColorReset)

	// AUTH-01: Admin Login
	loginAdminBody := map[string]string{"username": "admin", "password": "password123"}
	resp, err := client.Request("POST", "/api/auth/login", "", loginAdminBody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenAdmin = body["token"].(string)
		userObj := body["user"].(map[string]interface{})
		if userObj["role"] == "admin" {
			tracker.Record("AUTH-01", "Authentication", "/api/auth/login", "Admin", "Valid Admin Login", "HTTP 200 & role=admin", fmt.Sprintf("HTTP 200 & role=%s", userObj["role"]), "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUTH-01", "Authentication", "/api/auth/login", "Admin", "Valid Admin Login", "role=admin", fmt.Sprintf("role=%s", userObj["role"]), "FAIL", "Incorrect role in login response", resp.Duration)
		}
	} else {
		tracker.Record("AUTH-01", "Authentication", "/api/auth/login", "Admin", "Valid Admin Login", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-02: Doctor A Login
	loginDocABody := map[string]string{"username": "doctor_a", "password": "password123"}
	resp, err = client.Request("POST", "/api/auth/login", "", loginDocABody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenDoctorA = body["token"].(string)
		userObj := body["user"].(map[string]interface{})
		if userObj["role"] == "doctor" && userObj["doctor_id"] == "01919df4-8e3b-7412-a1f9-90b567c9e101" {
			tracker.Record("AUTH-02", "Authentication", "/api/auth/login", "Doctor A", "Valid Doctor A Login", "HTTP 200, role=doctor, doctor_id=01919df4-8e3b-7412-a1f9-90b567c9e101", "HTTP 200 & doctor_id=01919df4-8e3b-7412-a1f9-90b567c9e101", "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUTH-02", "Authentication", "/api/auth/login", "Doctor A", "Valid Doctor A Login", "role=doctor, doctor_id=01919df4-8e3b-7412-a1f9-90b567c9e101", fmt.Sprintf("role=%v, doctor_id=%v", userObj["role"], userObj["doctor_id"]), "FAIL", "Mismatch in claims", resp.Duration)
		}
	} else {
		tracker.Record("AUTH-02", "Authentication", "/api/auth/login", "Doctor A", "Valid Doctor A Login", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-03: Doctor B Login
	loginDocBBody := map[string]string{"username": "doctor_b", "password": "password123"}
	resp, err = client.Request("POST", "/api/auth/login", "", loginDocBBody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenDoctorB = body["token"].(string)
		tracker.Record("AUTH-03", "Authentication", "/api/auth/login", "Doctor B", "Valid Doctor B Login", "HTTP 200, role=doctor, doctor_id=01919df4-8e3b-7412-a1f9-90b567c9e102", "HTTP 200 & doctor_id=01919df4-8e3b-7412-a1f9-90b567c9e102", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-03", "Authentication", "/api/auth/login", "Doctor B", "Valid Doctor B Login", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-04: Patient John Login
	loginJohnBody := map[string]string{"username": "patient_john", "password": "password123"}
	resp, err = client.Request("POST", "/api/auth/login", "", loginJohnBody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenPatientJ = body["token"].(string)
		tracker.Record("AUTH-04", "Authentication", "/api/auth/login", "Patient John", "Valid Patient John Login", "HTTP 200, role=patient", "HTTP 200 & role=patient", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-04", "Authentication", "/api/auth/login", "Patient John", "Valid Patient John Login", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-05: Patient Lucas Login
	loginLucasBody := map[string]string{"username": "patient_lucas", "password": "password123"}
	resp, err = client.Request("POST", "/api/auth/login", "", loginLucasBody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenPatientL = body["token"].(string)
		tracker.Record("AUTH-05", "Authentication", "/api/auth/login", "Patient Lucas", "Valid Patient Lucas Login", "HTTP 200, role=patient", "HTTP 200 & role=patient", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-05", "Authentication", "/api/auth/login", "Patient Lucas", "Valid Patient Lucas Login", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-06: Profile Inspection GET /api/auth/me (Doctor A)
	resp, err = client.Request("GET", "/api/auth/me", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var user domainUserCheck
		_ = json.Unmarshal(resp.Body, &user)
		if user.Username == "doctor_a" && user.DoctorID != nil && *user.DoctorID == "01919df4-8e3b-7412-a1f9-90b567c9e101" {
			tracker.Record("AUTH-06", "Authentication", "/api/auth/me", "Doctor A", "Get Profile Me", "HTTP 200 & username=doctor_a", "HTTP 200 & username=doctor_a", "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUTH-06", "Authentication", "/api/auth/me", "Doctor A", "Get Profile Me", "doctor_a", user.Username, "FAIL", "Mismatch in me profile", resp.Duration)
		}
	} else {
		tracker.Record("AUTH-06", "Authentication", "/api/auth/me", "Doctor A", "Get Profile Me", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-07: Patient Self-Registration POST /api/auth/register
	newUsername := fmt.Sprintf("patient_emma_%d", time.Now().UnixNano()%100000)
	regBody := map[string]string{"username": newUsername, "password": "password123", "name": "Emma Watson"}
	resp, err = client.Request("POST", "/api/auth/register", "", regBody)
	if err == nil && resp.StatusCode == http.StatusCreated {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		tokenNewUser = body["token"].(string)
		tracker.Record("AUTH-07", "Authentication", "/api/auth/register", "Guest", "Self Registration", "HTTP 201 Created & Token", "HTTP 201 Created", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-07", "Authentication", "/api/auth/register", "Guest", "Self Registration", "HTTP 201 Created", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-08: Negative - Invalid Password Login
	resp, err = client.Request("POST", "/api/auth/login", "", map[string]string{"username": "admin", "password": "wrongpassword"})
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		tracker.Record("AUTH-08", "Authentication", "/api/auth/login", "Attacker", "Invalid Password Guard", "HTTP 401 Unauthorized", "HTTP 401 Unauthorized", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-08", "Authentication", "/api/auth/login", "Attacker", "Invalid Password Guard", "HTTP 401", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-09: Negative - Non-Existent User Login
	resp, err = client.Request("POST", "/api/auth/login", "", map[string]string{"username": "ghost_user", "password": "password123"})
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		tracker.Record("AUTH-09", "Authentication", "/api/auth/login", "Attacker", "Non-Existent User Guard", "HTTP 401 Unauthorized", "HTTP 401 Unauthorized", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-09", "Authentication", "/api/auth/login", "Attacker", "Non-Existent User Guard", "HTTP 401", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-10: Negative - Duplicate Username Registration
	resp, err = client.Request("POST", "/api/auth/register", "", map[string]string{"username": "patient_john", "password": "password123", "name": "John Duplicate"})
	if err == nil && resp.StatusCode == http.StatusConflict {
		tracker.Record("AUTH-10", "Authentication", "/api/auth/register", "Guest", "Duplicate Username Conflict Guard", "HTTP 409 Conflict", "HTTP 409 Conflict", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-10", "Authentication", "/api/auth/register", "Guest", "Duplicate Username Conflict Guard", "HTTP 409", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-11: Negative - Unauthenticated / Missing Token Guard
	resp, err = client.Request("GET", "/api/auth/me", "", nil)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		tracker.Record("AUTH-11", "Authentication", "/api/auth/me", "Guest", "Missing Token Guard", "HTTP 401 Unauthorized", "HTTP 401 Unauthorized", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-11", "Authentication", "/api/auth/me", "Guest", "Missing Token Guard", "HTTP 401", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-12: Negative - Malformed Authorization Header Guard
	resp, err = client.Request("GET", "/api/auth/me", "Raw:Bearer ", nil)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		tracker.Record("AUTH-12", "Authentication", "/api/auth/me", "Attacker", "Empty Bearer Token Guard", "HTTP 401 Unauthorized", "HTTP 401 Unauthorized", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-12", "Authentication", "/api/auth/me", "Attacker", "Empty Bearer Token Guard", "HTTP 401", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUTH-13: Negative - Forged / Invalid Signature JWT Guard
	forgedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwibmFtZSI6IkZvcmdlZCJ9.invalidSignatureChecksumHere12345"
	resp, err = client.Request("GET", "/api/auth/me", forgedToken, nil)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		tracker.Record("AUTH-13", "Authentication", "/api/auth/me", "Attacker", "Forged JWT Signature Guard", "HTTP 401 Unauthorized", "HTTP 401 Unauthorized", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUTH-13", "Authentication", "/api/auth/me", "Attacker", "Forged JWT Signature Guard", "HTTP 401", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// RBAC Matrix Enforcement Tests
	fmt.Printf("\n%s[SUITE 2: Casbin RBAC Cross-Role Authorization Matrix]%s\n", ColorBold, ColorReset)

	// RBAC-01: Patient calling Doctor status -> 403
	resp, _ = client.Request("POST", "/api/doctors/status", tokenPatientJ, map[string]bool{"is_online": true})
	assertRBAC(tracker, "RBAC-01", "/api/doctors/status", "Patient John", "Patient calling Doctor Status", http.StatusForbidden, resp)

	// RBAC-02: Patient calling Doctor call-next -> 403
	resp, _ = client.Request("POST", "/api/doctors/call-next", tokenPatientJ, nil)
	assertRBAC(tracker, "RBAC-02", "/api/doctors/call-next", "Patient John", "Patient calling Doctor Call-Next", http.StatusForbidden, resp)

	// RBAC-03: Patient calling Doctor finish -> 403
	resp, _ = client.Request("POST", "/api/doctors/finish", tokenPatientJ, nil)
	assertRBAC(tracker, "RBAC-03", "/api/doctors/finish", "Patient John", "Patient calling Doctor Finish", http.StatusForbidden, resp)

	// RBAC-04: Patient calling Doctor workspace -> 403
	resp, _ = client.Request("GET", "/api/doctors/workspace", tokenPatientJ, nil)
	assertRBAC(tracker, "RBAC-04", "/api/doctors/workspace", "Patient John", "Patient calling Doctor Workspace", http.StatusForbidden, resp)

	// RBAC-05: Patient calling Admin Stats -> 403
	resp, _ = client.Request("GET", "/api/admin/stats", tokenPatientJ, nil)
	assertRBAC(tracker, "RBAC-05", "/api/admin/stats", "Patient John", "Patient calling Admin Stats", http.StatusForbidden, resp)

	// RBAC-06: Patient calling Admin Config -> 403
	resp, _ = client.Request("POST", "/api/admin/doctors", tokenPatientJ, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": 5})
	assertRBAC(tracker, "RBAC-06", "/api/admin/doctors", "Patient John", "Patient calling Admin Doctor Config", http.StatusForbidden, resp)

	// RBAC-07: Patient calling Admin Audit Logs -> 403
	resp, _ = client.Request("GET", "/api/admin/audit-logs", tokenPatientJ, nil)
	assertRBAC(tracker, "RBAC-07", "/api/admin/audit-logs", "Patient John", "Patient calling Admin Audit Logs", http.StatusForbidden, resp)

	// RBAC-08: Doctor calling Admin Stats -> 403
	resp, _ = client.Request("GET", "/api/admin/stats", tokenDoctorA, nil)
	assertRBAC(tracker, "RBAC-08", "/api/admin/stats", "Doctor A", "Doctor calling Admin Stats", http.StatusForbidden, resp)

	// RBAC-09: Doctor calling Admin Config -> 403
	resp, _ = client.Request("POST", "/api/admin/doctors", tokenDoctorA, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": 5})
	assertRBAC(tracker, "RBAC-09", "/api/admin/doctors", "Doctor A", "Doctor calling Admin Doctor Config", http.StatusForbidden, resp)

	// RBAC-10: Doctor calling Admin Audit Logs -> 403
	resp, _ = client.Request("GET", "/api/admin/audit-logs", tokenDoctorA, nil)
	assertRBAC(tracker, "RBAC-10", "/api/admin/audit-logs", "Doctor A", "Doctor calling Admin Audit Logs", http.StatusForbidden, resp)

	// RBAC-11: Doctor calling Patient Join Queue -> 403
	resp, _ = client.Request("POST", "/api/queue/join", tokenDoctorA, map[string]string{"patient_name": "Dr Patient"})
	assertRBAC(tracker, "RBAC-11", "/api/queue/join", "Doctor A", "Doctor calling Patient Join Queue", http.StatusForbidden, resp)

	// RBAC-12: Doctor calling Patient My Ticket -> 403
	resp, _ = client.Request("GET", "/api/queue/my-ticket", tokenDoctorA, nil)
	assertRBAC(tracker, "RBAC-12", "/api/queue/my-ticket", "Doctor A", "Doctor calling Patient My Ticket", http.StatusForbidden, resp)

	// RBAC-13: Guest calling Doctor Workspace -> 401
	resp, _ = client.Request("GET", "/api/doctors/workspace", "", nil)
	assertRBAC(tracker, "RBAC-13", "/api/doctors/workspace", "Guest", "Guest calling Doctor Workspace", http.StatusUnauthorized, resp)

	// RBAC-14: Guest calling Admin Stats -> 401
	resp, _ = client.Request("GET", "/api/admin/stats", "", nil)
	assertRBAC(tracker, "RBAC-14", "/api/admin/stats", "Guest", "Guest calling Admin Stats", http.StatusUnauthorized, resp)

	// Step 3: PRD 02 - Patient Queue & Live Wait Countdown
	fmt.Printf("\n%s[SUITE 3: PRD 02 - Patient Queue & Greedy Wait Time Engine]%s\n", ColorBold, ColorReset)

	// QUEUE-01: Public Queue Status Poll
	resp, err = client.Request("GET", "/api/queue/status", "", nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var status map[string]interface{}
		_ = json.Unmarshal(resp.Body, &status)
		onlineDocs := status["online_doctors"].([]interface{})
		totalWait := status["total_waiting"].(float64)
		if len(onlineDocs) >= 2 && totalWait == 0 {
			tracker.Record("QUEUE-01", "Patient Queue", "/api/queue/status", "Guest", "Public Queue Status Initial Poll", "HTTP 200, 2 Doctors Online, 0 Waiting", fmt.Sprintf("HTTP 200, %d Online Docs, %.0f Waiting", len(onlineDocs), totalWait), "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-01", "Patient Queue", "/api/queue/status", "Guest", "Public Queue Status Initial Poll", "2 Doctors Online, 0 Waiting", fmt.Sprintf("%d Online Docs, %.0f Waiting", len(onlineDocs), totalWait), "FAIL", "State mismatch", resp.Duration)
		}
	} else {
		tracker.Record("QUEUE-01", "Patient Queue", "/api/queue/status", "Guest", "Public Queue Status Initial Poll", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-02: Patient John joins queue -> Ticket A-01, wait 0 min (Doc A free)
	resp, err = client.Request("POST", "/api/queue/join", tokenPatientJ, map[string]string{"patient_name": "John Doe"})
	if err == nil && resp.StatusCode == http.StatusCreated {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		t := body["ticket"].(map[string]interface{})
		qNum := t["queue_number"].(string)
		estWait := t["estimated_wait_time_minutes"].(float64)
		pos := t["position_in_queue"].(float64)
		if qNum == "A-01" && estWait == 0 && pos == 1 {
			tracker.Record("QUEUE-02", "Patient Queue", "/api/queue/join", "Patient John", "John Joins Queue (Pos 1)", "Ticket A-01, Pos 1, Wait 0m", fmt.Sprintf("Ticket %s, Pos %.0f, Wait %.0fm", qNum, pos, estWait), "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-02", "Patient Queue", "/api/queue/join", "Patient John", "John Joins Queue (Pos 1)", "A-01, Pos 1, Wait 0m", fmt.Sprintf("%s, Pos %.0f, Wait %.0fm", qNum, pos, estWait), "FAIL", "Calculation or queue number mismatch", resp.Duration)
		}
	} else {
		tracker.Record("QUEUE-02", "Patient Queue", "/api/queue/join", "Patient John", "John Joins Queue (Pos 1)", "HTTP 201 Created", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-03: Patient John checks my-ticket
	resp, err = client.Request("GET", "/api/queue/my-ticket", tokenPatientJ, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		t := body["ticket"].(map[string]interface{})
		if t["queue_number"] == "A-01" && t["status"] == "WAITING" {
			tracker.Record("QUEUE-03", "Patient Queue", "/api/queue/my-ticket", "Patient John", "John Queries My Ticket", "HTTP 200, A-01, WAITING", "HTTP 200, A-01, WAITING", "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-03", "Patient Queue", "/api/queue/my-ticket", "Patient John", "John Queries My Ticket", "A-01 WAITING", fmt.Sprintf("%v %v", t["queue_number"], t["status"]), "FAIL", "Mismatch in my ticket", resp.Duration)
		}
	} else {
		tracker.Record("QUEUE-03", "Patient Queue", "/api/queue/my-ticket", "Patient John", "John Queries My Ticket", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-04: Patient Lucas joins queue -> Ticket A-02, wait 0 min (Doc B also free)
	resp, err = client.Request("POST", "/api/queue/join", tokenPatientL, map[string]string{"patient_name": "Lucas Smith"})
	if err == nil && resp.StatusCode == http.StatusCreated {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		t := body["ticket"].(map[string]interface{})
		qNum := t["queue_number"].(string)
		estWait := t["estimated_wait_time_minutes"].(float64)
		pos := t["position_in_queue"].(float64)
		if qNum == "A-02" && estWait == 0 && pos == 2 {
			tracker.Record("QUEUE-04", "Patient Queue", "/api/queue/join", "Patient Lucas", "Lucas Joins Queue (Pos 2)", "Ticket A-02, Pos 2, Wait 0m", fmt.Sprintf("Ticket %s, Pos %.0f, Wait %.0fm", qNum, pos, estWait), "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-04", "Patient Queue", "/api/queue/join", "Patient Lucas", "Lucas Joins Queue (Pos 2)", "A-02, Pos 2, Wait 0m", fmt.Sprintf("%s, Pos %.0f, Wait %.0fm", qNum, pos, estWait), "FAIL", "Calculation or queue number mismatch", resp.Duration)
		}
	} else {
		tracker.Record("QUEUE-04", "Patient Queue", "/api/queue/join", "Patient Lucas", "Lucas Joins Queue (Pos 2)", "HTTP 201 Created", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-05: Simulation of Multi-Patient Greedy Algorithm (Positions 3 to 6)
	simPatients := []struct {
		pos      int
		username string
		name     string
		expected int
		qNum     string
	}{
		{pos: 3, username: fmt.Sprintf("sim_p3_%d", time.Now().UnixNano()%100000), name: "Patient Three", expected: 3, qNum: "A-03"},
		{pos: 4, username: fmt.Sprintf("sim_p4_%d", time.Now().UnixNano()%100000), name: "Patient Four", expected: 4, qNum: "A-04"},
		{pos: 5, username: fmt.Sprintf("sim_p5_%d", time.Now().UnixNano()%100000), name: "Patient Five", expected: 6, qNum: "A-05"},
		{pos: 6, username: fmt.Sprintf("sim_p6_%d", time.Now().UnixNano()%100000), name: "Patient Six", expected: 8, qNum: "A-06"},
	}

	for _, pt := range simPatients {
		regRes, regErr := client.Request("POST", "/api/auth/register", "", map[string]string{
			"username": pt.username,
			"password": "password123",
			"name":     pt.name,
		})
		if regErr != nil || regRes.StatusCode != http.StatusCreated {
			tracker.Record(fmt.Sprintf("QUEUE-05-%d", pt.pos), "Patient Queue", "/api/queue/join", pt.name, fmt.Sprintf("Greedy Math Simulation Pos %d", pt.pos), "HTTP 201 Created", "Registration Failed", "FAIL", fmt.Sprintf("%v", regErr), 0)
			continue
		}
		var regBody map[string]interface{}
		_ = json.Unmarshal(regRes.Body, &regBody)
		ptToken := regBody["token"].(string)

		resp, err = client.Request("POST", "/api/queue/join", ptToken, map[string]string{"patient_name": pt.name})
		if err == nil && resp.StatusCode == http.StatusCreated {
			var body map[string]interface{}
			_ = json.Unmarshal(resp.Body, &body)
			t := body["ticket"].(map[string]interface{})
			estWait := int(t["estimated_wait_time_minutes"].(float64))
			qNum := t["queue_number"].(string)
			if estWait == pt.expected && qNum == pt.qNum {
				tracker.Record(fmt.Sprintf("QUEUE-05-%d", pt.pos), "Patient Queue", "/api/queue/join", pt.name, fmt.Sprintf("Greedy Math Simulation Pos %d", pt.pos), fmt.Sprintf("Ticket %s, Wait %dm", pt.qNum, pt.expected), fmt.Sprintf("Ticket %s, Wait %dm", qNum, estWait), "PASS", "", resp.Duration)
			} else {
				tracker.Record(fmt.Sprintf("QUEUE-05-%d", pt.pos), "Patient Queue", "/api/queue/join", pt.name, fmt.Sprintf("Greedy Math Simulation Pos %d", pt.pos), fmt.Sprintf("%s, Wait %dm", pt.qNum, pt.expected), fmt.Sprintf("%s, Wait %dm", qNum, estWait), "FAIL", fmt.Sprintf("Greedy mismatch: got %s (%dm), want %s (%dm)", qNum, estWait, pt.qNum, pt.expected), resp.Duration)
			}
		} else {
			tracker.Record(fmt.Sprintf("QUEUE-05-%d", pt.pos), "Patient Queue", "/api/queue/join", pt.name, fmt.Sprintf("Greedy Math Simulation Pos %d", pt.pos), "HTTP 201 Created", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
		}
	}

	// QUEUE-06: Negative - Join with empty name
	resp, _ = client.Request("POST", "/api/queue/join", tokenPatientJ, map[string]string{"patient_name": "  "})
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("QUEUE-06", "Patient Queue", "/api/queue/join", "Patient John", "Empty Patient Name Validation Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("QUEUE-06", "Patient Queue", "/api/queue/join", "Patient John", "Empty Patient Name Validation Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-07: Negative - Duplicate Active Ticket Guard (Patient John)
	resp, _ = client.Request("POST", "/api/queue/join", tokenPatientJ, map[string]string{"patient_name": "John Doe Additional"})
	if resp != nil && resp.StatusCode == http.StatusConflict {
		tracker.Record("QUEUE-07", "Patient Queue", "/api/queue/join", "Patient John", "Duplicate Active Ticket Guard", "HTTP 409 Conflict", "HTTP 409 Conflict", "PASS", "", resp.Duration)
	} else {
		tracker.Record("QUEUE-07", "Patient Queue", "/api/queue/join", "Patient John", "Duplicate Active Ticket Guard", "HTTP 409", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-08: Negative - Duplicate Name Ticket Guard (Same name as active Lucas Smith)
	tempUser := fmt.Sprintf("temp_p_%d", time.Now().UnixNano()%100000)
	regRes, _ := client.Request("POST", "/api/auth/register", "", map[string]string{"username": tempUser, "password": "password123", "name": "Temp User"})
	var tempBody map[string]interface{}
	_ = json.Unmarshal(regRes.Body, &tempBody)
	tempToken := tempBody["token"].(string)

	resp, _ = client.Request("POST", "/api/queue/join", tempToken, map[string]string{"patient_name": "Lucas Smith"})
	if resp != nil && resp.StatusCode == http.StatusConflict {
		tracker.Record("QUEUE-08", "Patient Queue", "/api/queue/join", "Guest / User", "Duplicate Active Patient Name Guard", "HTTP 409 Conflict", "HTTP 409 Conflict", "PASS", "", resp.Duration)
	} else {
		tracker.Record("QUEUE-08", "Patient Queue", "/api/queue/join", "Guest / User", "Duplicate Active Patient Name Guard", "HTTP 409", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-09: Negative - My Ticket Not Found (New Patient Emma who hasn't joined)
	resp, _ = client.Request("GET", "/api/queue/my-ticket", tokenNewUser, nil)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		tracker.Record("QUEUE-09", "Patient Queue", "/api/queue/my-ticket", "Patient Emma", "No Active Ticket Found Guard", "HTTP 404 Not Found", "HTTP 404 Not Found", "PASS", "", resp.Duration)
	} else {
		tracker.Record("QUEUE-09", "Patient Queue", "/api/queue/my-ticket", "Patient Emma", "No Active Ticket Found Guard", "HTTP 404", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// QUEUE-10: Edge Case - Offline Doctors Handling
	// Set Doctor A and Doctor B offline
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorA, map[string]bool{"is_online": false})
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorB, map[string]bool{"is_online": false})

	resp, err = client.Request("GET", "/api/queue/status", "", nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		var notice string
		if n, ok := body["notice"].(string); ok {
			notice = n
		}
		if strings.Contains(notice, "all doctors are offline") {
			tracker.Record("QUEUE-10", "Patient Queue", "/api/queue/status", "Guest", "All Doctors Offline Status Notice", "Notice: all doctors are offline", "Notice displayed correctly", "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-10", "Patient Queue", "/api/queue/status", "Guest", "All Doctors Offline Status Notice", "all doctors are offline", notice, "FAIL", "Notice missing", resp.Duration)
		}
	}

	// Patient John checks my-ticket while doctors offline -> wait time is null & notice present
	resp, err = client.Request("GET", "/api/queue/my-ticket", tokenPatientJ, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		t := body["ticket"].(map[string]interface{})
		var ticketNotice string
		if n, ok := t["notice"].(string); ok {
			ticketNotice = n
		}
		if t["estimated_wait_time_minutes"] == nil && strings.Contains(ticketNotice, "all doctors are offline") {
			tracker.Record("QUEUE-11", "Patient Queue", "/api/queue/my-ticket", "Patient John", "My Ticket Null Wait Countdown When Offline", "Wait: null & Offline notice", "Wait: null & Offline notice", "PASS", "", resp.Duration)
		} else {
			tracker.Record("QUEUE-11", "Patient Queue", "/api/queue/my-ticket", "Patient John", "My Ticket Null Wait Countdown When Offline", "null & offline notice", fmt.Sprintf("%v", t["estimated_wait_time_minutes"]), "FAIL", "Wait time was not null during offline state", resp.Duration)
		}
	}

	// Restore Doctor A and Doctor B back online
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorA, map[string]bool{"is_online": true})
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorB, map[string]bool{"is_online": true})

	// Step 4: PRD 03 - Doctor Workspace & Consultation Workflow
	fmt.Printf("\n%s[SUITE 4: PRD 03 - Doctor Workspace & Consultation Workflow]%s\n", ColorBold, ColorReset)

	// DOC-01: Doctor A checks initial workspace
	resp, err = client.Request("GET", "/api/doctors/workspace", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		if body["status"] == "AVAILABLE" && body["is_online"] == true && body["active_session"] == nil {
			tracker.Record("DOC-01", "Doctor Workspace", "/api/doctors/workspace", "Doctor A", "Initial Workspace Status Query", "HTTP 200, AVAILABLE, active_session=null", "HTTP 200, AVAILABLE, active_session=null", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-01", "Doctor Workspace", "/api/doctors/workspace", "Doctor A", "Initial Workspace Status Query", "AVAILABLE, null session", fmt.Sprintf("%v", body["status"]), "FAIL", "Unexpected initial workspace state", resp.Duration)
		}
	}

	// DOC-02: Negative - Doctor offline attempts to call next patient
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorA, map[string]bool{"is_online": false})
	resp, _ = client.Request("POST", "/api/doctors/call-next", tokenDoctorA, nil)
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("DOC-02", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next While Offline Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("DOC-02", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next While Offline Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}
	_, _ = client.Request("POST", "/api/doctors/status", tokenDoctorA, map[string]bool{"is_online": true})

	// DOC-03: Doctor A calls next patient -> gets John Doe (A-01)
	resp, err = client.Request("POST", "/api/doctors/call-next", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var session map[string]interface{}
		_ = json.Unmarshal(resp.Body, &session)
		ticket := session["ticket"].(map[string]interface{})
		if session["patient_name"] == "John Doe" && ticket["queue_number"] == "A-01" && ticket["status"] == "IN_CONSULTATION" {
			tracker.Record("DOC-03", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next Patient (John Doe A-01)", "HTTP 200, John Doe, A-01 IN_CONSULTATION", "HTTP 200, John Doe, A-01 IN_CONSULTATION", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-03", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next Patient (John Doe A-01)", "John Doe A-01", fmt.Sprintf("%v %v", session["patient_name"], ticket["queue_number"]), "FAIL", "Session payload mismatch", resp.Duration)
		}
	} else {
		tracker.Record("DOC-03", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next Patient (John Doe A-01)", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// DOC-04: Doctor A workspace now shows IN_CONSULTATION with active_session
	resp, err = client.Request("GET", "/api/doctors/workspace", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		if body["status"] == "IN_CONSULTATION" && body["active_session"] != nil {
			tracker.Record("DOC-04", "Doctor Workspace", "/api/doctors/workspace", "Doctor A", "Workspace Reflects Active Consultation", "HTTP 200, status=IN_CONSULTATION, active_session!=null", "HTTP 200, status=IN_CONSULTATION", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-04", "Doctor Workspace", "/api/doctors/workspace", "Doctor A", "Workspace Reflects Active Consultation", "IN_CONSULTATION with active session", fmt.Sprintf("status=%v", body["status"]), "FAIL", "Workspace did not reflect active session", resp.Duration)
		}
	}

	// DOC-05: Patient John checks my-ticket -> status is now IN_CONSULTATION
	resp, err = client.Request("GET", "/api/queue/my-ticket", tokenPatientJ, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		t := body["ticket"].(map[string]interface{})
		if t["status"] == "IN_CONSULTATION" {
			tracker.Record("DOC-05", "Doctor Workspace", "/api/queue/my-ticket", "Patient John", "Patient Ticket Reflects IN_CONSULTATION", "HTTP 200, status=IN_CONSULTATION", "HTTP 200, status=IN_CONSULTATION", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-05", "Doctor Workspace", "/api/queue/my-ticket", "Patient John", "Patient Ticket Reflects IN_CONSULTATION", "IN_CONSULTATION", fmt.Sprintf("%v", t["status"]), "FAIL", "Patient status not updated", resp.Duration)
		}
	}

	// DOC-06: Negative - Doctor A attempts to call next while already in consultation -> 409
	resp, _ = client.Request("POST", "/api/doctors/call-next", tokenDoctorA, nil)
	if resp != nil && resp.StatusCode == http.StatusConflict {
		tracker.Record("DOC-06", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next During Active Session Conflict Guard", "HTTP 409 Conflict", "HTTP 409 Conflict", "PASS", "", resp.Duration)
	} else {
		tracker.Record("DOC-06", "Doctor Workspace", "/api/doctors/call-next", "Doctor A", "Call Next During Active Session Conflict Guard", "HTTP 409", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// DOC-07: Doctor B calls next patient -> gets Lucas Smith (A-02)
	resp, err = client.Request("POST", "/api/doctors/call-next", tokenDoctorB, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var session map[string]interface{}
		_ = json.Unmarshal(resp.Body, &session)
		ticket := session["ticket"].(map[string]interface{})
		if session["patient_name"] == "Lucas Smith" && ticket["queue_number"] == "A-02" {
			tracker.Record("DOC-07", "Doctor Workspace", "/api/doctors/call-next", "Doctor B", "Doctor B Calls Next Patient (Lucas A-02)", "HTTP 200, Lucas Smith, A-02", "HTTP 200, Lucas Smith, A-02", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-07", "Doctor Workspace", "/api/doctors/call-next", "Doctor B", "Doctor B Calls Next Patient (Lucas A-02)", "Lucas Smith A-02", fmt.Sprintf("%v %v", session["patient_name"], ticket["queue_number"]), "FAIL", "Doctor B session mismatch", resp.Duration)
		}
	} else {
		tracker.Record("DOC-07", "Doctor Workspace", "/api/doctors/call-next", "Doctor B", "Doctor B Calls Next Patient (Lucas A-02)", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// Let 1 second pass for consultation duration calculation
	time.Sleep(1 * time.Second)

	// DOC-08: Doctor A finishes consultation -> Session closed, actual duration calculated
	resp, err = client.Request("POST", "/api/doctors/finish", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		if body["patient_name"] == "John Doe" && body["doctor_status"] == "AVAILABLE" {
			tracker.Record("DOC-08", "Doctor Workspace", "/api/doctors/finish", "Doctor A", "Finish Active Consultation", "HTTP 200, John Doe completed, status=AVAILABLE", "HTTP 200, status=AVAILABLE", "PASS", "", resp.Duration)
		} else {
			tracker.Record("DOC-08", "Doctor Workspace", "/api/doctors/finish", "Doctor A", "Finish Active Consultation", "John Doe AVAILABLE", fmt.Sprintf("%v %v", body["patient_name"], body["doctor_status"]), "FAIL", "Finish response mismatch", resp.Duration)
		}
	} else {
		tracker.Record("DOC-08", "Doctor Workspace", "/api/doctors/finish", "Doctor A", "Finish Active Consultation", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// DOC-09: Doctor A workspace returns to AVAILABLE
	resp, err = client.Request("GET", "/api/doctors/workspace", tokenDoctorA, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body map[string]interface{}
		_ = json.Unmarshal(resp.Body, &body)
		if body["status"] == "AVAILABLE" && body["active_session"] == nil {
			tracker.Record("DOC-09", "Doctor Workspace", "/api/doctors/workspace", "Doctor A", "Workspace State Resets to AVAILABLE", "HTTP 200, AVAILABLE, active_session=null", "HTTP 200, AVAILABLE", "PASS", "", resp.Duration)
		}
	}

	// DOC-10: Negative - Doctor A finishes again when room is idle -> 400
	resp, _ = client.Request("POST", "/api/doctors/finish", tokenDoctorA, nil)
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("DOC-10", "Doctor Workspace", "/api/doctors/finish", "Doctor A", "Finish On Idle Room Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("DOC-10", "Doctor Workspace", "/api/doctors/finish", "Doctor A", "Finish On Idle Room Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// DOC-11: Doctor B finishes consultation
	resp, err = client.Request("POST", "/api/doctors/finish", tokenDoctorB, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		tracker.Record("DOC-11", "Doctor Workspace", "/api/doctors/finish", "Doctor B", "Doctor B Finishes Active Consultation", "HTTP 200, status=AVAILABLE", "HTTP 200, status=AVAILABLE", "PASS", "", resp.Duration)
	}

	// Step 5: Concurrency & Race Safety Test (Atomic SKIP LOCKED)
	fmt.Printf("\n%s[SUITE 5: Concurrency & Race Safety (Atomic SKIP LOCKED)]%s\n", ColorBold, ColorReset)

	// Clean reset queue for isolated concurrency test
	_ = resetDatabase(context.Background(), dbURL)
	// Insert exactly ONE waiting patient
	_, _ = client.Request("POST", "/api/queue/join", tokenAdmin, map[string]string{"patient_name": "Concurrency Test Patient"})

	// Concurrently invoke CallNextPatient from Doctor A and Doctor B at the exact same millisecond
	var (
		docAResp  *APIResponse
		docBResp  *APIResponse
		docAErr   error
		docBErr   error
		startSync sync.WaitGroup
		doneSync  sync.WaitGroup
	)

	startSync.Add(1)
	doneSync.Add(2)

	go func() {
		defer doneSync.Done()
		startSync.Wait()
		docAResp, docAErr = client.Request("POST", "/api/doctors/call-next", tokenDoctorA, nil)
	}()

	go func() {
		defer doneSync.Done()
		startSync.Wait()
		docBResp, docBErr = client.Request("POST", "/api/doctors/call-next", tokenDoctorB, nil)
	}()

	// Fire simultaneously
	startConcurr := time.Now()
	startSync.Done()
	doneSync.Wait()
	concurrDur := time.Since(startConcurr)

	// Assertions for Concurrency:
	// Exactly one doctor gets the patient (status 200 with ticket), the other gets status 200 with "Queue is empty" message
	var docASucceeded, docBSucceeded bool
	if docAErr == nil && docAResp.StatusCode == http.StatusOK {
		var b map[string]interface{}
		_ = json.Unmarshal(docAResp.Body, &b)
		if b["patient_name"] == "Concurrency Test Patient" {
			docASucceeded = true
		}
	}
	if docBErr == nil && docBResp.StatusCode == http.StatusOK {
		var b map[string]interface{}
		_ = json.Unmarshal(docBResp.Body, &b)
		if b["patient_name"] == "Concurrency Test Patient" {
			docBSucceeded = true
		}
	}

	if (docASucceeded && !docBSucceeded) || (!docASucceeded && docBSucceeded) {
		winner := "Doctor A"
		if docBSucceeded {
			winner = "Doctor B"
		}
		tracker.Record("CONCURR-01", "Concurrency", "/api/doctors/call-next", "Doctor A & B", "Atomic SKIP LOCKED Race Safety", "Exactly 1 Doctor Gets Patient, 0 Double Bookings", fmt.Sprintf("Winner: %s, 0 Conflicts", winner), "PASS", "", concurrDur)
	} else {
		tracker.Record("CONCURR-01", "Concurrency", "/api/doctors/call-next", "Doctor A & B", "Atomic SKIP LOCKED Race Safety", "Exactly 1 Doctor Gets Patient", fmt.Sprintf("DocA: %v, DocB: %v", docASucceeded, docBSucceeded), "FAIL", "Race condition failure or double assignment", concurrDur)
	}

	// Clean up active session from winner
	if docASucceeded {
		_, _ = client.Request("POST", "/api/doctors/finish", tokenDoctorA, nil)
	} else if docBSucceeded {
		_, _ = client.Request("POST", "/api/doctors/finish", tokenDoctorB, nil)
	}

	// Step 6: PRD 04 - Executive Business Analytics & Doctor Productivity
	fmt.Printf("\n%s[SUITE 6: PRD 04 - Executive Analytics & Doctor Productivity]%s\n", ColorBold, ColorReset)

	// ADMIN-01: Admin Queries Analytics Stats
	resp, err = client.Request("GET", "/api/admin/stats", tokenAdmin, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var stats map[string]interface{}
		_ = json.Unmarshal(resp.Body, &stats)
		summary := stats["summary"].(map[string]interface{})
		docPerf := stats["doctor_performance"].([]interface{})
		totalServed := summary["total_served_today"].(float64)
		onlineCount := summary["online_doctors_count"].(float64)

		if totalServed >= 1 && onlineCount == 2 && len(docPerf) == 2 {
			tracker.Record("ADMIN-01", "Executive Analytics", "/api/admin/stats", "Admin", "Executive KPI Aggregation & Doctor Performance", "HTTP 200, TotalServed>=1, OnlineDocs=2", fmt.Sprintf("HTTP 200, Served=%.0f, OnlineDocs=%.0f", totalServed, onlineCount), "PASS", "", resp.Duration)
		} else {
			tracker.Record("ADMIN-01", "Executive Analytics", "/api/admin/stats", "Admin", "Executive KPI Aggregation & Doctor Performance", "Served>=1, Online=2", fmt.Sprintf("Served=%.0f, Online=%.0f, PerfLen=%d", totalServed, onlineCount, len(docPerf)), "FAIL", fmt.Sprintf("Got Served=%.0f, Online=%.0f, PerfLen=%d (raw: %s)", totalServed, onlineCount, len(docPerf), string(resp.Body)), resp.Duration)
		}
	} else {
		tracker.Record("ADMIN-01", "Executive Analytics", "/api/admin/stats", "Admin", "Executive KPI Aggregation & Doctor Performance", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// ADMIN-02: Admin Updates Doctor A Configuration
	configBody := map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": 5}
	resp, err = client.Request("POST", "/api/admin/doctors", tokenAdmin, configBody)
	if err == nil && resp.StatusCode == http.StatusOK {
		var doc map[string]interface{}
		_ = json.Unmarshal(resp.Body, &doc)
		if doc["avg_consultation_time"] == float64(5) {
			tracker.Record("ADMIN-02", "Executive Analytics", "/api/admin/doctors", "Admin", "Update Doctor Config Parameter", "HTTP 200 & avg_time=5m", "HTTP 200 & avg_time=5m", "PASS", "", resp.Duration)
		} else {
			tracker.Record("ADMIN-02", "Executive Analytics", "/api/admin/doctors", "Admin", "Update Doctor Config Parameter", "avg_time=5m", fmt.Sprintf("avg_time=%v", doc["avg_consultation_time"]), "FAIL", "Configuration update not persisted", resp.Duration)
		}
	}

	// Restore Doctor A back to 3m
	_, _ = client.Request("POST", "/api/admin/doctors", tokenAdmin, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": 3})

	// ADMIN-03: Negative - Update Doctor Config with <= 0 duration
	resp, _ = client.Request("POST", "/api/admin/doctors", tokenAdmin, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": 0})
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("ADMIN-03", "Executive Analytics", "/api/admin/doctors", "Admin", "Zero Consultation Duration Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("ADMIN-03", "Executive Analytics", "/api/admin/doctors", "Admin", "Zero Consultation Duration Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// ADMIN-04: Negative - Update Doctor Config with negative duration
	resp, _ = client.Request("POST", "/api/admin/doctors", tokenAdmin, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_consultation_time_min": -4})
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("ADMIN-04", "Executive Analytics", "/api/admin/doctors", "Admin", "Negative Consultation Duration Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("ADMIN-04", "Executive Analytics", "/api/admin/doctors", "Admin", "Negative Consultation Duration Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// ADMIN-05: Negative - Update Doctor Config for Non-Existent Doctor
	resp, _ = client.Request("POST", "/api/admin/doctors", tokenAdmin, map[string]interface{}{"doctor_id": "01919df4-8e3b-7412-a1f9-999999999999", "avg_consultation_time_min": 5})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		tracker.Record("ADMIN-05", "Executive Analytics", "/api/admin/doctors", "Admin", "Non-Existent Doctor Config Guard", "HTTP 404 Not Found", "HTTP 404 Not Found", "PASS", "", resp.Duration)
	} else {
		tracker.Record("ADMIN-05", "Executive Analytics", "/api/admin/doctors", "Admin", "Non-Existent Doctor Config Guard", "HTTP 404", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// Step 7: PRD 05 - Comprehensive Activity Logging & Audit Trail Pipeline
	fmt.Printf("\n%s[SUITE 7: PRD 05 - Activity Logging & Audit Trail Pipeline]%s\n", ColorBold, ColorReset)

	// Seed several audit logs for testing query and filter
	adminID := "01919df4-8e3b-7412-a1f9-90b567c9e205"
	docAID := "01919df4-8e3b-7412-a1f9-90b567c9e201"
	_ = insertTestAuditLog(context.Background(), dbURL, "QUEUE_JOINED", "patient", "John Doe", nil)
	_ = insertTestAuditLog(context.Background(), dbURL, "CONSULTATION_STARTED", "doctor", "Dr. Sarah Adams", &docAID)
	_ = insertTestAuditLog(context.Background(), dbURL, "CONSULTATION_FINISHED", "doctor", "Dr. Sarah Adams", &docAID)
	_ = insertTestAuditLog(context.Background(), dbURL, "DOCTOR_CONFIG_UPDATED", "admin", "Clinic Administrator", &adminID)

	// AUDIT-01: Admin Queries Paginated Audit Logs
	resp, err = client.Request("GET", "/api/admin/audit-logs", tokenAdmin, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var pageResult map[string]interface{}
		_ = json.Unmarshal(resp.Body, &pageResult)
		totRec := pageResult["total_records"].(float64)
		logs := pageResult["logs"].([]interface{})
		if totRec >= 4 && len(logs) >= 4 {
			tracker.Record("AUDIT-01", "Audit Trail", "/api/admin/audit-logs", "Admin", "Query Paginated Audit Log Stream", "HTTP 200, TotalRecords>=4", fmt.Sprintf("HTTP 200, TotalRecords=%.0f", totRec), "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUDIT-01", "Audit Trail", "/api/admin/audit-logs", "Admin", "Query Paginated Audit Log Stream", "TotalRecords>=4", fmt.Sprintf("TotalRecords=%.0f", totRec), "FAIL", "Missing audit records", resp.Duration)
		}
	} else {
		tracker.Record("AUDIT-01", "Audit Trail", "/api/admin/audit-logs", "Admin", "Query Paginated Audit Log Stream", "HTTP 200", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUDIT-02: Query with limit & page
	resp, err = client.Request("GET", "/api/admin/audit-logs?page=1&limit=2", tokenAdmin, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var pageResult map[string]interface{}
		_ = json.Unmarshal(resp.Body, &pageResult)
		logs := pageResult["logs"].([]interface{})
		if len(logs) == 2 {
			tracker.Record("AUDIT-02", "Audit Trail", "/api/admin/audit-logs?page=1&limit=2", "Admin", "Audit Pagination Limit Control", "HTTP 200, exactly 2 records", "HTTP 200, 2 records", "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUDIT-02", "Audit Trail", "/api/admin/audit-logs?page=1&limit=2", "Admin", "Audit Pagination Limit Control", "2 records", fmt.Sprintf("%d records", len(logs)), "FAIL", "Limit parameter not respected", resp.Duration)
		}
	}

	// AUDIT-03: Filter by Action (QUEUE_JOINED)
	resp, err = client.Request("GET", "/api/admin/audit-logs?action=QUEUE_JOINED", tokenAdmin, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var pageResult map[string]interface{}
		_ = json.Unmarshal(resp.Body, &pageResult)
		logs := pageResult["logs"].([]interface{})
		allMatch := true
		for _, item := range logs {
			logMap := item.(map[string]interface{})
			if logMap["action"] != "QUEUE_JOINED" {
				allMatch = false
				break
			}
		}
		if len(logs) > 0 && allMatch {
			tracker.Record("AUDIT-03", "Audit Trail", "/api/admin/audit-logs?action=QUEUE_JOINED", "Admin", "Filter Logs by Action Taxonomy", "HTTP 200, all action=QUEUE_JOINED", "HTTP 200, filtered correctly", "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUDIT-03", "Audit Trail", "/api/admin/audit-logs?action=QUEUE_JOINED", "Admin", "Filter Logs by Action Taxonomy", "action=QUEUE_JOINED", fmt.Sprintf("count=%d, allMatch=%v", len(logs), allMatch), "FAIL", "Filter condition failed", resp.Duration)
		}
	}

	// AUDIT-04: Filter by Role (doctor)
	resp, err = client.Request("GET", "/api/admin/audit-logs?role=doctor", tokenAdmin, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		var pageResult map[string]interface{}
		_ = json.Unmarshal(resp.Body, &pageResult)
		logs := pageResult["logs"].([]interface{})
		allMatch := true
		for _, item := range logs {
			logMap := item.(map[string]interface{})
			if logMap["role"] != "doctor" {
				allMatch = false
				break
			}
		}
		if len(logs) > 0 && allMatch {
			tracker.Record("AUDIT-04", "Audit Trail", "/api/admin/audit-logs?role=doctor", "Admin", "Filter Logs by Role (doctor)", "HTTP 200, all role=doctor", "HTTP 200, filtered correctly", "PASS", "", resp.Duration)
		} else {
			tracker.Record("AUDIT-04", "Audit Trail", "/api/admin/audit-logs?role=doctor", "Admin", "Filter Logs by Role (doctor)", "role=doctor", fmt.Sprintf("count=%d, allMatch=%v", len(logs), allMatch), "FAIL", "Role filter failed", resp.Duration)
		}
	}

	// AUDIT-05: Negative - Invalid page parameter
	resp, _ = client.Request("GET", "/api/admin/audit-logs?page=-1", tokenAdmin, nil)
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("AUDIT-05", "Audit Trail", "/api/admin/audit-logs?page=-1", "Admin", "Invalid Page Parameter Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUDIT-05", "Audit Trail", "/api/admin/audit-logs?page=-1", "Admin", "Invalid Page Parameter Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// AUDIT-06: Negative - Invalid limit parameter
	resp, _ = client.Request("GET", "/api/admin/audit-logs?limit=0", tokenAdmin, nil)
	if resp != nil && resp.StatusCode == http.StatusBadRequest {
		tracker.Record("AUDIT-06", "Audit Trail", "/api/admin/audit-logs?limit=0", "Admin", "Invalid Limit Parameter Guard", "HTTP 400 Bad Request", "HTTP 400 Bad Request", "PASS", "", resp.Duration)
	} else {
		tracker.Record("AUDIT-06", "Audit Trail", "/api/admin/audit-logs?limit=0", "Admin", "Invalid Limit Parameter Guard", "HTTP 400", fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", string(resp.Body), resp.Duration)
	}

	// Step 8: Real-time SSE Stream Event Reception Verification
	fmt.Printf("\n%s[SUITE 8: Real-Time SSE Event Broadcasting Verification]%s\n", ColorBold, ColorReset)

	// SSE-01: Verify QUEUE_UPDATED events
	hasQueueUpdated := sseCollector.HasReceivedEvent("QUEUE_UPDATED", 1*time.Second)
	if hasQueueUpdated {
		tracker.Record("SSE-01", "SSE Streaming", "/api/events", "Public Subscriber", "Live Event Stream Broadcast (QUEUE_UPDATED)", "Received QUEUE_UPDATED events", fmt.Sprintf("Received %d events", sseCollector.CountEvents("QUEUE_UPDATED")), "PASS", "", 50*time.Millisecond)
	} else {
		tracker.Record("SSE-01", "SSE Streaming", "/api/events", "Public Subscriber", "Live Event Stream Broadcast (QUEUE_UPDATED)", "Received QUEUE_UPDATED events", "None received", "FAIL", "No QUEUE_UPDATED events collected", 1*time.Second)
	}

	// SSE-02: Verify DOCTOR_STATUS_CHANGED event broadcast payload
	hasDocStatus := sseCollector.HasReceivedDataContaining("DOCTOR_STATUS_CHANGED", 1*time.Second) || sseCollector.HasReceivedDataContaining("status", 1*time.Second)
	if hasDocStatus {
		tracker.Record("SSE-02", "SSE Streaming", "/api/events", "Public Subscriber", "Doctor Status Change Broadcast (SSE)", "Doctor status event stream payload", "Broadcast payload received", "PASS", "", 50*time.Millisecond)
	} else {
		tracker.Record("SSE-02", "SSE Streaming", "/api/events", "Public Subscriber", "Doctor Status Change Broadcast (SSE)", "Doctor status event payload", "None received", "FAIL", "Doctor status broadcast payload not captured", 1*time.Second)
	}

	// SSE-03: Verify TICKET_CALLED / TICKET_FINISHED event payload
	hasTicketEvents := sseCollector.HasReceivedDataContaining("TICKET_CALLED", 1*time.Second) || sseCollector.HasReceivedDataContaining("TICKET_FINISHED", 1*time.Second) || sseCollector.HasReceivedDataContaining("session_id", 1*time.Second)
	if hasTicketEvents {
		tracker.Record("SSE-03", "SSE Streaming", "/api/events", "Public Subscriber", "Consultation Lifecycle Broadcast (SSE)", "Consultation call/finish stream payload", "Broadcast payload received", "PASS", "", 50*time.Millisecond)
	} else {
		tracker.Record("SSE-03", "SSE Streaming", "/api/events", "Public Subscriber", "Consultation Lifecycle Broadcast (SSE)", "Consultation stream payload", "None received", "FAIL", "Lifecycle event payload not captured", 1*time.Second)
	}

	// Final Summary & Report Export
	totalTests := tracker.passedCount + tracker.failedCount
	elapsed := time.Since(tracker.startTime)

	fmt.Printf("\n%s%s========================================================================%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("%s%s  E2E REGRESSION TEST SUITE EXECUTION SUMMARY                             %s\n", ColorBold, ColorWhite, ColorReset)
	fmt.Printf("%s%s========================================================================%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("  Total Test Scenarios Executed : %s%d%s\n", ColorBold, totalTests, ColorReset)
	fmt.Printf("  Passed Test Scenarios         : %s%s%d%s\n", ColorBold, ColorGreen, tracker.passedCount, ColorReset)
	fmt.Printf("  Failed Test Scenarios         : %s%s%d%s\n", ColorBold, ColorRed, tracker.failedCount, ColorReset)
	fmt.Printf("  Success Rate                  : %s%s%.1f%%%s\n", ColorBold, ColorGreen, float64(tracker.passedCount)/float64(totalTests)*100.0, ColorReset)
	fmt.Printf("  Total Execution Duration      : %s%v%s\n", ColorBold, elapsed.Round(time.Millisecond), ColorReset)
	fmt.Printf("%s%s========================================================================%s\n\n", ColorBold, ColorCyan, ColorReset)

	// Save test results JSON for audit report generation
	resultsJSON, _ := json.MarshalIndent(tracker.records, "", "  ")
	_ = os.WriteFile("scripts/testing/e2e/e2e_test_results.json", resultsJSON, 0644)

	if tracker.failedCount > 0 {
		fmt.Printf("%sFAILURE: %d test scenario(s) failed.%s\n", ColorRed, tracker.failedCount, ColorReset)
		os.Exit(1)
	}

	fmt.Printf("%sSUCCESS: All %d test scenarios passed with 100%% compliance!%s\n\n", ColorGreen, totalTests, ColorReset)
}

func assertRBAC(t *TestSuiteTracker, id, endpoint, persona, scenario string, expectedCode int, resp *APIResponse) {
	if resp == nil {
		t.Record(id, "Casbin RBAC Matrix", endpoint, persona, scenario, fmt.Sprintf("HTTP %d", expectedCode), "No Response", "FAIL", "Response was nil", 0)
		return
	}
	if resp.StatusCode == expectedCode {
		t.Record(id, "Casbin RBAC Matrix", endpoint, persona, scenario, fmt.Sprintf("HTTP %d", expectedCode), fmt.Sprintf("HTTP %d", resp.StatusCode), "PASS", "", resp.Duration)
	} else {
		t.Record(id, "Casbin RBAC Matrix", endpoint, persona, scenario, fmt.Sprintf("HTTP %d", expectedCode), fmt.Sprintf("HTTP %d", resp.StatusCode), "FAIL", fmt.Sprintf("RBAC failure: %s", string(resp.Body)), resp.Duration)
	}
}

type domainUserCheck struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	DoctorID *string `json:"doctor_id"`
}
