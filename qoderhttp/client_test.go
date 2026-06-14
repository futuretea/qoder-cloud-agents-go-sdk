package qoderhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

func TestNewClient(t *testing.T) {
	c := NewClient(&Config{
		BaseURL: "https://api.qoder.com/api/v1/cloud",
		Token:   "test-token",
	})
	if c == nil {
		t.Fatal("expected client to be non-nil")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(nil)
	if c == nil {
		t.Fatal("expected client to be non-nil")
	}
}

func TestClientAuthInjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization: Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "agent_123", "name": "test-agent"})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	type req struct {
		Name string `json:"name"`
	}
	type resp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	var result resp
	err := c.POST("/agents").WithJSON(req{Name: "test-agent"}).Do(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "agent_123" {
		t.Errorf("expected id 'agent_123', got '%s'", result.ID)
	}
}

func TestClientQueryParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("after_id") != "agent_001" {
			t.Errorf("expected after_id=agent_001, got %s", r.URL.Query().Get("after_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	var result map[string]interface{}
	err := c.GET("/agents").
		WithQuery("limit", "10").
		WithQuery("after_id", "agent_001").
		Do(&result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "not_found_error",
				"message": "agent not found",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	var result map[string]interface{}
	err := c.GET("/agents/agent_nonexistent").Do(&result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
	if apiErr.Message != "agent not found" {
		t.Errorf("expected 'agent not found', got '%s'", apiErr.Message)
	}
}

func TestClientConflictError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "conflict_error",
				"message": "version mismatch",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	err := c.PUT("/agents/agent_001").WithJSON(map[string]int{"version": 1}).Do(nil)
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsConflict() {
		t.Error("expected IsConflict to be true")
	}
}

func TestQoderErrorMiddlewareNonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html><body>Internal Server Error</body></html>"))
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	err := c.GET("/resource").Do(nil)
	if err == nil {
		t.Fatal("expected error for non-JSON 500 response")
	}
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.ErrorType != "unknown_error" {
		t.Errorf("expected ErrorType 'unknown_error' for non-JSON body, got %q", apiErr.ErrorType)
	}
	if apiErr.Message == "" {
		t.Error("expected non-empty error message for non-JSON 500 response")
	}
	if !apiErr.IsServerError() {
		t.Error("expected IsServerError to be true")
	}
}

func TestQoderErrorMiddlewareEmptyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "", // empty error type
				"message": "", // empty message
			},
		})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	err := c.GET("/resource").Do(nil)
	if err == nil {
		t.Fatal("expected error for response with empty error message")
	}
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Message == "" {
		t.Error("expected non-empty error message for response with empty error message")
	}
}

func TestPostMultipart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			t.Error("expected Content-Type header")
		}
		if r.Header.Get("Idempotency-Key") != "key-123" {
			t.Errorf("expected Idempotency-Key header %q, got %q", "key-123", r.Header.Get("Idempotency-Key"))
		}

		// Parse the multipart form
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("expected file field: %v", err)
			return
		}
		defer func() { _ = file.Close() }()

		content, _ := io.ReadAll(file)
		if string(content) != "test-file-content" {
			t.Errorf("unexpected file content: %s", string(content))
		}

		if r.FormValue("purpose") != "user_upload" {
			t.Errorf("unexpected purpose: %s", r.FormValue("purpose"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "file_123", "filename": "test.txt"})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	var result map[string]string
	err := PostMultipart(context.Background(), c, "/files", "file", "test.txt", []byte("test-file-content"),
		map[string]string{"purpose": "user_upload"}, map[string]string{"Idempotency-Key": "key-123"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "file_123" {
		t.Errorf("expected id 'file_123', got '%s'", result["id"])
	}
}

func TestPostMultipart_FieldOrdering(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
			return
		}

		// Fields should appear in sorted alphabetical order in the raw body.
		bodyStr := string(body)
		idxA := strings.Index(bodyStr, `name="a"`)
		idxB := strings.Index(bodyStr, `name="b"`)
		idxC := strings.Index(bodyStr, `name="c"`)
		if idxA == -1 || idxB == -1 || idxC == -1 {
			t.Errorf("expected all fields in body, got a=%d b=%d c=%d", idxA, idxB, idxC)
		}
		if idxA >= idxB || idxB >= idxC {
			t.Errorf("expected fields sorted a < b < c in body, got a=%d b=%d c=%d", idxA, idxB, idxC)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "file_123"})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	// Insert fields in reverse alphabetical order to verify sorting.
	extraFields := map[string]string{
		"c": "3",
		"b": "2",
		"a": "1",
	}

	var result map[string]string
	err := PostMultipart(context.Background(), c, "/files", "file", "test.txt", []byte("content"),
		extraFields, nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIErrorMethods(t *testing.T) {
	err := &APIError{StatusCode: 401, ErrorType: "authentication_error", Message: "invalid token"}
	if !err.IsUnauthorized() {
		t.Error("expected IsUnauthorized")
	}
	if err.IsNotFound() {
		t.Error("expected not IsNotFound")
	}
	if err.IsConflict() {
		t.Error("expected not IsConflict")
	}

	svrErr := &APIError{StatusCode: 500, ErrorType: "api_error", Message: "internal"}
	if !svrErr.IsServerError() {
		t.Error("expected IsServerError")
	}
}

func TestIsAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "not found"}
	_, ok := IsAPIError(err)
	if !ok {
		t.Error("expected IsAPIError to return true for *APIError")
	}

	_, ok = IsAPIError(nil)
	if ok {
		t.Error("expected IsAPIError to return false for nil")
	}

	t.Run("unwraps wrapped error via fmt.Errorf(%w)", func(t *testing.T) {
		original := &APIError{StatusCode: 404, ErrorType: "not_found_error", Message: "not found"}
		wrapped := fmt.Errorf("response middleware error: %w", original)
		apiErr, ok := IsAPIError(wrapped)
		if !ok {
			t.Fatal("expected IsAPIError to unwrap the error chain")
		}
		if apiErr != original {
			t.Error("expected IsAPIError to return the original *APIError pointer")
		}
		if !apiErr.IsNotFound() {
			t.Error("expected IsNotFound to be true for unwrapped error")
		}
	})
}

func TestSSEStream(t *testing.T) {
	sseData := "id: evt_001\nevent: agent.message\ndata: {\"type\":\"agent.message\",\"content\":\"hello\"}\n\nid: evt_002\nevent: agent.thinking\ndata: {\"type\":\"agent.thinking\",\"content\":\"thinking...\"}\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseData))
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	resp, err := c.GET("/stream").WithHeader("Accept", "text/event-stream").DoWithResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream := NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	evt1, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading event 1: %v", err)
	}
	if evt1.ID != "evt_001" || evt1.Event != "agent.message" {
		t.Errorf("unexpected event: id=%s event=%s", evt1.ID, evt1.Event)
	}

	evt2, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading event 2: %v", err)
	}
	if evt2.ID != "evt_002" || evt2.Event != "agent.thinking" {
		t.Errorf("unexpected event: id=%s event=%s", evt2.ID, evt2.Event)
	}

	_, err = stream.Next(t.Context())
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestApplyListParams(t *testing.T) {
	t.Run("nil params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params for nil ListParams, got %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		req := ApplyListParams(c.GET("/test"), nil)
		var result map[string]interface{}
		if err := req.Do(&result); err != nil {
			t.Fatalf("nil params should not cause error: %v", err)
		}
	})

	t.Run("with params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("limit") != "20" {
				t.Errorf("expected limit=20, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("after_id") != "cursor_abc" {
				t.Errorf("expected after_id=cursor_abc, got %s", r.URL.Query().Get("after_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		params := &types.ListParams{Limit: 20, AfterID: "cursor_abc"}
		req := ApplyListParams(c.GET("/test"), params)
		var result map[string]interface{}
		if err := req.Do(&result); err != nil {
			t.Fatalf("non-nil params should not cause error: %v", err)
		}
	})

	t.Run("empty params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params for empty ListParams, got %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		req := ApplyListParams(c.GET("/test"), &types.ListParams{})
		var result map[string]interface{}
		if err := req.Do(&result); err != nil {
			t.Fatalf("empty params should not cause error: %v", err)
		}
	})
}

func TestApplyIdempotencyKey(t *testing.T) {
	t.Run("with key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Idempotency-Key") != "my-idempotency-key" {
				t.Errorf("expected Idempotency-Key header, got %q", r.Header.Get("Idempotency-Key"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		var result map[string]string
		if err := ApplyIdempotencyKey(c.POST("/test"), "my-idempotency-key").Do(&result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("without key", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Idempotency-Key") != "" {
				t.Errorf("expected no Idempotency-Key header, got %q", r.Header.Get("Idempotency-Key"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		var result map[string]string
		if err := ApplyIdempotencyKey(c.POST("/test")).Do(&result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty key string", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Idempotency-Key") != "" {
				t.Errorf("expected no Idempotency-Key header for empty key, got %q", r.Header.Get("Idempotency-Key"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
		}))
		defer srv.Close()

		c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		var result map[string]string
		if err := ApplyIdempotencyKey(c.POST("/test"), "").Do(&result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAPIErrorIsPermissionError(t *testing.T) {
	err := &APIError{StatusCode: 403, ErrorType: "permission_error", Message: "access denied"}
	if !err.IsPermissionError() {
		t.Error("expected IsPermissionError for 403")
	}

	err2 := &APIError{StatusCode: 404, ErrorType: "not_found_error", Message: "not found"}
	if err2.IsPermissionError() {
		t.Error("expected not IsPermissionError for 404")
	}
}

func TestAPIErrorIsInvalidRequest(t *testing.T) {
	err := &APIError{StatusCode: 400, ErrorType: "invalid_request_error", Message: "bad input"}
	if !err.IsInvalidRequest() {
		t.Error("expected IsInvalidRequest for invalid_request_error type")
	}

	err2 := &APIError{StatusCode: 400, ErrorType: "other_error", Message: "bad input"}
	if err2.IsInvalidRequest() {
		t.Error("expected not IsInvalidRequest for other_error type")
	}
}

func TestAPIErrorWithParam(t *testing.T) {
	t.Run("with param", func(t *testing.T) {
		err := &APIError{StatusCode: 422, ErrorType: "validation_error", Message: "invalid field", Param: "name"}
		errStr := err.Error()
		expected := `qoder API error 422 (validation_error): invalid field (param: name)`
		if errStr != expected {
			t.Errorf("unexpected error string: %s", errStr)
		}
	})

	t.Run("without param", func(t *testing.T) {
		err := &APIError{StatusCode: 422, ErrorType: "validation_error", Message: "invalid field"}
		errStr := err.Error()
		expected := `qoder API error 422 (validation_error): invalid field`
		if errStr != expected {
			t.Errorf("unexpected error string: %s", errStr)
		}
	})
}

func TestSSEStreamCloseIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: hello\n\n"))
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	resp, err := c.GET("/stream").DoWithResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream := NewSSEStream(resp)
	// Read one event to ensure the stream is active
	_, err = stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading event: %v", err)
	}

	// Close twice — must not panic
	if err := stream.Close(); err != nil {
		t.Errorf("first close should succeed: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Errorf("second close should succeed (idempotent): %v", err)
	}
}

func TestSSEStreamMultiLineData(t *testing.T) {
	sseData := "id: evt_001\nevent: agent.message\ndata: line1\ndata: line2\ndata: line3\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseData))
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	resp, err := c.GET("/stream").DoWithResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream := NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	evt, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(evt.Data) != "line1\nline2\nline3" {
		t.Errorf("expected multiline data, got '%s'", string(evt.Data))
	}
}

func TestSSEStreamContextCancellation(t *testing.T) {
	// Use io.Pipe to simulate a stream that blocks after the first event.
	// httptest cannot be used here because its handlers run synchronously,
	// so a blocking handler prevents the HTTP response from being delivered.
	pr, pw := io.Pipe()
	done := make(chan struct{})

	go func() {
		defer func() { _ = pw.Close() }()
		_, _ = pw.Write([]byte("id: evt_001\nevent: agent.message\ndata: hello\n\n"))
		<-done // block until test signals completion
	}()
	defer close(done) // ensure goroutine exits even if test fails

	resp := &http.Response{Body: pr}
	stream := NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	// Read the first event (should succeed).
	_, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading first event: %v", err)
	}

	// Cancel context — Next should return context.Canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = stream.Next(ctx)
	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestSSEStreamCommentsAndUnknownFields(t *testing.T) {
	sseData := ": this is a comment\ndata: valid-data\n\n:id: ignored-field\nevent: agent.message\ndata: hello\nretry: 3000\n\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sseData))
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	resp, err := c.GET("/stream").WithHeader("Accept", "text/event-stream").DoWithResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stream := NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	evt1, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading event 1: %v", err)
	}
	if string(evt1.Data) != "valid-data" {
		t.Errorf("expected data 'valid-data', got '%s'", string(evt1.Data))
	}
	if evt1.Event != "" {
		t.Errorf("expected empty event for comment-prefixed line, got '%s'", evt1.Event)
	}

	evt2, err := stream.Next(t.Context())
	if err != nil {
		t.Fatalf("unexpected error reading event 2: %v", err)
	}
	if evt2.Event != "agent.message" {
		t.Errorf("expected event 'agent.message', got '%s'", evt2.Event)
	}
	// Unknown field "retry" should be silently ignored
}

func TestPostMultipartErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "invalid purpose",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	var result map[string]string
	err := PostMultipart(context.Background(), c, "/files", "file", "test.txt", []byte("content"),
		map[string]string{"purpose": "invalid"}, nil, &result)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if !apiErr.IsInvalidRequest() {
		t.Error("expected IsInvalidRequest to be true")
	}
	if apiErr.Message != "invalid purpose" {
		t.Errorf("expected 'invalid purpose', got '%s'", apiErr.Message)
	}
}

func TestIsAPIErrorNonWrapping(t *testing.T) {
	original := &APIError{StatusCode: 404, ErrorType: "not_found_error", Message: "not found"}
	wrapped := fmt.Errorf("response middleware error: %v", original)
	_, ok := IsAPIError(wrapped)
	if ok {
		t.Error("expected IsAPIError to return false for non-wrapping error format")
	}
}

func TestSSEStreamNextAfterClose(t *testing.T) {
	pr, pw := io.Pipe()
	done := make(chan struct{})

	go func() {
		defer func() { _ = pw.Close() }()
		<-done // block until test signals, so no data is buffered
	}()
	defer close(done) // ensure goroutine exits even if test fails

	resp := &http.Response{Body: pr}
	stream := NewSSEStream(resp)

	if err := stream.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	_, err := stream.Next(t.Context())
	if err == nil {
		t.Error("expected error calling Next() after Close()")
	}
}

// delayedReader is an io.ReadCloser that blocks Read until the test signals
// the reader to proceed. It is used to exercise the SSE cache path where a
// parse completes concurrently with context cancellation.
type delayedReader struct {
	data      []byte
	readReady chan struct{}
	readyOnce sync.Once
	unblock   chan struct{}
}

func (r *delayedReader) Read(p []byte) (int, error) {
	r.readyOnce.Do(func() { close(r.readReady) })
	<-r.unblock
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func (r *delayedReader) Close() error {
	select {
	case <-r.unblock:
	default:
		close(r.unblock)
	}
	return nil
}

func TestSSEStream_CachesEventOnContextCancellation(t *testing.T) {
	evtData := []byte("id: evt_001\nevent: agent.message\ndata: hello\n\n")
	dr := &delayedReader{
		data:      evtData,
		readReady: make(chan struct{}),
		unblock:   make(chan struct{}),
	}

	resp := &http.Response{Body: dr}
	stream := NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-dr.readReady
		cancel()
	}()

	// This call blocks until Close() is triggered by cancellation; the parse
	// goroutine then unblocks, reads the event, and caches it.
	_, err := stream.Next(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// The next call should return the cached event without reading again.
	evt, err := stream.Next(context.Background())
	if err != nil {
		t.Fatalf("unexpected error reading cached event: %v", err)
	}
	if evt.ID != "evt_001" {
		t.Errorf("expected cached event ID evt_001, got %s", evt.ID)
	}
}

func TestQoderErrorMiddleware_NilBody(t *testing.T) {
	err := QoderErrorMiddleware(&http.Response{StatusCode: http.StatusInternalServerError, Body: nil})
	if err == nil {
		t.Fatal("expected error for nil body")
	}
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Message != "empty response body" {
		t.Errorf("expected message %q, got %q", "empty response body", apiErr.Message)
	}
}

func TestQoderErrorMiddleware_TruncatedFallbackBody(t *testing.T) {
	longMsg := strings.Repeat("x", maxErrorBodySize+100)
	body := fmt.Sprintf(`{"type":"error","error":{"type":"api_error","message":%q}}`, longMsg)
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	err := QoderErrorMiddleware(resp)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := IsAPIError(err)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !strings.HasSuffix(apiErr.Message, " (truncated)") {
		t.Errorf("expected truncated message suffix, got %q", apiErr.Message)
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid ID", id: "agent_123", wantErr: false},
		{name: "valid UUID", id: "550e8400-e29b-41d4-a716-446655440000", wantErr: false},
		{name: "empty string", id: "", wantErr: true},
		{name: "contains slash", id: "a/b", wantErr: true},
		{name: "contains backslash", id: `a\b`, wantErr: true},
		{name: "contains dot-dot", id: "a..b", wantErr: true},
		{name: "contains double dot-dot", id: "....", wantErr: true},
		{name: "starts with slash", id: "/etc/passwd", wantErr: true},
		{name: "traversal attempt", id: "../../etc/passwd", wantErr: true},
		{name: "URL-encoded slash", id: "a%2fb", wantErr: true},
		{name: "URL-encoded slash uppercase", id: "a%2Fb", wantErr: true},
		{name: "URL-encoded backslash", id: `a%5cb`, wantErr: true},
		{name: "URL-encoded dot-dot", id: "a%2e%2eb", wantErr: true},
		{name: "URL-encoded mixed case dot-dot", id: "%2E%2E", wantErr: true},
		{name: "URL-encoded null byte", id: "a%00b", wantErr: true},
		{name: "contains hash", id: "agent_123#frag", wantErr: true},
		{name: "contains question mark", id: "agent_123?foo=bar", wantErr: true},
		{name: "contains space", id: "agent 123", wantErr: true},
		{name: "contains tab", id: "agent\t123", wantErr: true},
		{name: "contains newline", id: "agent\n123", wantErr: true},
		{name: "contains control char", id: "agent\x00123", wantErr: true},
		{name: "percent sign in normal ID", id: "agent_50%_discount", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID(%q) error = %v, wantErr = %v", tt.id, err, tt.wantErr)
			}
		})
	}
}
