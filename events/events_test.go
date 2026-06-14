package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpclient "github.com/futuretea/go-http-client"
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

// rawPathAPI constructs an API that activates the raw-HTTP Stream path:
// WithHTTPClient + WithBaseURL make rawHTTPClient non-nil and baseURL non-empty,
// which is the branch production uses via qoder.Client.Events().
func rawPathAPI(srv *httptest.Server, token string) *API {
	c := httpclient.NewClient(&httpclient.Config{BaseURL: srv.URL})
	return NewAPI(c, WithHTTPClient(srv.Client()), WithBaseURL(srv.URL), WithToken(token))
}

func TestStream_RawPath_HeadersAndAfterIDPresent(t *testing.T) {
	var gotAuth, gotAccept, gotAfterID string
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotAfterID = r.URL.Query().Get("after_id")
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := rawPathAPI(srv, "tok")
	resp, err := api.Stream(context.Background(), "sess_123", "evt_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotAuth != "Bearer tok" {
		t.Errorf("expected Authorization %q, got %q", "Bearer tok", gotAuth)
	}
	if gotAccept != "text/event-stream" {
		t.Errorf("expected Accept %q, got %q", "text/event-stream", gotAccept)
	}
	if gotAfterID != "evt_001" {
		t.Errorf("expected after_id %q, got %q", "evt_001", gotAfterID)
	}
	if gotRawQuery != "after_id=evt_001" {
		t.Errorf("expected raw query %q, got %q", "after_id=evt_001", gotRawQuery)
	}
}

func TestStream_RawPath_NoAfterIDOmitted(t *testing.T) {
	var gotRawQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := rawPathAPI(srv, "tok")
	resp, err := api.Stream(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if gotRawQuery != "" {
		t.Errorf("expected no query params on raw path, got %q", gotRawQuery)
	}
}

// TestStream_RawPath_Non2xxErrorParsing verifies that a non-2xx JSON error
// response on the raw Stream path is surfaced as a typed *qoderhttp.APIError.
//
// The raw path always sets the request header Accept: text/event-stream and then
// runs QoderErrorMiddleware(resp). The middleware skips error parsing only when
// the *response* Content-Type is text/event-stream; an error response carries a
// JSON envelope (application/json), so it must be parsed into an *APIError rather
// than handed back as a raw body for the caller to (mis)parse as SSE.
func TestStream_RawPath_Non2xxErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "not_found_error",
				"message": "session not found",
			},
		})
	}))
	defer srv.Close()

	api := rawPathAPI(srv, "tok")
	resp, err := api.Stream(context.Background(), "sess_123")
	if err == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		t.Fatal("expected a typed error for 404 response on the raw path, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if apiErr.ErrorType != "not_found_error" {
		t.Errorf("expected error type %q, got %q", "not_found_error", apiErr.ErrorType)
	}
}

func TestStream_RawPath_InvalidBaseURL(t *testing.T) {
	c := httpclient.NewClient(&httpclient.Config{})
	api := NewAPI(c,
		WithHTTPClient(&http.Client{}),
		WithBaseURL("://bad-url"),
		WithToken("tok"),
	)

	resp, err := api.Stream(context.Background(), "sess_123")
	if err == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		t.Fatal("expected error for invalid base URL, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "invalid base URL") {
		t.Errorf("expected error to mention %q, got %q", "invalid base URL", got)
	}
}

// TestStream_RawPath_IncrementalNoBuffering proves the raw path does not buffer
// the whole SSE body before returning: the server writes and flushes one event,
// then blocks until the test cancels the request context. Stream must return and
// the first event must be readable while the response is still open.
func TestStream_RawPath_IncrementalNoBuffering(t *testing.T) {
	serverDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("ResponseWriter does not support Flusher")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: first\n\n"))
		flusher.Flush()
		// Block until the client cancels the request; the response is never
		// completed by the server. If Stream buffered the body, it would hang
		// here forever instead of returning after the first flush.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api := rawPathAPI(srv, "tok")
	resp, err := api.Stream(ctx, "sess_123")
	if err != nil {
		t.Fatalf("Stream returned before server finished, but with error: %v", err)
	}

	stream := qoderhttp.NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	evt, err := stream.Next(ctx)
	if err != nil {
		t.Fatalf("expected first event, got error: %v", err)
	}
	if got := string(evt.Data); got != "first" {
		t.Errorf("expected first event data %q, got %q", "first", got)
	}

	// Server is still blocked (response not finished) when we read the event.
	select {
	case <-serverDone:
		t.Fatal("server handler returned before client cancellation; cannot prove non-buffering")
	default:
	}

	// Cancel to release the blocked handler goroutine and avoid leaks.
	cancel()
	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("server handler did not exit after context cancellation")
	}
}
