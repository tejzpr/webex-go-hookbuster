package forwarder

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
)

func TestForwardSendsPostRequest(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Parse host and port from test server URL
	parts := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	event := config.WebhookEvent{
		Resource:  "messages",
		Event:     "created",
		Data:      map[string]interface{}{"id": "123", "verb": "post"},
		Timestamp: 1000,
	}

	err := Forward(host, port, event)
	if err != nil {
		t.Fatalf("Forward() returned error: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}

	// Verify the body is valid JSON matching our event
	var decoded config.WebhookEvent
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("failed to unmarshal received body: %v", err)
	}

	if decoded.Resource != "messages" {
		t.Errorf("resource = %q, want %q", decoded.Resource, "messages")
	}
	if decoded.Event != "created" {
		t.Errorf("event = %q, want %q", decoded.Event, "created")
	}
	if decoded.Timestamp != 1000 {
		t.Errorf("timestamp = %d, want %d", decoded.Timestamp, 1000)
	}
}

func TestForwardReturnsErrorOnConnectionFailure(t *testing.T) {
	event := config.WebhookEvent{
		Resource: "rooms",
		Event:    "created",
		Data:     map[string]interface{}{},
	}

	// Port 1 should be unreachable
	err := Forward("127.0.0.1", 1, event)
	if err == nil {
		t.Error("Forward() should return error for unreachable target")
	}
}

func TestForwardSendsPrettyPrintedJSON(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	event := config.WebhookEvent{
		Resource:  "messages",
		Event:     "created",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: 999,
	}

	err := Forward(host, port, event)
	if err != nil {
		t.Fatalf("Forward() returned error: %v", err)
	}

	// Pretty-printed JSON uses 4-space indentation
	body := string(receivedBody)
	if !strings.Contains(body, "    ") {
		t.Error("expected pretty-printed JSON with 4-space indent")
	}
}

// ── ForwardToURL tests ──────────────────────────────────────────────────

func TestForwardToURL_SendsPostRequest(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	event := config.WebhookEvent{
		Resource:  "messages",
		Event:     "created",
		Data:      map[string]interface{}{"id": "456", "verb": "post"},
		Timestamp: 2000,
	}

	err := ForwardToURL(server.URL, event)
	if err != nil {
		t.Fatalf("ForwardToURL() returned error: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}

	var decoded config.WebhookEvent
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("failed to unmarshal received body: %v", err)
	}

	if decoded.Resource != "messages" {
		t.Errorf("resource = %q, want %q", decoded.Resource, "messages")
	}
	if decoded.Timestamp != 2000 {
		t.Errorf("timestamp = %d, want %d", decoded.Timestamp, 2000)
	}
}

func TestForwardToURL_ReturnsErrorOnConnectionFailure(t *testing.T) {
	event := config.WebhookEvent{
		Resource: "rooms",
		Event:    "created",
		Data:     map[string]interface{}{},
	}

	err := ForwardToURL("http://127.0.0.1:1", event)
	if err == nil {
		t.Error("ForwardToURL() should return error for unreachable target")
	}
}

func TestForwardToURL_PrettyPrintedJSON(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	event := config.WebhookEvent{
		Resource:  "memberships",
		Event:     "created",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: 3000,
	}

	err := ForwardToURL(server.URL, event)
	if err != nil {
		t.Fatalf("ForwardToURL() returned error: %v", err)
	}

	body := string(receivedBody)
	if !strings.Contains(body, "    ") {
		t.Error("expected pretty-printed JSON with 4-space indent")
	}
}

func TestForward_DelegatesToForwardToURL(t *testing.T) {
	// Verify that Forward builds the URL and delegates correctly
	var receivedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	event := config.WebhookEvent{
		Resource: "rooms",
		Event:    "updated",
		Data:     map[string]interface{}{},
	}

	err := Forward(host, port, event)
	if err != nil {
		t.Fatalf("Forward() returned error: %v", err)
	}

	expectedHost := host + ":" + parts[1]
	if receivedURL != expectedHost {
		t.Errorf("received host = %q, want %q", receivedURL, expectedHost)
	}
}

func TestForwardSetsContentLength(t *testing.T) {
	var receivedContentLength string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentLength = r.Header.Get("Content-Length")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parts := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	event := config.WebhookEvent{
		Resource: "rooms",
		Event:    "updated",
		Data:     map[string]interface{}{},
	}

	err := Forward(host, port, event)
	if err != nil {
		t.Fatalf("Forward() returned error: %v", err)
	}

	if receivedContentLength == "" {
		t.Error("Content-Length header should be set")
	}

	cl, err := strconv.Atoi(receivedContentLength)
	if err != nil || cl <= 0 {
		t.Errorf("Content-Length should be a positive number, got %q", receivedContentLength)
	}
}
