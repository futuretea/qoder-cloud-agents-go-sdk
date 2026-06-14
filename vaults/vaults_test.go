package vaults

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

func TestVault_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults" {
			t.Errorf("expected path /vaults, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		if reqBody["display_name"] != "test-vault" {
			t.Errorf("expected display_name 'test-vault', got %v", reqBody["display_name"])
		}

		creds, ok := reqBody["credentials"].([]interface{})
		if !ok {
			t.Fatal("expected credentials array in request body")
		}
		if len(creds) == 0 {
			t.Fatal("expected at least one credential in request body")
		}
		cred := creds[0].(map[string]interface{})
		if cred["access_token"] != "secret-token" {
			t.Errorf("expected access_token 'secret-token', got %v", cred["access_token"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "vault_123",
			"type":         "credential_vault",
			"display_name": "test-vault",
			"status":       "active",
			"credentials": []map[string]interface{}{
				{
					"id":             "cred_456",
					"mcp_server_url": "https://mcp.example.com",
					"protocol":       "http",
					"type":           "static_bearer",
					"created_at":     "2026-06-14T12:00:00Z",
				},
			},
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &CreateVaultRequest{
		DisplayName: "test-vault",
		Credentials: []CreateCredential{
			{
				MCPServerURL: "https://mcp.example.com",
				Protocol:     "http",
				Type:         "static_bearer",
				AccessToken:  "secret-token",
			},
		},
	}

	resp, err := api.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "vault_123" {
		t.Errorf("expected ID 'vault_123', got %s", resp.ID)
	}
	if resp.DisplayName != "test-vault" {
		t.Errorf("expected display_name 'test-vault', got %s", resp.DisplayName)
	}
	if len(resp.Credentials) != 1 {
		t.Fatalf("expected 1 credential in response, got %d", len(resp.Credentials))
	}
	if resp.Credentials[0].ID != "cred_456" {
		t.Errorf("expected credential ID 'cred_456', got %s", resp.Credentials[0].ID)
	}
}

func TestVault_Create_WithCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults" {
			t.Errorf("expected path /vaults, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		if reqBody["display_name"] != "my-vault" {
			t.Errorf("expected display_name 'my-vault', got %v", reqBody["display_name"])
		}

		creds, ok := reqBody["credentials"].([]interface{})
		if !ok || len(creds) == 0 {
			t.Fatal("expected credentials array with at least one element")
		}
		cred := creds[0].(map[string]interface{})
		if cred["mcp_server_url"] != "https://mcp.example.com" {
			t.Errorf("expected mcp_server_url 'https://mcp.example.com', got %v", cred["mcp_server_url"])
		}
		if cred["protocol"] != "http" {
			t.Errorf("expected protocol 'http', got %v", cred["protocol"])
		}
		if cred["type"] != "static_bearer" {
			t.Errorf("expected type 'static_bearer', got %v", cred["type"])
		}
		if cred["access_token"] != "builder-token" {
			t.Errorf("expected access_token 'builder-token', got %v", cred["access_token"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "vault_456",
			"type":         "credential_vault",
			"display_name": "my-vault",
			"status":       "active",
			"created_at":   "2026-06-14T12:00:00Z",
			"updated_at":   "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewCreateRequest("my-vault").
		WithCredential(CreateCredential{
			MCPServerURL: "https://mcp.example.com",
			Protocol:     "http",
			Type:         "static_bearer",
			AccessToken:  "builder-token",
		})

	resp, err := api.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "vault_456" {
		t.Errorf("expected ID 'vault_456', got %s", resp.ID)
	}
}

func TestVault_Create_WithIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults" {
			t.Errorf("expected path /vaults, got %s", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "vault-idem-key" {
			t.Errorf("expected Idempotency-Key 'vault-idem-key', got %q", key)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "vault_789",
			"type":         "credential_vault",
			"display_name": "idem-vault",
			"status":       "active",
			"created_at":   "2026-06-14T12:00:00Z",
			"updated_at":   "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	vault, err := api.Create(context.Background(),
		NewCreateRequest("idem-vault"),
		"vault-idem-key",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vault == nil {
		t.Fatal("expected non-nil vault")
	}
	if vault.ID != "vault_789" {
		t.Errorf("expected ID 'vault_789', got %s", vault.ID)
	}
}

func TestVault_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/vaults/vault_123" {
			t.Errorf("expected path /vaults/vault_123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "vault_123",
			"type":         "credential_vault",
			"display_name": "my-vault",
			"status":       "active",
			"credentials": []map[string]interface{}{
				{
					"id":             "cred_1",
					"mcp_server_url": "https://mcp.example.com",
					"protocol":       "http",
					"type":           "static_bearer",
					"archived":       false,
					"created_at":     "2026-06-14T12:00:00Z",
				},
			},
			"created_at": "2026-06-14T12:00:00Z",
			"updated_at": "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Get(context.Background(), "vault_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "vault_123" {
		t.Errorf("expected ID 'vault_123', got %s", resp.ID)
	}
	if resp.DisplayName != "my-vault" {
		t.Errorf("expected display_name 'my-vault', got %s", resp.DisplayName)
	}
	if len(resp.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(resp.Credentials))
	}
	if resp.Credentials[0].ID != "cred_1" {
		t.Errorf("expected credential ID 'cred_1', got %s", resp.Credentials[0].ID)
	}
	if resp.Credentials[0].MCPServerURL != "https://mcp.example.com" {
		t.Errorf("expected mcp_server_url 'https://mcp.example.com', got %s", resp.Credentials[0].MCPServerURL)
	}
}

func TestVault_Archive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults/vault_123/archive" {
			t.Errorf("expected path /vaults/vault_123/archive, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":           "vault_123",
			"type":         "credential_vault",
			"display_name": "my-vault",
			"status":       "archived",
			"archived_at":  "2026-06-14T12:00:00Z",
			"created_at":   "2026-06-14T10:00:00Z",
			"updated_at":   "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.Archive(context.Background(), "vault_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "vault_123" {
		t.Errorf("expected ID 'vault_123', got %s", resp.ID)
	}
	if resp.Status != "archived" {
		t.Errorf("expected status 'archived', got %s", resp.Status)
	}
	if resp.ArchivedAt == nil {
		t.Error("expected archived_at to be set")
	}
}

func TestVault_List(t *testing.T) {
	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/vaults" {
				t.Errorf("expected path /vaults, got %s", r.URL.Path)
			}

			if r.URL.Query().Get("limit") != "10" {
				t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("after_id") != "vault_1" {
				t.Errorf("expected after_id=vault_1, got %s", r.URL.Query().Get("after_id"))
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":           "vault_2",
						"type":         "credential_vault",
						"display_name": "vault-two",
						"status":       "active",
						"created_at":   "2026-06-14T12:00:00Z",
						"updated_at":   "2026-06-14T12:00:00Z",
					},
				},
				"first_id": "vault_2",
				"last_id":  "vault_2",
				"has_more": false,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		params := &types.ListParams{
			Limit:   10,
			AfterID: "vault_1",
		}

		resp, err := api.List(context.Background(), params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Data) != 1 {
			t.Fatalf("expected 1 vault, got %d", len(resp.Data))
		}
		if resp.Data[0].ID != "vault_2" {
			t.Errorf("expected vault ID 'vault_2', got %s", resp.Data[0].ID)
		}
		if resp.HasMore {
			t.Error("expected HasMore to be false")
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

	t.Run("nil params returns empty query", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params for nil ListParams, got %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Vault]{
				Data:    []Vault{},
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
	})
}

func TestCredential_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults/vault_123/credentials" {
			t.Errorf("expected path /vaults/vault_123/credentials, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		if reqBody["mcp_server_url"] != "https://mcp.example.com" {
			t.Errorf("expected mcp_server_url 'https://mcp.example.com', got %v", reqBody["mcp_server_url"])
		}
		if reqBody["protocol"] != "http" {
			t.Errorf("expected protocol 'http', got %v", reqBody["protocol"])
		}
		if reqBody["type"] != "static_bearer" {
			t.Errorf("expected type 'static_bearer', got %v", reqBody["type"])
		}
		if reqBody["access_token"] != "cred-token" {
			t.Errorf("expected access_token 'cred-token', got %v", reqBody["access_token"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "cred_789",
			"mcp_server_url": "https://mcp.example.com",
			"protocol":       "http",
			"type":           "static_bearer",
			"created_at":     "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := &CreateCredentialRequest{
		MCPServerURL: "https://mcp.example.com",
		Protocol:     "http",
		Type:         "static_bearer",
		AccessToken:  "cred-token",
	}

	resp, err := api.CreateCredential(context.Background(), "vault_123", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "cred_789" {
		t.Errorf("expected ID 'cred_789', got %s", resp.ID)
	}
	if resp.MCPServerURL != "https://mcp.example.com" {
		t.Errorf("expected mcp_server_url 'https://mcp.example.com', got %s", resp.MCPServerURL)
	}
}

func TestCredential_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/vaults/vault_123/credentials" {
			t.Errorf("expected path /vaults/vault_123/credentials, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %s", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":             "cred_1",
					"mcp_server_url": "https://mcp1.example.com",
					"protocol":       "http",
					"type":           "static_bearer",
					"created_at":     "2026-06-14T12:00:00Z",
				},
				{
					"id":             "cred_2",
					"mcp_server_url": "https://mcp2.example.com",
					"protocol":       "http",
					"type":           "static_bearer",
					"created_at":     "2026-06-14T13:00:00Z",
				},
			},
			"first_id": "cred_1",
			"last_id":  "cred_2",
			"has_more": true,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	params := &types.ListParams{Limit: 5}

	resp, err := api.ListCredentials(context.Background(), "vault_123", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "cred_1" {
		t.Errorf("expected first cred ID 'cred_1', got %s", resp.Data[0].ID)
	}
	if !resp.HasMore {
		t.Error("expected HasMore to be true")
	}
}

func TestCredential_List_InvalidParams(t *testing.T) {
	// No server needed - validation fails client-side before HTTP call.
	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)
	_, err := api.ListCredentials(context.Background(), "vault_123", &types.ListParams{Limit: -1})
	if err == nil {
		t.Error("expected error for invalid Limit")
	}
}

func TestCredential_List_NilParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params for nil ListParams, got %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Credential]{
			Data:    []Credential{},
			HasMore: false,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	result, err := api.ListCredentials(context.Background(), "vault_123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestCredential_Archive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/vaults/vault_123/credentials/cred_456/archive" {
			t.Errorf("expected path /vaults/vault_123/credentials/cred_456/archive, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "cred_456",
			"mcp_server_url": "https://mcp.example.com",
			"protocol":       "http",
			"type":           "static_bearer",
			"archived":       true,
			"created_at":     "2026-06-14T12:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	resp, err := api.ArchiveCredential(context.Background(), "vault_123", "cred_456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != "cred_456" {
		t.Errorf("expected ID 'cred_456', got %s", resp.ID)
	}
	if !resp.Archived {
		t.Error("expected Archived to be true")
	}
}

func TestVault_InvalidID(t *testing.T) {
	// Client-side validation rejects invalid IDs before any HTTP call.
	// The httptest server MUST never be reached — if it is, validation failed.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid vault ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	dummyCredReq := &CreateCredentialRequest{
		MCPServerURL: "https://mcp.example.com",
		Protocol:     "http",
		Type:         "static_bearer",
		AccessToken:  "token",
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Get empty ID",
			fn:   func() error { _, err := api.Get(context.Background(), ""); return err },
		},
		{
			name: "Get path traversal",
			fn:   func() error { _, err := api.Get(context.Background(), "../../etc"); return err },
		},
		{
			name: "Get encoded slash",
			fn:   func() error { _, err := api.Get(context.Background(), "%2fetc"); return err },
		},
		{
			name: "Get literal slash",
			fn:   func() error { _, err := api.Get(context.Background(), "a/b"); return err },
		},
		{
			name: "Get encoded backslash",
			fn:   func() error { _, err := api.Get(context.Background(), "%5cetc"); return err },
		},
		{
			name: "Get encoded dot",
			fn:   func() error { _, err := api.Get(context.Background(), "%2e%2e"); return err },
		},
		{
			name: "Get hash character",
			fn:   func() error { _, err := api.Get(context.Background(), "id#1"); return err },
		},
		{
			name: "Archive empty ID",
			fn:   func() error { _, err := api.Archive(context.Background(), ""); return err },
		},
		{
			name: "Archive path traversal",
			fn:   func() error { _, err := api.Archive(context.Background(), ".."); return err },
		},
		{
			name: "CreateCredential empty vaultID",
			fn: func() error {
				_, err := api.CreateCredential(context.Background(), "", dummyCredReq)
				return err
			},
		},
		{
			name: "CreateCredential invalid vaultID",
			fn: func() error {
				_, err := api.CreateCredential(context.Background(), "bad#id", dummyCredReq)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestNewStaticBearerCredential(t *testing.T) {
	cred := NewStaticBearerCredential("https://mcp.example.com", "http", "my-access-token")

	if cred.MCPServerURL != "https://mcp.example.com" {
		t.Errorf("expected MCPServerURL 'https://mcp.example.com', got %s", cred.MCPServerURL)
	}
	if cred.Protocol != "http" {
		t.Errorf("expected Protocol 'http', got %s", cred.Protocol)
	}
	if cred.Type != "static_bearer" {
		t.Errorf("expected Type 'static_bearer', got %s", cred.Type)
	}
	if cred.AccessToken != "my-access-token" {
		t.Errorf("expected AccessToken 'my-access-token', got %s", cred.AccessToken)
	}
}

func TestVault_Create_Error(t *testing.T) {
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

	_, err := api.Create(context.Background(), NewCreateRequest("test"))
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

func TestCredential_Create_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "invalid_request_error",
				"message": "vault not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	req := NewStaticBearerCredential("https://mcp.example.com", "http", "test-token")
	_, err := api.CreateCredential(context.Background(), "vault_123", &req)
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

func TestVault_Create_NilRequest(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
	if !strings.Contains(err.Error(), "must not be nil") {
		t.Errorf("expected error to mention 'must not be nil', got %q", err.Error())
	}
}

func TestCredential_Create_NilRequest(t *testing.T) {
	t.Parallel()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	// nil-request guard runs before ValidateID, so any vaultID works.
	_, err := api.CreateCredential(context.Background(), "vault_001", nil)
	if err == nil {
		t.Fatal("expected error for nil request, got nil")
	}
	if !strings.Contains(err.Error(), "must not be nil") {
		t.Errorf("expected error to mention 'must not be nil', got %q", err.Error())
	}
}
