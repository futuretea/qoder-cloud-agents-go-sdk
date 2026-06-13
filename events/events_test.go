package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

func TestNewUserMessage(t *testing.T) {
	evt := NewUserMessage("hello")
	if evt.Type != EventTypeUserMessage {
		t.Errorf("expected type %q, got %q", EventTypeUserMessage, evt.Type)
	}
	if evt.Content != "hello" {
		t.Errorf("expected content %q, got %q", "hello", evt.Content)
	}
}

func TestNewInterruptEvent(t *testing.T) {
	evt := NewInterruptEvent()
	if evt.Type != EventTypeUserInterrupt {
		t.Errorf("expected type %q, got %q", EventTypeUserInterrupt, evt.Type)
	}
}

func TestNewToolConfirmationEvent(t *testing.T) {
	evt, err := NewToolConfirmationEvent("tool_123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != EventTypeUserToolConfirmation {
		t.Errorf("expected type %q, got %q", EventTypeUserToolConfirmation, evt.Type)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(evt.Content), &payload); err != nil {
		t.Fatalf("invalid JSON content: %v", err)
	}
	if payload["tool_use_id"] != "tool_123" {
		t.Errorf("expected tool_use_id tool_123, got %v", payload["tool_use_id"])
	}
	if payload["approved"] != true {
		t.Errorf("expected approved true, got %v", payload["approved"])
	}
}

func TestNewToolConfirmationEvent_ApprovedFalse(t *testing.T) {
	evt, err := NewToolConfirmationEvent("tool_123", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(evt.Content), &payload); err != nil {
		t.Fatalf("invalid JSON content: %v", err)
	}
	if payload["approved"] != false {
		t.Errorf("expected approved false, got %v", payload["approved"])
	}
}

func TestNewCustomToolResultEvent(t *testing.T) {
	result := map[string]any{"status": "ok", "value": 42}
	evt, err := NewCustomToolResultEvent("tool_123", result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Type != EventTypeUserCustomToolResult {
		t.Errorf("expected type %q, got %q", EventTypeUserCustomToolResult, evt.Type)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(evt.Content), &payload); err != nil {
		t.Fatalf("invalid JSON content: %v", err)
	}
	if payload["tool_use_id"] != "tool_123" {
		t.Errorf("expected tool_use_id tool_123, got %v", payload["tool_use_id"])
	}

	decodedResult, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %T", payload["result"])
	}
	if decodedResult["status"] != "ok" {
		t.Errorf("expected status ok, got %v", decodedResult["status"])
	}
}

func TestNewCustomToolResultEvent_MarshalError(t *testing.T) {
	// Channels cannot be JSON-marshaled, so this should return an error.
	result := map[string]any{"bad": make(chan int)}
	_, err := NewCustomToolResultEvent("tool_123", result)
	if err == nil {
		t.Fatal("expected error for unmarshalable result, got nil")
	}
}

func TestStream_ResumptionUsesAfterIDQueryParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/sess_123/events/stream" {
			t.Errorf("expected path %q, got %q", "/sessions/sess_123/events/stream", r.URL.Path)
		}
		if r.URL.Query().Get("after_id") != "evt_001" {
			t.Errorf("expected after_id=evt_001, got %q", r.URL.Query().Get("after_id"))
		}
		if r.Header.Get("Last-Event-ID") != "" {
			t.Errorf("expected no Last-Event-ID header, got %q", r.Header.Get("Last-Event-ID"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Stream(context.Background(), "sess_123", "evt_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestStream_NoLastEventIDOmitsAfterID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}
		if r.Header.Get("Last-Event-ID") != "" {
			t.Errorf("expected no Last-Event-ID header, got %q", r.Header.Get("Last-Event-ID"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Stream(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}
