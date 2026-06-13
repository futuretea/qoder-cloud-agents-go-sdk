package environments

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

func TestCreateIncludesConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/environments" {
			t.Errorf("expected path /environments, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		config, ok := payload["config"]
		if !ok {
			t.Fatal("expected request body to contain 'config' field")
		}

		configMap, ok := config.(map[string]interface{})
		if !ok {
			t.Fatalf("expected config to be an object, got %T", config)
		}
		if configMap["type"] != "sandbox" {
			t.Errorf("expected config.type 'sandbox', got %v", configMap["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Environment{
			ID:     "env_123",
			Name:   "prod",
			Status: "active",
			Config: EnvConfig{Type: "sandbox"},
		})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	req := NewCreateRequest("prod", EnvConfig{Type: "sandbox"})
	env, err := api.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.ID != "env_123" {
		t.Errorf("expected id 'env_123', got '%s'", env.ID)
	}
}

func TestCreateWithConfigOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		config := payload["config"].(map[string]interface{})
		if config["type"] != "vm" {
			t.Errorf("expected config.type 'vm', got %v", config["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Environment{ID: "env_456"})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	req := NewCreateRequest("prod", EnvConfig{Type: "sandbox"}).WithConfig(EnvConfig{Type: "vm"})
	if _, err := api.Create(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/environments/env_123" {
			t.Errorf("expected path /environments/env_123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	if err := api.Delete(context.Background(), "env_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInvalidID(t *testing.T) {
	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	if err := api.Delete(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
}
