package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

func TestModel_List(t *testing.T) {
	priceFactor := 1.5
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("expected path /models, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":             "model_001",
					"type":           "llm",
					"display_name":   "Claude Opus",
					"source":         "anthropic",
					"is_enabled":     true,
					"is_new":         false,
					"price_factor":   1.5,
					"efforts":        []string{"low", "medium", "high"},
					"default_effort": "medium",
				},
				{
					"id":             "model_002",
					"type":           "llm",
					"display_name":   "DeepSeek V4",
					"source":         "deepseek",
					"is_enabled":     true,
					"is_new":         true,
					"price_factor":   nil,
					"efforts":        nil,
					"default_effort": "",
				},
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	models, err := api.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	// First model: all fields populated.
	m1 := models[0]
	if m1.ID != "model_001" {
		t.Errorf("expected ID 'model_001', got '%s'", m1.ID)
	}
	if m1.Type != "llm" {
		t.Errorf("expected Type 'llm', got '%s'", m1.Type)
	}
	if m1.DisplayName != "Claude Opus" {
		t.Errorf("expected DisplayName 'Claude Opus', got '%s'", m1.DisplayName)
	}
	if m1.Source != "anthropic" {
		t.Errorf("expected Source 'anthropic', got '%s'", m1.Source)
	}
	if !m1.IsEnabled {
		t.Error("expected IsEnabled to be true")
	}
	if m1.IsNew {
		t.Error("expected IsNew to be false")
	}
	if m1.PriceFactor == nil {
		t.Error("expected PriceFactor to be non-nil")
	} else if *m1.PriceFactor != priceFactor {
		t.Errorf("expected PriceFactor %f, got %f", priceFactor, *m1.PriceFactor)
	}
	if len(m1.Efforts) != 3 {
		t.Errorf("expected 3 Efforts, got %d", len(m1.Efforts))
	} else {
		if m1.Efforts[0] != "low" || m1.Efforts[1] != "medium" || m1.Efforts[2] != "high" {
			t.Errorf("expected Efforts [low medium high], got %v", m1.Efforts)
		}
	}
	if m1.DefaultEffort != "medium" {
		t.Errorf("expected DefaultEffort 'medium', got '%s'", m1.DefaultEffort)
	}

	// Second model: optional fields are absent.
	m2 := models[1]
	if m2.ID != "model_002" {
		t.Errorf("expected ID 'model_002', got '%s'", m2.ID)
	}
	if m2.PriceFactor != nil {
		t.Errorf("expected PriceFactor to be nil, got %f", *m2.PriceFactor)
	}
	if m2.Efforts != nil {
		t.Errorf("expected Efforts to be nil, got %v", m2.Efforts)
	}
	if m2.DefaultEffort != "" {
		t.Errorf("expected DefaultEffort to be empty, got '%s'", m2.DefaultEffort)
	}
	if !m2.IsNew {
		t.Error("expected IsNew to be true")
	}
}

func TestModel_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("expected path /models, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	models, err := api.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if models == nil {
		t.Fatal("expected non-nil slice, got nil")
	}
	if len(models) != 0 {
		t.Errorf("expected empty slice, got %d models", len(models))
	}
}

func TestModel_List_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	_, err := api.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsServerError() {
		t.Error("expected IsServerError to be true")
	}
	if apiErr.Message != "internal server error" {
		t.Errorf("expected message 'internal server error', got '%s'", apiErr.Message)
	}
}

func TestModel_List_Pagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/models" {
			t.Errorf("expected path /models, got %s", r.URL.Path)
		}

		// models.List does not use ListParams — verify no pagination query params are sent.
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "model_001",
					"type": "llm",
				},
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	models, err := api.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "model_001" {
		t.Errorf("expected ID 'model_001', got '%s'", models[0].ID)
	}
}
