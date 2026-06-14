package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

func newTestAPI(baseURL string) *API {
	return NewAPI(httpclient.NewClient(&httpclient.Config{BaseURL: baseURL}))
}

// invalidSessionIDs covers the validation rules in qoderhttp.ValidateID.
var invalidSessionIDs = []struct {
	name string
	id   string
}{
	{name: "empty", id: ""},
	{name: "slash", id: "a/b"},
	{name: "backslash", id: `a\b`},
	{name: "dot-dot", id: "a..b"},
	{name: "url-encoded-slash", id: "a%2fb"},
	{name: "hash", id: "a#frag"},
	{name: "question-mark", id: "a?x=1"},
	{name: "space", id: "a b"},
	{name: "tab", id: "a\tb"},
	{name: "newline", id: "a\nb"},
	{name: "control-char", id: "a\x00b"},
}

func TestAPI_Send_InvalidSessionID(t *testing.T) {
	api := newTestAPI("https://api.qoder.com")
	req := NewSendRequest(NewUserMessage("hello"))

	for _, tt := range invalidSessionIDs {
		t.Run(tt.name, func(t *testing.T) {
			err := api.Send(context.Background(), tt.id, req)
			if err == nil {
				t.Errorf("expected error for invalid session ID %q, got nil", tt.id)
			}
		})
	}
}

func TestAPI_Send_NilRequest(t *testing.T) {
	api := newTestAPI("https://api.qoder.com")

	err := api.Send(context.Background(), "session_123", nil)
	if err == nil {
		t.Fatal("expected error for nil SendEventRequest")
	}
	if err.Error() != "events: SendEventRequest must not be nil" {
		t.Errorf("expected nil request error, got %q", err.Error())
	}
}

func TestAPI_Send_NilEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request body contains an empty/null events array.
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body["events"] != nil {
			t.Errorf("expected events field to be null for nil Events slice, got %v", body["events"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	api := newTestAPI(srv.URL)
	// Non-nil request with nil Events slice.
	req := &SendEventRequest{Events: nil}
	err := api.Send(context.Background(), "session_123", req)
	if err != nil {
		t.Fatalf("unexpected error for nil Events: %v", err)
	}
}

func TestAPI_Send_ValidSessionID(t *testing.T) {
	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	api := newTestAPI(srv.URL)
	err := api.Send(context.Background(), "session_123", NewSendRequest(NewUserMessage("hello")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestPath != "/sessions/session_123/events" {
		t.Errorf("expected path %q, got %q", "/sessions/session_123/events", requestPath)
	}
}

func TestAPI_Send_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]string{
				"type":    "api_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	err := api.Send(context.Background(), "session_123", NewSendRequest(NewUserMessage("hello")))
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsServerError() {
		t.Errorf("expected server error, got status %d", apiErr.StatusCode)
	}
}

func TestAPI_SendMessage(t *testing.T) {
	var requestPath string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	api := newTestAPI(srv.URL)
	err := api.SendMessage(context.Background(), "session_123", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestPath != "/sessions/session_123/events" {
		t.Errorf("expected path %q, got %q", "/sessions/session_123/events", requestPath)
	}

	events, ok := body["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected 1 event, got %v", body["events"])
	}
	evt, ok := events[0].(map[string]any)
	if !ok {
		t.Fatalf("expected event object, got %T", events[0])
	}
	if evt["type"] != EventTypeUserMessage {
		t.Errorf("expected type %q, got %q", EventTypeUserMessage, evt["type"])
	}
	if evt["content"] != "hello" {
		t.Errorf("expected content %q, got %q", "hello", evt["content"])
	}
}

func TestAPI_List_InvalidSessionID(t *testing.T) {
	api := newTestAPI("https://api.qoder.com")
	for _, tt := range invalidSessionIDs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := api.List(context.Background(), tt.id, nil)
			if err == nil {
				t.Errorf("expected error for invalid session ID %q, got nil", tt.id)
			}
		})
	}
}

func TestAPI_List_ValidSessionID(t *testing.T) {
	t.Run("with pagination", func(t *testing.T) {
		var requestPath string
		var query string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath = r.URL.Path
			query = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}, "has_more": false})
		}))
		defer srv.Close()

		api := newTestAPI(srv.URL)
		_, err := api.List(context.Background(), "session_123", &types.ListParams{Limit: 10, AfterID: "evt_001"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requestPath != "/sessions/session_123/events" {
			t.Errorf("expected path %q, got %q", "/sessions/session_123/events", requestPath)
		}
		if query != "after_id=evt_001&limit=10" && query != "limit=10&after_id=evt_001" {
			t.Errorf("unexpected query: %s", query)
		}
	})

	t.Run("invalid params returns error", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		api := newTestAPI("http://localhost")
		_, err := api.List(context.Background(), "session_123", &types.ListParams{Limit: -1})
		if err == nil {
			t.Error("expected error for invalid Limit")
		}
	})
}

func TestAPI_List_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]string{
				"type":    "api_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.List(context.Background(), "session_123", nil)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsServerError() {
		t.Errorf("expected server error, got status %d", apiErr.StatusCode)
	}
}

func TestAPI_Stream_InvalidSessionID(t *testing.T) {
	api := newTestAPI("https://api.qoder.com")
	for _, tt := range invalidSessionIDs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := api.Stream(context.Background(), tt.id)
			if err == nil {
				t.Errorf("expected error for invalid session ID %q, got nil", tt.id)
			}
		})
	}
}

func TestAPI_Stream_ValidSessionID(t *testing.T) {
	var requestPath string
	var acceptHeader string
	var afterID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		acceptHeader = r.Header.Get("Accept")
		afterID = r.URL.Query().Get("after_id")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := newTestAPI(srv.URL)
	resp, err := api.Stream(context.Background(), "session_123", "evt_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if requestPath != "/sessions/session_123/events/stream" {
		t.Errorf("expected path %q, got %q", "/sessions/session_123/events/stream", requestPath)
	}
	if acceptHeader != "text/event-stream" {
		t.Errorf("expected Accept header %q, got %q", "text/event-stream", acceptHeader)
	}
	if afterID != "evt_001" {
		t.Errorf("expected after_id query param %q, got %q", "evt_001", afterID)
	}
}

func TestAPI_Stream_LastEventID_EmptySkipped(t *testing.T) {
	var afterID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		afterID = r.URL.Query().Get("after_id")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := newTestAPI(srv.URL)
	resp, err := api.Stream(context.Background(), "session_123", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if afterID != "" {
		t.Errorf("expected no after_id query param for empty value, got %q", afterID)
	}
}

func TestNewAPI(t *testing.T) {
	api := NewAPI(httpclient.NewClient(&httpclient.Config{}))
	if api == nil {
		t.Fatal("expected API to be non-nil")
	}
}

func TestStreamEvent_TypeAlias(t *testing.T) {
	// StreamEvent is a type alias for qoderhttp.SSEEvent.
	evt := StreamEvent(qoderhttp.SSEEvent{ID: "evt_001"})
	if evt.ID != "evt_001" {
		t.Errorf("expected ID evt_001, got %s", evt.ID)
	}
}

// doerWrapper is an httpclient.Doer that is NOT a *http.Client. It is used to
// verify that UpdateStreamConfig falls back to the default streaming client
// (keeping the raw path usable) when the supplied Doer is not a *http.Client.
type doerWrapper struct {
	inner *http.Client
}

func (d *doerWrapper) Do(req *http.Request) (*http.Response, error) {
	return d.inner.Do(req)
}

func TestUpdateStreamConfig_HTTPClientUsesRawPath(t *testing.T) {
	var gotAuth, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	// Start with a buffered-only API (no baseURL on the events side), then push
	// a real *http.Client config in, activating the raw path.
	api := NewAPI(httpclient.NewClient(&httpclient.Config{BaseURL: srv.URL}))
	api.UpdateStreamConfig(srv.URL, "tok", srv.Client())

	resp, err := api.Stream(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Raw-path-specific behavior: the request carries the bearer token and the
	// SSE Accept header set by the raw branch.
	if gotAuth != "Bearer tok" {
		t.Errorf("expected Authorization %q, got %q", "Bearer tok", gotAuth)
	}
	if gotAccept != "text/event-stream" {
		t.Errorf("expected Accept %q, got %q", "text/event-stream", gotAccept)
	}
}

func TestUpdateStreamConfig_NonHTTPClientFallsBackToDefault(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := NewAPI(httpclient.NewClient(&httpclient.Config{BaseURL: srv.URL}))

	// Pass a Doer that is not a *http.Client. UpdateStreamConfig must reset
	// rawHTTPClient to the shared default (NOT nil), keeping the raw path usable
	// when baseURL is set. The default client reaches the httptest server over a
	// real local TCP connection.
	var doer httpclient.Doer = &doerWrapper{inner: srv.Client()}
	api.UpdateStreamConfig(srv.URL, "tok", doer)

	resp, err := api.Stream(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Reaching the server with the SSE Accept header confirms the raw path ran
	// via the default client rather than the buffered fallback or a nil client.
	if gotAccept != "text/event-stream" {
		t.Errorf("expected Accept %q, got %q", "text/event-stream", gotAccept)
	}
}

func TestUpdateStreamConfig_NilDoerFallsBackToDefault(t *testing.T) {
	var reached bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	api := NewAPI(httpclient.NewClient(&httpclient.Config{BaseURL: srv.URL}))
	// A nil Doer must not leave rawHTTPClient nil; it falls back to the default,
	// so Stream stays on the raw path.
	api.UpdateStreamConfig(srv.URL, "tok", nil)

	resp, err := api.Stream(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if !reached {
		t.Error("expected raw path to reach the server after nil Doer fallback")
	}
}
