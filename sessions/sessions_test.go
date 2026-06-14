package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

func TestAgentRef_MarshalJSON_StringForm(t *testing.T) {
	ref := NewAgentRef("agent_abc123")
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if string(data) != `"agent_abc123"` {
		t.Errorf("expected string form, got %s", data)
	}
}

func TestAgentRef_MarshalJSON_ObjectForm(t *testing.T) {
	ref := NewAgentRefWithVersion("agent_abc123", 3)
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	expected := `{"id":"agent_abc123","version":3}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, data)
	}
}

func TestAgentRef_UnmarshalJSON_StringForm(t *testing.T) {
	var ref AgentRef
	err := json.Unmarshal([]byte(`"agent_xyz"`), &ref)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if ref.ID != "agent_xyz" {
		t.Errorf("expected ID agent_xyz, got %s", ref.ID)
	}
	if ref.Version != 0 {
		t.Errorf("expected version 0, got %d", ref.Version)
	}
}

func TestAgentRef_UnmarshalJSON_ObjectForm(t *testing.T) {
	var ref AgentRef
	err := json.Unmarshal([]byte(`{"id":"agent_xyz","version":5}`), &ref)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if ref.ID != "agent_xyz" {
		t.Errorf("expected ID agent_xyz, got %s", ref.ID)
	}
	if ref.Version != 5 {
		t.Errorf("expected version 5, got %d", ref.Version)
	}
}

func TestAgentRef_UnmarshalJSON_Invalid(t *testing.T) {
	var ref AgentRef
	err := json.Unmarshal([]byte(`123`), &ref)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestAgentRef_RoundTrip_String(t *testing.T) {
	original := NewAgentRef("agent_rtt")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded AgentRef
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ID != original.ID || decoded.Version != original.Version {
		t.Errorf("round-trip mismatch: %+v vs %+v", original, decoded)
	}
}

func TestAgentRef_RoundTrip_Object(t *testing.T) {
	original := NewAgentRefWithVersion("agent_rtt", 7)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded AgentRef
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ID != original.ID || decoded.Version != original.Version {
		t.Errorf("round-trip mismatch: %+v vs %+v", original, decoded)
	}
}

func TestCancel_ReturnsCancelingStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_123/cancel" {
			t.Errorf("expected path /sessions/sess_123/cancel, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"canceling"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Cancel(context.Background(), "sess_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil CancelResponse")
	}
	if resp.Status != "canceling" {
		t.Errorf("expected status 'canceling', got %q", resp.Status)
	}
}

func TestSession_UnmarshalJSON_WithResources(t *testing.T) {
	payload := `{
		"id": "sess_res456",
		"agent": "agent_abc",
		"status": "running",
		"created_at": "2026-06-13T00:00:00Z",
		"updated_at": "2026-06-13T00:00:00Z",
		"resources": [
			{"file_id": "file_1", "path": "/docs/readme.md", "type": "file"},
			{"url": "https://github.com/org/repo", "mount_path": "/repo", "type": "github_repository"}
		]
	}`

	var session Session
	if err := json.Unmarshal([]byte(payload), &session); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(session.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(session.Resources))
	}
	if session.Resources[0].FileID != "file_1" || session.Resources[0].Path != "/docs/readme.md" || session.Resources[0].Type != "file" {
		t.Errorf("unexpected first resource: %+v", session.Resources[0])
	}
	if session.Resources[1].URL != "https://github.com/org/repo" || session.Resources[1].MountPath != "/repo" || session.Resources[1].Type != "github_repository" {
		t.Errorf("unexpected second resource: %+v", session.Resources[1])
	}
}

func TestDelete_SendsDELETEAndHandles200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_789" {
			t.Errorf("expected path /sessions/sess_789, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	if err := api.Delete(context.Background(), "sess_789"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_WithPaginationParams(t *testing.T) {
	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/sessions" {
				t.Errorf("expected path /sessions, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("limit") != "10" {
				t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("after_id") != "after_xyz" {
				t.Errorf("expected after_id=after_xyz, got %s", r.URL.Query().Get("after_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"sess_1","agent":"agent_a","status":"running","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}],"first_id":"sess_1","last_id":"sess_1","has_more":false}`))
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		params := &types.ListParams{Limit: 10, AfterID: "after_xyz"}
		resp, err := api.List(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 session, got %d", len(resp.Data))
		}
		if resp.Data[0].ID != "sess_1" {
			t.Errorf("expected sess_1, got %s", resp.Data[0].ID)
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

func TestList_WithNilParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/sessions" {
			t.Errorf("expected path /sessions, got %s", r.URL.Path)
		}
		// nil params means no query string.
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"has_more":false}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.HasMore {
		t.Error("expected HasMore=false")
	}
}

func TestList_ServerError(t *testing.T) {
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

	_, err := api.List(context.Background(), nil)
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

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sessions" {
			t.Errorf("expected path /sessions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sess_new","agent":"agent_abc","status":"starting","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewCreateRequest("agent_abc").WithTitle("test sesh")
	sess, err := api.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID != "sess_new" {
		t.Errorf("expected sess_new, got %s", sess.ID)
	}
	if sess.Status != "starting" {
		t.Errorf("expected starting, got %s", sess.Status)
	}
}

func TestCreate_AllBuilderOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		// Parse the request body to verify all fields were serialized.
		var body CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.Agent.ID != "agent_full" {
			t.Errorf("expected agent_full, got %s", body.Agent.ID)
		}
		if body.Title != "full test" {
			t.Errorf("expected title 'full test', got %s", body.Title)
		}
		if body.EnvironmentID != "env_123" {
			t.Errorf("expected env_123, got %s", body.EnvironmentID)
		}
		if body.DeltaFlushIntervalMs != 500 {
			t.Errorf("expected 500, got %d", body.DeltaFlushIntervalMs)
		}
		if body.Metadata["key1"] != "val1" {
			t.Errorf("expected metadata key1=val1, got %v", body.Metadata)
		}
		if body.EnvironmentVariables != "FOO=bar\nBAZ=qux" {
			t.Errorf("expected env vars, got %q", body.EnvironmentVariables)
		}
		if len(body.Resources) != 2 {
			t.Fatalf("expected 2 resources, got %d", len(body.Resources))
		}
		if body.Resources[0].Type != "file" || body.Resources[0].FileID != "file_1" {
			t.Errorf("unexpected resource 0: %+v", body.Resources[0])
		}
		if body.Resources[1].Type != "github_repository" || body.Resources[1].URL != "https://github.com/org/repo" {
			t.Errorf("unexpected resource 1: %+v", body.Resources[1])
		}
		if len(body.VaultIDs) != 1 || body.VaultIDs[0] != "vault_1" {
			t.Errorf("unexpected vault IDs: %v", body.VaultIDs)
		}
		if len(body.MemoryStoreIDs) != 1 || body.MemoryStoreIDs[0] != "mem_1" {
			t.Errorf("unexpected memory store IDs: %v", body.MemoryStoreIDs)
		}
		if body.Environment.(map[string]interface{})["name"] != "inline_env" {
			t.Errorf("unexpected inline environment: %v", body.Environment)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sess_full","agent":"agent_full","status":"running","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewCreateRequest("agent_full").
		WithTitle("full test").
		WithEnvironment("env_123").
		WithMetadata(types.Metadata{"key1": "val1"}).
		WithDeltaFlushInterval(500).
		WithResource(NewResourceFile("file_1", "/docs/readme.md")).
		WithResource(NewResourceGitHub("https://github.com/org/repo", "/repo")).
		WithVault("vault_1").
		WithMemoryStore("mem_1").
		WithEnvironmentVariables("FOO=bar\nBAZ=qux").
		WithInlineEnvironment(map[string]interface{}{"name": "inline_env"})

	sess, err := api.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID != "sess_full" {
		t.Errorf("expected sess_full, got %s", sess.ID)
	}
}

func TestCreate_WithIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "idem-key-123" {
			t.Errorf("expected Idempotency-Key 'idem-key-123', got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sess_idem","agent":"agent_abc","status":"starting","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewCreateRequest("agent_abc")
	sess, err := api.Create(context.Background(), req, "idem-key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID != "sess_idem" {
		t.Errorf("expected sess_idem, got %s", sess.ID)
	}
}

func TestCreate_NilRequest(t *testing.T) {
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost:0", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestCreate_EmptyAgentID(t *testing.T) {
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost:0", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &CreateSessionRequest{}
	_, err := api.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty Agent.ID, got nil")
	}
}

func TestCreate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "agent not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewCreateRequest("agent_bad")
	_, err := api.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_get" {
			t.Errorf("expected path /sessions/sess_get, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_get","agent":"agent_abc","status":"running","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	sess, err := api.Get(context.Background(), "sess_get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID != "sess_get" {
		t.Errorf("expected sess_get, got %s", sess.ID)
	}
}

func TestGet_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with slash", id: "bad/id"},
		{name: "with dotdot", id: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			_, err := api.Get(context.Background(), tt.id)
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "not_found",
				"message": "session not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Get(context.Background(), "sess_404")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUpdate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_upd" {
			t.Errorf("expected path /sessions/sess_upd, got %s", r.URL.Path)
		}

		var body UpdateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Title != "new title" {
			t.Errorf("expected title 'new title', got %q", body.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_upd","agent":"agent_abc","title":"new title","status":"running","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewUpdateRequest().WithTitle("new title")
	sess, err := api.Update(context.Background(), "sess_upd", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.Title != "new title" {
		t.Errorf("expected title 'new title', got %q", sess.Title)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with slash", id: "bad/id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			req := NewUpdateRequest().WithTitle("irrelevant")
			_, err := api.Update(context.Background(), tt.id, req)
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

func TestUpdate_NilRequest(t *testing.T) {
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost:0", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Update(context.Background(), "sess_xyz", nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
}

func TestUpdate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "conflict",
				"message": "session is archived",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewUpdateRequest().WithTitle("late")
	_, err := api.Update(context.Background(), "sess_conflict", req)
	if err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Archive
// ---------------------------------------------------------------------------

func TestArchive_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_arch/archive" {
			t.Errorf("expected path /sessions/sess_arch/archive, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_arch","agent":"agent_abc","status":"archived","created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	sess, err := api.Archive(context.Background(), "sess_arch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.Status != "archived" {
		t.Errorf("expected status 'archived', got %q", sess.Status)
	}
}

func TestArchive_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with dotdot", id: "../unauthorized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			_, err := api.Archive(context.Background(), tt.id)
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

func TestArchive_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "not_found",
				"message": "session sess_bad not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Archive(context.Background(), "sess_bad")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound to be true")
	}
}

// ---------------------------------------------------------------------------
// AddResources
// ---------------------------------------------------------------------------

func TestAddResources_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/sess_res/resources" {
			t.Errorf("expected path /sessions/sess_res/resources, got %s", r.URL.Path)
		}

		var body struct {
			Resources []Resource `json:"resources"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Resources) != 2 {
			t.Fatalf("expected 2 resources, got %d", len(body.Resources))
		}
		if body.Resources[0].Type != "file" || body.Resources[0].FileID != "file_add" || body.Resources[0].Path != "/new/doc.txt" {
			t.Errorf("unexpected resource 0: %+v", body.Resources[0])
		}
		if body.Resources[1].Type != "github_repository" || body.Resources[1].URL != "https://github.com/org/r" || body.Resources[1].MountPath != "/mnt" {
			t.Errorf("unexpected resource 1: %+v", body.Resources[1])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_res","agent":"agent_abc","status":"running","resources":[{"type":"file","file_id":"file_add","path":"/new/doc.txt"},{"type":"github_repository","url":"https://github.com/org/r","mount_path":"/mnt"}],"created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resources := []Resource{
		NewResourceFile("file_add", "/new/doc.txt"),
		NewResourceGitHub("https://github.com/org/r", "/mnt"),
	}
	sess, err := api.AddResources(context.Background(), "sess_res", resources)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sess.Resources) != 2 {
		t.Fatalf("expected 2 resources in response, got %d", len(sess.Resources))
	}
	if sess.Resources[0].FileID != "file_add" {
		t.Errorf("expected file_add, got %s", sess.Resources[0].FileID)
	}
}

func TestAddResources_EmptySlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body struct {
			Resources []Resource `json:"resources"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		// Server should accept empty resources (attaches nothing).
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sess_res","agent":"agent_abc","status":"running","resources":[],"created_at":"2026-06-14T00:00:00Z","updated_at":"2026-06-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	sess, err := api.AddResources(context.Background(), "sess_res", []Resource{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session")
	}
	if len(sess.Resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(sess.Resources))
	}
}

func TestAddResources_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with backslash", id: "bad\\id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			_, err := api.AddResources(context.Background(), tt.id, []Resource{NewResourceFile("f", "/p")})
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cancel -- invalid ID and server error (happy-path already covered above)
// ---------------------------------------------------------------------------

func TestCancel_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with hash", id: "bad#id"},
		{name: "with question", id: "bad?id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			_, err := api.Cancel(context.Background(), tt.id)
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

func TestCancel_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "conflict",
				"message": "session is not running",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Cancel(context.Background(), "sess_done")
	if err == nil {
		t.Fatal("expected error for 409, got nil")
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		t.Fatalf("expected *qoderhttp.APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Delete -- invalid ID (happy-path already covered above)
// ---------------------------------------------------------------------------

func TestDelete_InvalidID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: ""},
		{name: "with dotdot", id: "../sess"},
		{name: "with slash", id: "bad/sess"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("unexpected HTTP call for invalid ID %q", tt.id)
			}))
			defer srv.Close()

			c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
			api := NewAPI(c)

			err := api.Delete(context.Background(), tt.id)
			if err == nil {
				t.Fatal("expected error for invalid ID, got nil")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Builder tests
// ---------------------------------------------------------------------------

func TestCreateSessionRequest_Builder(t *testing.T) {
	t.Parallel()

	inlineEnv := map[string]interface{}{
		"name":        "ephemeral",
		"description": "inline test env",
	}

	req := NewCreateRequest("agent_builder").
		WithTitle("builder test").
		WithEnvironment("env_builder").
		WithMetadata(types.Metadata{"k": "v"}).
		WithDeltaFlushInterval(250).
		WithResource(NewResourceFile("file_b", "/path/b.txt")).
		WithResource(NewResourceGitHub("https://github.com/b/repo", "/b")).
		WithVault("vault_b1").
		WithVault("vault_b2").
		WithMemoryStore("mem_b").
		WithEnvironmentVariables("A=B").
		WithInlineEnvironment(inlineEnv)

	if req.Agent.ID != "agent_builder" {
		t.Errorf("expected agent_builder, got %s", req.Agent.ID)
	}
	if req.Title != "builder test" {
		t.Errorf("expected 'builder test', got %q", req.Title)
	}
	if req.EnvironmentID != "env_builder" {
		t.Errorf("expected env_builder, got %s", req.EnvironmentID)
	}
	if req.Metadata["k"] != "v" {
		t.Errorf("expected metadata k=v, got %v", req.Metadata)
	}
	if req.DeltaFlushIntervalMs != 250 {
		t.Errorf("expected 250, got %d", req.DeltaFlushIntervalMs)
	}
	if len(req.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(req.Resources))
	}
	if req.Resources[0].FileID != "file_b" || req.Resources[0].Path != "/path/b.txt" {
		t.Errorf("unexpected resource 0: %+v", req.Resources[0])
	}
	if req.Resources[1].URL != "https://github.com/b/repo" || req.Resources[1].MountPath != "/b" {
		t.Errorf("unexpected resource 1: %+v", req.Resources[1])
	}
	if len(req.VaultIDs) != 2 || req.VaultIDs[0] != "vault_b1" || req.VaultIDs[1] != "vault_b2" {
		t.Errorf("unexpected vault IDs: %v", req.VaultIDs)
	}
	if len(req.MemoryStoreIDs) != 1 || req.MemoryStoreIDs[0] != "mem_b" {
		t.Errorf("unexpected memory store IDs: %v", req.MemoryStoreIDs)
	}
	if req.EnvironmentVariables != "A=B" {
		t.Errorf("expected 'A=B', got %q", req.EnvironmentVariables)
	}
	if req.Environment.(map[string]interface{})["name"] != "ephemeral" {
		t.Errorf("unexpected inline env: %v", req.Environment)
	}
}

func TestUpdateSessionRequest_Builder(t *testing.T) {
	t.Parallel()

	req := NewUpdateRequest().
		WithTitle("updated title").
		WithMetadata(types.Metadata{"updated": "yes"})

	if req.Title != "updated title" {
		t.Errorf("expected 'updated title', got %q", req.Title)
	}
	if req.Metadata["updated"] != "yes" {
		t.Errorf("expected metadata updated=yes, got %v", req.Metadata)
	}
}

func TestNewResourceFile(t *testing.T) {
	t.Parallel()

	r := NewResourceFile("file_x", "/path/x.txt")
	if r.Type != "file" {
		t.Errorf("expected type 'file', got %q", r.Type)
	}
	if r.FileID != "file_x" {
		t.Errorf("expected file_x, got %s", r.FileID)
	}
	if r.Path != "/path/x.txt" {
		t.Errorf("expected /path/x.txt, got %s", r.Path)
	}
}

func TestNewResourceGitHub(t *testing.T) {
	t.Parallel()

	r := NewResourceGitHub("https://github.com/x/y", "/mount/x")
	if r.Type != "github_repository" {
		t.Errorf("expected type 'github_repository', got %q", r.Type)
	}
	if r.URL != "https://github.com/x/y" {
		t.Errorf("expected url, got %s", r.URL)
	}
	if r.MountPath != "/mount/x" {
		t.Errorf("expected /mount/x, got %s", r.MountPath)
	}
}
