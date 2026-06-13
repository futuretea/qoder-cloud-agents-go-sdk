package qoderhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
		json.NewEncoder(w).Encode(map[string]string{"id": "agent_123", "name": "test-agent"})
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
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if !apiErr.IsServerError() {
		t.Error("expected IsServerError to be true")
	}
}

func TestQoderErrorMiddlewareEmptyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
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
	if apiErr.ErrorType != "unknown_error" {
		t.Errorf("expected ErrorType 'unknown_error' for empty message, got %q", apiErr.ErrorType)
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
		defer file.Close()

		content, _ := io.ReadAll(file)
		if string(content) != "test-file-content" {
			t.Errorf("unexpected file content: %s", string(content))
		}

		if r.FormValue("purpose") != "user_upload" {
			t.Errorf("unexpected purpose: %s", r.FormValue("purpose"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "file_123", "filename": "test.txt"})
	}))
	defer srv.Close()

	c := NewClient(&Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})

	var result map[string]string
	err := PostMultipart(context.Background(), c, "/files", "file", "test.txt", []byte("test-file-content"),
		map[string]string{"purpose": "user_upload"}, nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "file_123" {
		t.Errorf("expected id 'file_123', got '%s'", result["id"])
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer stream.Close()

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
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
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
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
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
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []string{}, "has_more": false})
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
			json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
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
			json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
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
			json.NewEncoder(w).Encode(map[string]string{"id": "res_123"})
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer stream.Close()

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
		defer pw.Close()
		_, _ = pw.Write([]byte("id: evt_001\nevent: agent.message\ndata: hello\n\n"))
		<-done // block until test signals completion
	}()

	resp := &http.Response{Body: pr}
	stream := NewSSEStream(resp)
	defer stream.Close()

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

	close(done) // signal goroutine to exit cleanly
}
