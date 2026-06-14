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
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
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
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		config, ok := payload["config"].(map[string]interface{})
		if !ok {
			t.Fatal("expected config to be an object")
		}
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

func TestList_InvalidParams(t *testing.T) {
	t.Parallel()

	// No server needed - validation fails client-side before HTTP call.
	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)
	_, err := api.List(context.Background(), &types.ListParams{Limit: -1})
	if err == nil {
		t.Error("expected error for invalid Limit")
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/environments" {
			t.Errorf("expected path /environments, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("after_id") != "env_001" {
			t.Errorf("expected after_id=env_001, got %q", r.URL.Query().Get("after_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		lastID := "env_002"
		_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Environment]{
			Data: []Environment{
				{
					ID:          "env_002",
					Type:        "environment",
					Name:        "prod",
					Description: "production environment",
					Status:      "active",
					Config: EnvConfig{
						Type:       "sandbox",
						Networking: Networking{Type: "restricted"},
						Packages: Packages{
							Apt: []string{"curl", "git"},
							Pip: []string{"requests"},
							Npm: []string{"typescript"},
						},
					},
					Metadata:  types.Metadata{"team": "ai", "env": "prod"},
					CreatedAt: "2026-06-14T10:00:00Z",
					UpdatedAt: "2026-06-14T12:00:00Z",
				},
			},
			LastID:  &lastID,
			HasMore: false,
		})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	result, err := api.List(context.Background(), &types.ListParams{Limit: 10, AfterID: "env_001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(result.Data))
	}
	if result.HasMore {
		t.Error("expected HasMore to be false")
	}

	env := result.Data[0]
	if env.ID != "env_002" {
		t.Errorf("expected ID 'env_002', got %q", env.ID)
	}
	if env.Name != "prod" {
		t.Errorf("expected Name 'prod', got %q", env.Name)
	}
	if env.Status != "active" {
		t.Errorf("expected Status 'active', got %q", env.Status)
	}
	if env.Config.Type != "sandbox" {
		t.Errorf("expected Config.Type 'sandbox', got %q", env.Config.Type)
	}
	if env.Config.Networking.Type != "restricted" {
		t.Errorf("expected Config.Networking.Type 'restricted', got %q", env.Config.Networking.Type)
	}
	if len(env.Config.Packages.Apt) != 2 || env.Config.Packages.Apt[0] != "curl" {
		t.Errorf("expected Config.Packages.Apt [curl git], got %+v", env.Config.Packages.Apt)
	}
	if len(env.Config.Packages.Pip) != 1 || env.Config.Packages.Pip[0] != "requests" {
		t.Errorf("expected Config.Packages.Pip [requests], got %+v", env.Config.Packages.Pip)
	}
	if len(env.Config.Packages.Npm) != 1 || env.Config.Packages.Npm[0] != "typescript" {
		t.Errorf("expected Config.Packages.Npm [typescript], got %+v", env.Config.Packages.Npm)
	}
	if env.Metadata["team"] != "ai" {
		t.Errorf("expected Metadata team='ai', got %q", env.Metadata["team"])
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/environments/env_123" {
			t.Errorf("expected path /environments/env_123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Environment{
			ID:          "env_123",
			Type:        "environment",
			Name:        "staging",
			Description: "staging environment",
			Status:      "active",
			Config: EnvConfig{
				Type:       "vm",
				Networking: Networking{Type: "open"},
				Packages:   Packages{Apt: []string{"vim"}},
			},
			Metadata:  types.Metadata{"owner": "platform"},
			CreatedAt: "2026-06-14T10:00:00Z",
			UpdatedAt: "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	env, err := api.Get(context.Background(), "env_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatal("expected non-nil environment")
	}
	if env.ID != "env_123" {
		t.Errorf("expected ID 'env_123', got %q", env.ID)
	}
	if env.Name != "staging" {
		t.Errorf("expected Name 'staging', got %q", env.Name)
	}
	if env.Config.Type != "vm" {
		t.Errorf("expected Config.Type 'vm', got %q", env.Config.Type)
	}
	if env.Config.Networking.Type != "open" {
		t.Errorf("expected Config.Networking.Type 'open', got %q", env.Config.Networking.Type)
	}
	if len(env.Config.Packages.Apt) != 1 || env.Config.Packages.Apt[0] != "vim" {
		t.Errorf("expected Config.Packages.Apt [vim], got %+v", env.Config.Packages.Apt)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"env": "staging"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/environments/env_123" {
			t.Errorf("expected path /environments/env_123, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req UpdateEnvRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "updated-env" {
			t.Errorf("expected Name 'updated-env', got %q", req.Name)
		}
		if req.Description != "updated description" {
			t.Errorf("expected Description 'updated description', got %q", req.Description)
		}
		if req.Config == nil {
			t.Fatal("expected Config to be set in request body")
		}
		if req.Config.Type != "vm" {
			t.Errorf("expected Config.Type 'vm', got %q", req.Config.Type)
		}
		if req.Config.Networking.Type != "restricted" {
			t.Errorf("expected Config.Networking.Type 'restricted', got %q", req.Config.Networking.Type)
		}
		if len(req.Config.Packages.Pip) != 1 || req.Config.Packages.Pip[0] != "numpy" {
			t.Errorf("expected Config.Packages.Pip [numpy], got %+v", req.Config.Packages.Pip)
		}
		if len(req.Metadata) != 1 || req.Metadata["env"] != "staging" {
			t.Errorf("expected Metadata env='staging', got %+v", req.Metadata)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Environment{
			ID:          "env_123",
			Type:        "environment",
			Name:        req.Name,
			Description: req.Description,
			Status:      "active",
			Config:      *req.Config,
			Metadata:    req.Metadata,
			CreatedAt:   "2026-06-14T10:00:00Z",
			UpdatedAt:   "2026-06-14T13:00:00Z",
		})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	env, err := api.Update(context.Background(), "env_123",
		NewUpdateRequest().
			WithName("updated-env").
			WithDescription("updated description").
			WithConfig(EnvConfig{
				Type:       "vm",
				Networking: Networking{Type: "restricted"},
				Packages:   Packages{Pip: []string{"numpy"}},
			}).
			WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatal("expected non-nil environment")
	}
	if env.ID != "env_123" {
		t.Errorf("expected ID 'env_123', got %q", env.ID)
	}
	if env.Name != "updated-env" {
		t.Errorf("expected Name 'updated-env', got %q", env.Name)
	}
	if env.Config.Type != "vm" {
		t.Errorf("expected Config.Type 'vm', got %q", env.Config.Type)
	}
}

func TestArchive(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/environments/env_123/archive" {
			t.Errorf("expected path /environments/env_123/archive, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		archivedAt := "2026-06-14T13:00:00Z"
		_ = json.NewEncoder(w).Encode(Environment{
			ID:         "env_123",
			Type:       "environment",
			Name:       "prod",
			Status:     "archived",
			Config:     EnvConfig{Type: "sandbox"},
			ArchivedAt: &archivedAt,
			CreatedAt:  "2026-06-14T10:00:00Z",
			UpdatedAt:  "2026-06-14T13:00:00Z",
		})
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	env, err := api.Archive(context.Background(), "env_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatal("expected non-nil environment")
	}
	if env.ID != "env_123" {
		t.Errorf("expected ID 'env_123', got %q", env.ID)
	}
	if env.Status != "archived" {
		t.Errorf("expected Status 'archived', got %q", env.Status)
	}
	if env.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set")
	}
}

func TestCreate_NilRequest(t *testing.T) {
	t.Parallel()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	_, err := api.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestUpdate_NilRequest(t *testing.T) {
	t.Parallel()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	_, err := api.Update(context.Background(), "env_123", nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

// TestUpdate_NilRequestBeforeInvalidID pins the validation ordering: the
// production code checks req == nil BEFORE ValidateID. A nil request combined
// with an invalid id must surface the nil-request error, not the invalid-id one.
func TestUpdate_NilRequestBeforeInvalidID(t *testing.T) {
	t.Parallel()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)

	_, err := api.Update(context.Background(), "a/b", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "environments: UpdateEnvRequest must not be nil" {
		t.Errorf("expected nil-request error to take precedence over invalid-id, got %q", err.Error())
	}
}

func TestInvalidID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid environment ID")
	}))
	defer srv.Close()

	client := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(client)
	ctx := context.Background()
	validUpdateReq := NewUpdateRequest().WithName("test")

	invalidIDs := []string{"", "a/b", "..", "%2e", "a b"}

	tests := []struct {
		verb string
		fn   func(id string) error
	}{
		{verb: "Get", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{verb: "Update", fn: func(id string) error { _, err := api.Update(ctx, id, validUpdateReq); return err }},
		{verb: "Archive", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
		{verb: "Delete", fn: func(id string) error { return api.Delete(ctx, id) }},
	}

	for _, tt := range tests {
		for _, id := range invalidIDs {
			tt, id := tt, id
			t.Run(tt.verb+"_"+id, func(t *testing.T) {
				if err := tt.fn(id); err == nil {
					t.Errorf("expected error for invalid ID %q, got nil", id)
				}
			})
		}
	}
}
