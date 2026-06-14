package agents

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

func TestArchive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/agents/agent_123/archive" {
			t.Errorf("expected path /agents/agent_123/archive, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"archived":    true,
			"archived_at": "2026-06-13T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Archive(context.Background(), "agent_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.Archived {
		t.Errorf("expected Archived to be true")
	}
	if resp.ArchivedAt == "" {
		t.Errorf("expected ArchivedAt to be non-empty")
	}
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/agents/agent_123" {
			t.Errorf("expected path /agents/agent_123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	if err := api.Delete(context.Background(), "agent_123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("with_pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/agents" {
				t.Errorf("expected path /agents, got %s", r.URL.Path)
			}

			limit := r.URL.Query().Get("limit")
			if limit != "10" {
				t.Errorf("expected limit=10, got %q", limit)
			}
			afterID := r.URL.Query().Get("after_id")
			if afterID != "agent_001" {
				t.Errorf("expected after_id=agent_001, got %q", afterID)
			}

			w.Header().Set("Content-Type", "application/json")
			lastID := "agent_003"
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Agent]{
				Data: []Agent{
					{ID: "agent_002", Name: "Agent Two", Model: "claude-sonnet-4-20250514"},
					{ID: "agent_003", Name: "Agent Three", Model: "claude-opus-4-20250514"},
				},
				LastID:  &lastID,
				HasMore: false,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		result, err := api.List(context.Background(), &types.ListParams{
			Limit:   10,
			AfterID: "agent_001",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 agents, got %d", len(result.Data))
		}
		if result.Data[0].ID != "agent_002" {
			t.Errorf("expected first agent ID 'agent_002', got %q", result.Data[0].ID)
		}
		if result.HasMore {
			t.Error("expected HasMore to be false")
		}
	})

	t.Run("nil_params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/agents" {
				t.Errorf("expected path /agents, got %s", r.URL.Path)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params, got %q", r.URL.RawQuery)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Agent]{
				Data:    []Agent{},
				HasMore: false,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		result, err := api.List(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 0 {
			t.Errorf("expected 0 agents, got %d", len(result.Data))
		}
	})

	t.Run("empty_params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/agents" {
				t.Errorf("expected path /agents, got %s", r.URL.Path)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params, got %q", r.URL.RawQuery)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Agent]{
				Data: []Agent{
					{ID: "agent_004", Name: "Agent Four", Model: "claude-haiku-4-20250514"},
				},
				HasMore: true,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		result, err := api.List(context.Background(), &types.ListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 1 {
			t.Errorf("expected 1 agent, got %d", len(result.Data))
		}
		if !result.HasMore {
			t.Error("expected HasMore to be true")
		}
	})

	t.Run("invalid params returns error", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)
		_, err := api.List(context.Background(), &types.ListParams{Limit: -1})
		if err == nil {
			t.Error("expected error for invalid Limit")
		}
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"env": "production", "team": "ai"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/agents" {
			t.Errorf("expected path /agents, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req CreateAgentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "test-agent" {
			t.Errorf("expected Name 'test-agent', got %q", req.Name)
		}
		if req.Model != "claude-sonnet-4-20250514" {
			t.Errorf("expected Model 'claude-sonnet-4-20250514', got %q", req.Model)
		}
		if req.System != "You are a helpful assistant." {
			t.Errorf("expected System 'You are a helpful assistant.', got %q", req.System)
		}
		if req.Description != "A test agent" {
			t.Errorf("expected Description 'A test agent', got %q", req.Description)
		}
		if len(req.Tools) != 1 || req.Tools[0].Type != "code_execution" {
			t.Errorf("expected Tools [code_execution], got %+v", req.Tools)
		}
		if len(req.MCPServers) != 1 || req.MCPServers[0].Name != "filesystem" {
			t.Errorf("expected MCPServers [filesystem], got %+v", req.MCPServers)
		}
		if len(req.Skills) != 1 || req.Skills[0].SkillID != "skill_001" {
			t.Errorf("expected Skills [skill_001], got %+v", req.Skills)
		}
		if len(req.Metadata) != 2 {
			t.Errorf("expected 2 metadata entries, got %d", len(req.Metadata))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Agent{
			ID:          "agent_010",
			Type:        "agent",
			Name:        req.Name,
			Model:       req.Model,
			System:      req.System,
			Description: req.Description,
			Tools:       req.Tools,
			MCPServers:  req.MCPServers,
			Skills:      req.Skills,
			Metadata:    req.Metadata,
			Version:     1,
			Archived:    false,
			CreatedAt:   "2026-06-14T12:00:00Z",
			UpdatedAt:   "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	agent, err := api.Create(context.Background(),
		NewCreateRequest("test-agent", "claude-sonnet-4-20250514").
			WithSystem("You are a helpful assistant.").
			WithDescription("A test agent").
			WithTool(Tool{Type: "code_execution"}).
			WithMCPServer(MCPServer{Type: "mcp", Name: "filesystem", URL: "http://localhost:8080"}).
			WithSkill(SkillRef{Type: "skill", SkillID: "skill_001", Version: 1}).
			WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.ID != "agent_010" {
		t.Errorf("expected ID 'agent_010', got %q", agent.ID)
	}
	if agent.Type != "agent" {
		t.Errorf("expected Type 'agent', got %q", agent.Type)
	}
	if agent.Name != "test-agent" {
		t.Errorf("expected Name 'test-agent', got %q", agent.Name)
	}
	if agent.Version != 1 {
		t.Errorf("expected Version 1, got %d", agent.Version)
	}
}

func TestCreate_WithIdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/agents" {
			t.Errorf("expected path /agents, got %s", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "idem-key-abc" {
			t.Errorf("expected Idempotency-Key 'idem-key-abc', got %q", key)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Agent{
			ID:    "agent_020",
			Type:  "agent",
			Name:  "idem-agent",
			Model: "claude-sonnet-4-20250514",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	agent, err := api.Create(context.Background(),
		NewCreateRequest("idem-agent", "claude-sonnet-4-20250514"),
		"idem-key-abc",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.ID != "agent_020" {
		t.Errorf("expected ID 'agent_020', got %q", agent.ID)
	}
}

func TestCreate_Error(t *testing.T) {
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

	_, err := api.Create(context.Background(), NewCreateRequest("test-agent", "claude-sonnet"))
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
}

func TestGet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agents/agent_001" {
			t.Errorf("expected path /agents/agent_001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Agent{
			ID:          "agent_001",
			Type:        "agent",
			Name:        "My Agent",
			Model:       "claude-sonnet-4-20250514",
			System:      "You are helpful.",
			Description: "A test agent",
			Version:     3,
			Archived:    false,
			CreatedAt:   "2026-01-01T00:00:00Z",
			UpdatedAt:   "2026-06-14T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	agent, err := api.Get(context.Background(), "agent_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.ID != "agent_001" {
		t.Errorf("expected ID 'agent_001', got %q", agent.ID)
	}
	if agent.Name != "My Agent" {
		t.Errorf("expected Name 'My Agent', got %q", agent.Name)
	}
	if agent.Version != 3 {
		t.Errorf("expected Version 3, got %d", agent.Version)
	}
}

func TestGet_Error(t *testing.T) {
	t.Parallel()

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

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Get(context.Background(), "agent_001")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"env": "staging"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/agents/agent_001" {
			t.Errorf("expected path /agents/agent_001, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req UpdateAgentRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Version != 2 {
			t.Errorf("expected Version 2, got %d", req.Version)
		}
		if req.Name != "updated-agent" {
			t.Errorf("expected Name 'updated-agent', got %q", req.Name)
		}
		if req.Model != "claude-opus-4-20250514" {
			t.Errorf("expected Model 'claude-opus-4-20250514', got %q", req.Model)
		}
		if req.System != "Updated system prompt." {
			t.Errorf("expected System 'Updated system prompt.', got %q", req.System)
		}
		if req.Description != "Updated description" {
			t.Errorf("expected Description 'Updated description', got %q", req.Description)
		}
		if len(req.Tools) != 1 || req.Tools[0].Type != "web_search" {
			t.Errorf("expected Tools [web_search], got %+v", req.Tools)
		}
		if len(req.MCPServers) != 1 || req.MCPServers[0].Name != "github" {
			t.Errorf("expected MCPServers [github], got %+v", req.MCPServers)
		}
		if len(req.Skills) != 1 || req.Skills[0].SkillID != "skill_002" {
			t.Errorf("expected Skills [skill_002], got %+v", req.Skills)
		}
		if len(req.Metadata) != 1 {
			t.Errorf("expected 1 metadata entry, got %d", len(req.Metadata))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Agent{
			ID:          "agent_001",
			Type:        "agent",
			Name:        req.Name,
			Model:       req.Model,
			System:      req.System,
			Description: req.Description,
			Tools:       req.Tools,
			MCPServers:  req.MCPServers,
			Skills:      req.Skills,
			Metadata:    req.Metadata,
			Version:     3,
			Archived:    false,
			CreatedAt:   "2026-01-01T00:00:00Z",
			UpdatedAt:   "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	agent, err := api.Update(context.Background(), "agent_001",
		NewUpdateRequest(2).
			WithName("updated-agent").
			WithModel("claude-opus-4-20250514").
			WithSystem("Updated system prompt.").
			WithDescription("Updated description").
			WithTool(Tool{Type: "web_search"}).
			WithMCPServer(MCPServer{Type: "mcp", Name: "github", URL: "https://github.com"}).
			WithSkill(SkillRef{Type: "skill", SkillID: "skill_002", Version: 1}).
			WithMetadata(meta),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
	if agent.ID != "agent_001" {
		t.Errorf("expected ID 'agent_001', got %q", agent.ID)
	}
	if agent.Version != 3 {
		t.Errorf("expected Version 3, got %d", agent.Version)
	}
	if agent.Name != "updated-agent" {
		t.Errorf("expected Name 'updated-agent', got %q", agent.Name)
	}
}

func TestUpdate_Error(t *testing.T) {
	t.Parallel()

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

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Update(context.Background(), "agent_001", NewUpdateRequest(1).WithName("test"))
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
}

func TestListVersions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/agents/agent_001/versions" {
			t.Errorf("expected path /agents/agent_001/versions, got %s", r.URL.Path)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "20" {
			t.Errorf("expected limit=20, got %q", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(types.PaginatedResponse[AgentVersion]{
			Data: []AgentVersion{
				{Version: 3, Model: "claude-sonnet-4-20250514", System: "v3 system", CreatedAt: "2026-06-14T00:00:00Z"},
				{Version: 2, Model: "claude-sonnet-4-20250514", System: "v2 system", CreatedAt: "2026-06-01T00:00:00Z"},
				{Version: 1, Model: "claude-haiku-4-20250514", System: "v1 system", CreatedAt: "2026-05-01T00:00:00Z"},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	result, err := api.ListVersions(context.Background(), "agent_001", &types.ListParams{Limit: 20})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Data) != 3 {
		t.Errorf("expected 3 versions, got %d", len(result.Data))
	}
	if result.Data[0].Version != 3 {
		t.Errorf("expected first version 3, got %d", result.Data[0].Version)
	}
	if result.HasMore {
		t.Error("expected HasMore to be false")
	}
}

func TestListVersions_InvalidParams(t *testing.T) {
	t.Parallel()

	// No server needed - validation fails client-side before HTTP call.
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)
	_, err := api.ListVersions(context.Background(), "agent_001", &types.ListParams{Limit: 200})
	if err == nil {
		t.Error("expected error for invalid Limit > 100")
	}
}

func TestInvalidID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid agent ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)
	ctx := context.Background()
	validUpdateReq := NewUpdateRequest(1).WithName("test")

	tests := []struct {
		name string
		id   string
		fn   func(id string) error
	}{
		{name: "Get_empty", id: "", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_slash", id: "a/b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_dotdot", id: "a..b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_space", id: "a b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_encoded", id: "a%2fb", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Update_empty", id: "", fn: func(id string) error { _, err := api.Update(ctx, id, validUpdateReq); return err }},
		{name: "Update_slash", id: "a/b", fn: func(id string) error { _, err := api.Update(ctx, id, validUpdateReq); return err }},
		{name: "Update_dotdot", id: "a..b", fn: func(id string) error { _, err := api.Update(ctx, id, validUpdateReq); return err }},
		{name: "Archive_empty", id: "", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
		{name: "Archive_slash", id: "a/b", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
		{name: "Archive_dotdot", id: "a..b", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
		{name: "Delete_empty", id: "", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "Delete_slash", id: "a/b", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "Delete_dotdot", id: "a..b", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "ListVersions_empty", id: "", fn: func(id string) error { _, err := api.ListVersions(ctx, id, nil); return err }},
		{name: "ListVersions_slash", id: "a/b", fn: func(id string) error { _, err := api.ListVersions(ctx, id, nil); return err }},
		{name: "ListVersions_dotdot", id: "a..b", fn: func(id string) error { _, err := api.ListVersions(ctx, id, nil); return err }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.id)
			if err == nil {
				t.Errorf("expected error for invalid ID %q, got nil", tt.id)
			}
		})
	}
}
func TestCreate_NilRequest(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestCreate_EmptyName(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Create(context.Background(), &CreateAgentRequest{Name: "", Model: "x"})
	if err == nil {
		t.Fatal("expected error for empty Name, got nil")
	}
	if err.Error() != "agents: CreateAgentRequest.Name is required" {
		t.Errorf("expected 'agents: CreateAgentRequest.Name is required', got %q", err.Error())
	}
}

func TestCreate_EmptyModel(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Create(context.Background(), &CreateAgentRequest{Name: "x", Model: ""})
	if err == nil {
		t.Fatal("expected error for empty Model, got nil")
	}
	if err.Error() != "agents: CreateAgentRequest.Model is required" {
		t.Errorf("expected 'agents: CreateAgentRequest.Model is required', got %q", err.Error())
	}
}

func TestUpdate_NilRequest(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Update(context.Background(), "agent_123", nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestArchive_Error(t *testing.T) {
	t.Parallel()

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

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Archive(context.Background(), "agent_123")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}

	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
}

func TestCreateAgentRequest_Builder(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"env": "production", "team": "platform"}
	tool := Tool{Type: "code_execution", EnabledTools: []string{"bash", "python"}}
	mcp := MCPServer{Type: "mcp", Name: "filesystem", URL: "http://localhost:8080"}
	skill := SkillRef{Type: "skill", SkillID: "skill_001", Version: 1}

	req := NewCreateRequest("my-agent", "claude-sonnet-4-20250514").
		WithSystem("You are helpful.").
		WithDescription("My test agent").
		WithTool(tool).
		WithMCPServer(mcp).
		WithSkill(skill).
		WithMetadata(meta)

	if req.Name != "my-agent" {
		t.Errorf("expected Name 'my-agent', got %q", req.Name)
	}
	if req.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected Model 'claude-sonnet-4-20250514', got %q", req.Model)
	}
	if req.System != "You are helpful." {
		t.Errorf("expected System 'You are helpful.', got %q", req.System)
	}
	if req.Description != "My test agent" {
		t.Errorf("expected Description 'My test agent', got %q", req.Description)
	}
	if len(req.Tools) != 1 {
		t.Errorf("expected 1 Tool, got %d", len(req.Tools))
	}
	if req.Tools[0].Type != "code_execution" {
		t.Errorf("expected Tool Type 'code_execution', got %q", req.Tools[0].Type)
	}
	if len(req.Tools[0].EnabledTools) != 2 {
		t.Errorf("expected 2 EnabledTools, got %d", len(req.Tools[0].EnabledTools))
	}
	if len(req.MCPServers) != 1 {
		t.Errorf("expected 1 MCPServer, got %d", len(req.MCPServers))
	}
	if req.MCPServers[0].Name != "filesystem" {
		t.Errorf("expected MCPServer Name 'filesystem', got %q", req.MCPServers[0].Name)
	}
	if len(req.Skills) != 1 {
		t.Errorf("expected 1 Skill, got %d", len(req.Skills))
	}
	if req.Skills[0].SkillID != "skill_001" {
		t.Errorf("expected Skill SkillID 'skill_001', got %q", req.Skills[0].SkillID)
	}
	if len(req.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(req.Metadata))
	}
	if req.Metadata["env"] != "production" {
		t.Errorf("expected metadata env='production', got %q", req.Metadata["env"])
	}

	// Verify chaining returns the same pointer.
	req2 := NewCreateRequest("x", "y")
	if req2.WithSystem("s") != req2 {
		t.Error("WithSystem should return the same pointer for chaining")
	}
	if req2.WithDescription("d") != req2 {
		t.Error("WithDescription should return the same pointer for chaining")
	}
	if req2.WithTool(Tool{Type: "t"}) != req2 {
		t.Error("WithTool should return the same pointer for chaining")
	}
	if req2.WithMCPServer(MCPServer{Type: "mcp", Name: "n"}) != req2 {
		t.Error("WithMCPServer should return the same pointer for chaining")
	}
	if req2.WithSkill(SkillRef{Type: "skill", SkillID: "s"}) != req2 {
		t.Error("WithSkill should return the same pointer for chaining")
	}
	if req2.WithMetadata(nil) != req2 {
		t.Error("WithMetadata should return the same pointer for chaining")
	}
}

func TestUpdateAgentRequest_Builder(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"source": "api"}
	tool := Tool{Type: "web_search"}
	mcp := MCPServer{Type: "mcp", Name: "github", URL: "https://github.com"}
	skill := SkillRef{Type: "skill", SkillID: "skill_002", Version: 2}

	req := NewUpdateRequest(3).
		WithName("updated-agent").
		WithModel("claude-opus-4-20250514").
		WithSystem("Updated system.").
		WithDescription("Updated desc.").
		WithTool(tool).
		WithMCPServer(mcp).
		WithSkill(skill).
		WithMetadata(meta)

	if req.Version != 3 {
		t.Errorf("expected Version 3, got %d", req.Version)
	}
	if req.Name != "updated-agent" {
		t.Errorf("expected Name 'updated-agent', got %q", req.Name)
	}
	if req.Model != "claude-opus-4-20250514" {
		t.Errorf("expected Model 'claude-opus-4-20250514', got %q", req.Model)
	}
	if req.System != "Updated system." {
		t.Errorf("expected System 'Updated system.', got %q", req.System)
	}
	if req.Description != "Updated desc." {
		t.Errorf("expected Description 'Updated desc.', got %q", req.Description)
	}
	if len(req.Tools) != 1 {
		t.Errorf("expected 1 Tool, got %d", len(req.Tools))
	}
	if req.Tools[0].Type != "web_search" {
		t.Errorf("expected Tool Type 'web_search', got %q", req.Tools[0].Type)
	}
	if len(req.MCPServers) != 1 {
		t.Errorf("expected 1 MCPServer, got %d", len(req.MCPServers))
	}
	if req.MCPServers[0].Name != "github" {
		t.Errorf("expected MCPServer Name 'github', got %q", req.MCPServers[0].Name)
	}
	if len(req.Skills) != 1 {
		t.Errorf("expected 1 Skill, got %d", len(req.Skills))
	}
	if req.Skills[0].SkillID != "skill_002" {
		t.Errorf("expected Skill SkillID 'skill_002', got %q", req.Skills[0].SkillID)
	}
	if req.Skills[0].Version != 2 {
		t.Errorf("expected Skill Version 2, got %d", req.Skills[0].Version)
	}
	if len(req.Metadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(req.Metadata))
	}
	if req.Metadata["source"] != "api" {
		t.Errorf("expected metadata source='api', got %q", req.Metadata["source"])
	}

	// Verify chaining returns the same pointer.
	req2 := NewUpdateRequest(1)
	if req2.WithName("n") != req2 {
		t.Error("WithName should return the same pointer for chaining")
	}
	if req2.WithModel("m") != req2 {
		t.Error("WithModel should return the same pointer for chaining")
	}
	if req2.WithSystem("s") != req2 {
		t.Error("WithSystem should return the same pointer for chaining")
	}
	if req2.WithDescription("d") != req2 {
		t.Error("WithDescription should return the same pointer for chaining")
	}
	if req2.WithTool(Tool{Type: "t"}) != req2 {
		t.Error("WithTool should return the same pointer for chaining")
	}
	if req2.WithMCPServer(MCPServer{Type: "mcp", Name: "n"}) != req2 {
		t.Error("WithMCPServer should return the same pointer for chaining")
	}
	if req2.WithSkill(SkillRef{Type: "skill", SkillID: "s"}) != req2 {
		t.Error("WithSkill should return the same pointer for chaining")
	}
	if req2.WithMetadata(nil) != req2 {
		t.Error("WithMetadata should return the same pointer for chaining")
	}
}
