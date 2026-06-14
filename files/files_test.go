package files

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

func TestAPI_GetContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/file_123/content" {
			t.Errorf("expected path /files/file_123/content, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"url":        "https://example.com/file",
			"expires_at": "2026-06-13T13:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.GetContent(context.Background(), "file_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.URL != "https://example.com/file" {
		t.Errorf("expected URL %q, got %q", "https://example.com/file", resp.URL)
	}
	if resp.ExpiresAt != "2026-06-13T13:00:00Z" {
		t.Errorf("expected ExpiresAt %q, got %q", "2026-06-13T13:00:00Z", resp.ExpiresAt)
	}
}

func TestAPI_GetContent_InvalidID(t *testing.T) {
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.GetContent(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestAPI_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/files/file_123" {
			t.Errorf("expected path /files/file_123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	if err := api.Delete(context.Background(), "file_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPI_Delete_InvalidID(t *testing.T) {
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	if err := api.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestAPI_List(t *testing.T) {
	t.Parallel()

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/files" {
				t.Errorf("expected path /files, got %s", r.URL.Path)
			}
			if v := r.URL.Query().Get("purpose"); v != "assistants" {
				t.Errorf("expected purpose=assistants, got %q", v)
			}
			if v := r.URL.Query().Get("limit"); v != "10" {
				t.Errorf("expected limit=10, got %q", v)
			}
			if v := r.URL.Query().Get("after_id"); v != "file_1" {
				t.Errorf("expected after_id=file_1, got %q", v)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"file_id":    "file_2",
						"type":       "file",
						"filename":   "doc.pdf",
						"purpose":    "assistants",
						"status":     "processed",
						"size_bytes": 1024,
						"created_at": "2026-06-14T12:00:00Z",
						"updated_at": "2026-06-14T12:00:00Z",
					},
					{
						"file_id":    "file_3",
						"type":       "file",
						"filename":   "notes.txt",
						"purpose":    "assistants",
						"status":     "processed",
						"size_bytes": 512,
						"created_at": "2026-06-14T13:00:00Z",
						"updated_at": "2026-06-14T13:00:00Z",
					},
				},
				"first_id": "file_2",
				"last_id":  "file_3",
				"has_more": false,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		params := &types.ListParams{Limit: 10, AfterID: "file_1"}
		resp, err := api.List(context.Background(), "assistants", params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Data) != 2 {
			t.Fatalf("expected 2 files, got %d", len(resp.Data))
		}
		if resp.Data[0].ID != "file_2" {
			t.Errorf("expected file ID 'file_2', got %s", resp.Data[0].ID)
		}
		if resp.Data[1].ID != "file_3" {
			t.Errorf("expected file ID 'file_3', got %s", resp.Data[1].ID)
		}
		if resp.HasMore {
			t.Error("expected HasMore to be false")
		}
		if resp.FirstID == nil || *resp.FirstID != "file_2" {
			t.Errorf("expected FirstID 'file_2', got %v", resp.FirstID)
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)
		_, err := api.List(context.Background(), "", &types.ListParams{Limit: -1})
		if err == nil {
			t.Error("expected error for invalid Limit")
		}
	})
}

func TestAPI_List_NoFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("expected path /files, got %s", r.URL.Path)
		}
		// No purpose or pagination query params expected.
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data":     []interface{}{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.List(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Data) != 0 {
		t.Fatalf("expected 0 files, got %d", len(resp.Data))
	}
}

func TestAPI_List_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	_, err := api.List(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T", err)
	}
	if !apiErr.IsServerError() {
		t.Error("expected IsServerError")
	}
}

// ---------------------------------------------------------------------------
// Upload
// ---------------------------------------------------------------------------

func TestAPI_Upload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("expected path /files, got %s", r.URL.Path)
		}

		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart/form-data Content-Type, got %s", r.Header.Get("Content-Type"))
		}

		// Parse multipart form (max 10 MB).
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		if v := r.FormValue("purpose"); v != "assistants" {
			t.Errorf("expected purpose field 'assistants', got %q", v)
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file field: %v", err)
		}
		if header.Filename != "hello.txt" {
			t.Errorf("expected filename 'hello.txt', got %q", header.Filename)
		}
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("failed to read uploaded file: %v", err)
		}
		if string(data) != "hello world" {
			t.Errorf("expected file data 'hello world', got %q", string(data))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"file_id":    "file_upload_1",
			"type":       "file",
			"filename":   "hello.txt",
			"purpose":    "assistants",
			"status":     "processed",
			"size_bytes": 11,
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &UploadFileRequest{
		Filename: "hello.txt",
		Data:     []byte("hello world"),
		Purpose:  "assistants",
	}
	resp, err := api.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "file_upload_1" {
		t.Errorf("expected ID 'file_upload_1', got %s", resp.ID)
	}
	if resp.Filename != "hello.txt" {
		t.Errorf("expected Filename 'hello.txt', got %s", resp.Filename)
	}
	if resp.Purpose != "assistants" {
		t.Errorf("expected Purpose 'assistants', got %s", resp.Purpose)
	}
	if resp.Size != 11 {
		t.Errorf("expected Size 11, got %d", resp.Size)
	}
}

func TestAPI_Upload_WithMetadata(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("expected path /files, got %s", r.URL.Path)
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		if v := r.FormValue("metadata[source]"); v != "api" {
			t.Errorf("expected metadata[source] 'api', got %q", v)
		}
		if v := r.FormValue("metadata[version]"); v != "1.0" {
			t.Errorf("expected metadata[version] '1.0', got %q", v)
		}
		if v := r.FormValue("purpose"); v != "assistants" {
			t.Errorf("expected purpose 'assistants', got %q", v)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"file_id":    "file_meta_1",
			"type":       "file",
			"filename":   "data.json",
			"purpose":    "assistants",
			"status":     "processed",
			"size_bytes": 2,
			"metadata": map[string]string{
				"source":  "api",
				"version": "1.0",
			},
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &UploadFileRequest{
		Filename: "data.json",
		Data:     []byte("{}"),
		Purpose:  "assistants",
		Metadata: types.Metadata{
			"source":  "api",
			"version": "1.0",
		},
	}
	resp, err := api.Upload(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "file_meta_1" {
		t.Errorf("expected ID 'file_meta_1', got %s", resp.ID)
	}
}

func TestAPI_Upload_WithIdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/files" {
			t.Errorf("expected path /files, got %s", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "file-upload-key" {
			t.Errorf("expected Idempotency-Key 'file-upload-key', got %q", key)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"file_id":    "file_idem_1",
			"type":       "file",
			"filename":   "doc.txt",
			"purpose":    "assistants",
			"status":     "processed",
			"size_bytes": 5,
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &UploadFileRequest{
		Filename: "doc.txt",
		Data:     []byte("hello"),
		Purpose:  "assistants",
	}
	resp, err := api.Upload(context.Background(), req, "file-upload-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "file_idem_1" {
		t.Errorf("expected ID 'file_idem_1', got %s", resp.ID)
	}
}

func TestAPI_Upload_NilRequest(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Upload(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil UploadFileRequest")
	}
	if err.Error() != "files: UploadFileRequest must not be nil" {
		t.Errorf("expected nil request error, got %q", err.Error())
	}
}

func TestAPI_Upload_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "invalid file format",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &UploadFileRequest{
		Filename: "bad.txt",
		Data:     []byte("data"),
		Purpose:  "assistants",
	}
	_, err := api.Upload(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T", err)
	}
	if !apiErr.IsInvalidRequest() {
		t.Error("expected IsInvalidRequest")
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestAPI_Get(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/file_get_1" {
			t.Errorf("expected path /files/file_get_1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"file_id":    "file_get_1",
			"type":       "file",
			"filename":   "report.pdf",
			"purpose":    "assistants",
			"status":     "processed",
			"size_bytes": 2048,
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Get(context.Background(), "file_get_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "file_get_1" {
		t.Errorf("expected ID 'file_get_1', got %s", resp.ID)
	}
	if resp.Filename != "report.pdf" {
		t.Errorf("expected Filename 'report.pdf', got %s", resp.Filename)
	}
	if resp.Status != "processed" {
		t.Errorf("expected Status 'processed', got %s", resp.Status)
	}
	if resp.Size != 2048 {
		t.Errorf("expected Size 2048, got %d", resp.Size)
	}
}

func TestAPI_Get_InvalidID(t *testing.T) {
	// Client-side validation rejects invalid IDs before any HTTP call.
	// The httptest server MUST never be reached — if it is, validation failed.
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid file ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty ID", id: ""},
		{name: "path traversal", id: "../../etc"},
		{name: "encoded slash", id: "%2fetc"},
		{name: "literal slash", id: "a/b"},
		{name: "encoded backslash", id: "%5cetc"},
		{name: "encoded dot", id: "%2e%2e"},
		{name: "hash character", id: "id#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := api.Get(context.Background(), tt.id)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAPI_Get_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "file not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Get(context.Background(), "file_missing")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound")
	}
}

// ---------------------------------------------------------------------------
// GetContent — path traversal and server error
// ---------------------------------------------------------------------------

func TestAPI_GetContent_PathTraversal(t *testing.T) {
	// Client-side validation rejects invalid IDs before any HTTP call.
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid file ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	tests := []struct {
		name string
		id   string
	}{
		{name: "path traversal", id: "../../etc"},
		{name: "encoded slash", id: "%2fetc"},
		{name: "literal slash", id: "a/b"},
		{name: "encoded backslash", id: "%5cetc"},
		{name: "encoded dot", id: "%2e%2e"},
		{name: "hash character", id: "id#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := api.GetContent(context.Background(), tt.id)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAPI_GetContent_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "file not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.GetContent(context.Background(), "file_missing")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound")
	}
}

// ---------------------------------------------------------------------------
// Delete — path traversal and server error
// ---------------------------------------------------------------------------

func TestAPI_Delete_PathTraversal(t *testing.T) {
	// Client-side validation rejects invalid IDs before any HTTP call.
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid file ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	tests := []struct {
		name string
		id   string
	}{
		{name: "path traversal", id: "../../etc"},
		{name: "encoded slash", id: "%2fetc"},
		{name: "literal slash", id: "a/b"},
		{name: "encoded backslash", id: "%5cetc"},
		{name: "encoded dot", id: "%2e%2e"},
		{name: "hash character", id: "id#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := api.Delete(context.Background(), tt.id)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAPI_Delete_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "file not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	err := api.Delete(context.Background(), "file_missing")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound")
	}
}

func TestUpload_EmptyFilename(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Upload(context.Background(), &UploadFileRequest{Filename: "", Data: []byte("x"), Purpose: "assistants"})
	if err == nil {
		t.Fatal("expected error for empty Filename, got nil")
	}
	if err.Error() != "files: UploadFileRequest.Filename is required" {
		t.Errorf("expected 'files: UploadFileRequest.Filename is required', got %q", err.Error())
	}
}

func TestUpload_NilData(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Upload(context.Background(), &UploadFileRequest{Filename: "x.txt", Data: nil, Purpose: "assistants"})
	if err == nil {
		t.Fatal("expected error for nil Data, got nil")
	}
	if err.Error() != "files: UploadFileRequest.Data is required" {
		t.Errorf("expected 'files: UploadFileRequest.Data is required', got %q", err.Error())
	}
}

func TestUpload_EmptyPurpose(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Upload(context.Background(), &UploadFileRequest{Filename: "x.txt", Data: []byte("x"), Purpose: ""})
	if err == nil {
		t.Fatal("expected error for empty Purpose, got nil")
	}
	if err.Error() != "files: UploadFileRequest.Purpose is required" {
		t.Errorf("expected 'files: UploadFileRequest.Purpose is required', got %q", err.Error())
	}
}
