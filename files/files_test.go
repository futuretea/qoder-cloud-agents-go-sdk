package files

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
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
