package memorystores

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

// --- Store-level CRUD ---

func TestStore_Create(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores" {
			t.Errorf("expected path /memory_stores, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req CreateStoreRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Name != "test-store" {
			t.Errorf("expected name 'test-store', got %q", req.Name)
		}
		if req.Description != "a test store" {
			t.Errorf("expected description 'a test store', got %q", req.Description)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryStore{
			ID:          "store_001",
			Name:        req.Name,
			Description: req.Description,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	store, err := api.Create(context.Background(),
		NewCreateStoreRequest("test-store").WithDescription("a test store"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.ID != "store_001" {
		t.Errorf("expected ID 'store_001', got %q", store.ID)
	}
}

func TestStore_Create_WithIdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores" {
			t.Errorf("expected path /memory_stores, got %s", r.URL.Path)
		}
		if key := r.Header.Get("Idempotency-Key"); key != "idem-key-123" {
			t.Errorf("expected Idempotency-Key 'idem-key-123', got %q", key)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryStore{
			ID:   "store_002",
			Name: "idem-store",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	store, err := api.Create(context.Background(),
		NewCreateStoreRequest("idem-store"),
		"idem-key-123",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.ID != "store_002" {
		t.Errorf("expected ID 'store_002', got %q", store.ID)
	}
}

func TestStore_Get(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001" {
			t.Errorf("expected path /memory_stores/store_001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryStore{
			ID:          "store_001",
			Type:        "memory_store",
			Name:        "my-store",
			Description: "store desc",
			Status:      "active",
			EntryCount:  5,
			TotalSize:   1024,
			CreatedAt:   "2026-01-01T00:00:00Z",
			UpdatedAt:   "2026-06-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	store, err := api.Get(context.Background(), "store_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.ID != "store_001" {
		t.Errorf("expected ID 'store_001', got %q", store.ID)
	}
	if store.Name != "my-store" {
		t.Errorf("expected Name 'my-store', got %q", store.Name)
	}
	if store.Description != "store desc" {
		t.Errorf("expected Description 'store desc', got %q", store.Description)
	}
	if store.Status != "active" {
		t.Errorf("expected Status 'active', got %q", store.Status)
	}
	if store.EntryCount != 5 {
		t.Errorf("expected EntryCount 5, got %d", store.EntryCount)
	}
	if store.TotalSize != 1024 {
		t.Errorf("expected TotalSize 1024, got %d", store.TotalSize)
	}
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/memory_stores" {
				t.Errorf("expected path /memory_stores, got %s", r.URL.Path)
			}

			limit := r.URL.Query().Get("limit")
			if limit != "10" {
				t.Errorf("expected limit=10, got %q", limit)
			}
			afterID := r.URL.Query().Get("after_id")
			if afterID != "store_001" {
				t.Errorf("expected after_id=store_001, got %q", afterID)
			}

			w.Header().Set("Content-Type", "application/json")
			lastID := "store_003"
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[MemoryStore]{
				Data: []MemoryStore{
					{ID: "store_002", Name: "store-two"},
					{ID: "store_003", Name: "store-three"},
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
			AfterID: "store_001",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 stores, got %d", len(result.Data))
		}
		if result.Data[0].ID != "store_002" {
			t.Errorf("expected first store ID 'store_002', got %q", result.Data[0].ID)
		}
		if result.HasMore {
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
}

func TestStore_Archive(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/archive" {
			t.Errorf("expected path /memory_stores/store_001/archive, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		archivedAt := "2026-06-14T00:00:00Z"
		_ = json.NewEncoder(w).Encode(MemoryStore{
			ID:         "store_001",
			Name:       "my-store",
			Status:     "archived",
			ArchivedAt: &archivedAt,
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	store, err := api.Archive(context.Background(), "store_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.Status != "archived" {
		t.Errorf("expected Status 'archived', got %q", store.Status)
	}
	if store.ArchivedAt == nil || *store.ArchivedAt != "2026-06-14T00:00:00Z" {
		t.Errorf("expected ArchivedAt '2026-06-14T00:00:00Z', got %v", store.ArchivedAt)
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001" {
			t.Errorf("expected path /memory_stores/store_001, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	if err := api.Delete(context.Background(), "store_001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Entry-level CRUD ---

func TestEntry_Create(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/memories" {
			t.Errorf("expected path /memory_stores/store_001/memories, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req CreateEntryRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Path != "/notes/project" {
			t.Errorf("expected path '/notes/project', got %q", req.Path)
		}
		if req.Content != "remember to update docs" {
			t.Errorf("expected content 'remember to update docs', got %q", req.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryEntry{
			ID:            "entry_001",
			Path:          req.Path,
			Content:       req.Content,
			ContentSHA256: "abc123",
			Version:       1,
			Status:        "active",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	entry, err := api.CreateEntry(context.Background(), "store_001",
		NewCreateEntryRequest("/notes/project", "remember to update docs"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.ID != "entry_001" {
		t.Errorf("expected ID 'entry_001', got %q", entry.ID)
	}
	if entry.Path != "/notes/project" {
		t.Errorf("expected Path '/notes/project', got %q", entry.Path)
	}
}

func TestEntry_Get(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/memories/entry_001" {
			t.Errorf("expected path /memory_stores/store_001/memories/entry_001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryEntry{
			ID:            "entry_001",
			Path:          "/notes/project",
			Content:       "remember to update docs",
			ContentSHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			Version:       1,
			Status:        "active",
			CreatedAt:     "2026-06-01T00:00:00Z",
			UpdatedAt:     "2026-06-14T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	entry, err := api.GetEntry(context.Background(), "store_001", "entry_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Content != "remember to update docs" {
		t.Errorf("expected Content 'remember to update docs', got %q", entry.Content)
	}
	if entry.ContentSHA256 != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("expected ContentSHA256, got %q", entry.ContentSHA256)
	}
	if entry.ID != "entry_001" {
		t.Errorf("expected ID 'entry_001', got %q", entry.ID)
	}
}

func TestEntry_List(t *testing.T) {
	t.Parallel()

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/memory_stores/store_001/memories" {
				t.Errorf("expected path /memory_stores/store_001/memories, got %s", r.URL.Path)
			}

			limit := r.URL.Query().Get("limit")
			if limit != "5" {
				t.Errorf("expected limit=5, got %q", limit)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[MemoryEntry]{
				Data: []MemoryEntry{
					{ID: "entry_001", Path: "/notes/a", ContentSHA256: "a1", Version: 1, Status: "active"},
					{ID: "entry_002", Path: "/notes/b", ContentSHA256: "b1", Version: 2, Status: "active"},
				},
				HasMore: true,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		result, err := api.ListEntries(context.Background(), "store_001", &types.ListParams{Limit: 5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 entries, got %d", len(result.Data))
		}
		if !result.HasMore {
			t.Error("expected HasMore to be true")
		}
	})

	t.Run("invalid params returns error", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: "http://localhost", Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)
		_, err := api.ListEntries(context.Background(), "store_001", &types.ListParams{Limit: -1})
		if err == nil {
			t.Error("expected error for invalid Limit")
		}
	})
}

func TestEntry_Update(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/memories/entry_001" {
			t.Errorf("expected path /memory_stores/store_001/memories/entry_001, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var req UpdateEntryRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if req.Content != "updated content" {
			t.Errorf("expected content 'updated content', got %q", req.Content)
		}
		if req.ContentSHA256 != "abc123def" {
			t.Errorf("expected ContentSHA256 'abc123def', got %q", req.ContentSHA256)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryEntry{
			ID:            "entry_001",
			Path:          "/notes/project",
			Content:       req.Content,
			ContentSHA256: "newsha256",
			Version:       2,
			Status:        "active",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	entry, err := api.UpdateEntry(context.Background(), "store_001", "entry_001",
		NewUpdateEntryRequest("updated content").WithContentSHA256("abc123def"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Version != 2 {
		t.Errorf("expected Version 2, got %d", entry.Version)
	}
	if entry.ContentSHA256 != "newsha256" {
		t.Errorf("expected ContentSHA256 'newsha256', got %q", entry.ContentSHA256)
	}
}

func TestEntry_Delete(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/memories/entry_001" {
			t.Errorf("expected path /memory_stores/store_001/memories/entry_001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MemoryEntry{
			ID:            "entry_001",
			Path:          "/notes/project",
			ContentSHA256: "oldsha",
			Version:       3,
			Status:        "deleted",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	entry, err := api.DeleteEntry(context.Background(), "store_001", "entry_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.ID != "entry_001" {
		t.Errorf("expected ID 'entry_001', got %q", entry.ID)
	}
	if entry.Status != "deleted" {
		t.Errorf("expected Status 'deleted', got %q", entry.Status)
	}
}

// --- Version operations ---

func TestVersion_List(t *testing.T) {
	t.Parallel()

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/memory_stores/store_001/versions" {
				t.Errorf("expected path /memory_stores/store_001/versions, got %s", r.URL.Path)
			}

			limit := r.URL.Query().Get("limit")
			if limit != "20" {
				t.Errorf("expected limit=20, got %q", limit)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Version]{
				Data: []Version{
					{ID: "ver_002", EntryID: "entry_001", ContentSHA256: "c2", Version: 2, Action: "update", Redacted: false},
					{ID: "ver_001", EntryID: "entry_001", ContentSHA256: "c1", Version: 1, Action: "create", Redacted: false},
				},
				HasMore: false,
			})
		}))
		defer srv.Close()

		c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
		api := NewAPI(c)

		result, err := api.ListVersions(context.Background(), "store_001", &types.ListParams{Limit: 20})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 versions, got %d", len(result.Data))
		}
		if result.Data[0].Action != "update" {
			t.Errorf("expected first version Action 'update', got %q", result.Data[0].Action)
		}
		if result.HasMore {
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
}

func TestVersion_Get(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/versions/ver_001" {
			t.Errorf("expected path /memory_stores/store_001/versions/ver_001, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{
			ID:            "ver_001",
			EntryID:       "entry_001",
			Content:       "original content",
			ContentSHA256: "sha256hash",
			Version:       1,
			Action:        "create",
			Redacted:      false,
			CreatedAt:     "2026-06-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	version, err := api.GetVersion(context.Background(), "store_001", "ver_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version == nil {
		t.Fatal("expected non-nil version")
	}
	if version.ID != "ver_001" {
		t.Errorf("expected ID 'ver_001', got %q", version.ID)
	}
	if version.Redacted {
		t.Error("expected Redacted to be false")
	}
	if version.EntryID != "entry_001" {
		t.Errorf("expected EntryID 'entry_001', got %q", version.EntryID)
	}
	if version.Action != "create" {
		t.Errorf("expected Action 'create', got %q", version.Action)
	}
}

func TestVersion_Redact(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/memory_stores/store_001/versions/ver_001/redact" {
			t.Errorf("expected path /memory_stores/store_001/versions/ver_001/redact, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{
			ID:            "ver_001",
			EntryID:       "entry_001",
			ContentSHA256: "sha256hash",
			Version:       1,
			Action:        "create",
			Redacted:      true,
			CreatedAt:     "2026-06-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	version, err := api.RedactVersion(context.Background(), "store_001", "ver_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version == nil {
		t.Fatal("expected non-nil version")
	}
	if !version.Redacted {
		t.Error("expected Redacted to be true after redact")
	}
}

// --- Invalid ID error paths ---

func TestStore_InvalidID(t *testing.T) {
	// Table-driven test: invalid IDs for store-level Get, Delete, Archive.
	// Validation happens client-side, so the HTTP handler must NOT be called.

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid store ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)
	ctx := context.Background()

	tests := []struct {
		name string
		id   string
		fn   func(id string) error
	}{
		{name: "Get_empty", id: "", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_slash", id: "a/b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_dotdot", id: "a..b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Get_space", id: "a b", fn: func(id string) error { _, err := api.Get(ctx, id); return err }},
		{name: "Delete_empty", id: "", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "Delete_slash", id: "a/b", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "Delete_dotdot", id: "a..b", fn: func(id string) error { return api.Delete(ctx, id) }},
		{name: "Archive_empty", id: "", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
		{name: "Archive_slash", id: "a/b", fn: func(id string) error { _, err := api.Archive(ctx, id); return err }},
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

func TestEntry_InvalidID(t *testing.T) {
	// Table-driven test: invalid storeID / entryID for entry-level methods.
	// Validation happens client-side, so the HTTP handler must NOT be called.

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid entry ID")
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)
	ctx := context.Background()
	validEntryReq := NewCreateEntryRequest("/notes/x", "hello")
	validUpdateReq := NewUpdateEntryRequest("new content")

	tests := []struct {
		name    string
		storeID string
		entryID string
		fn      func() error
	}{
		// Invalid storeID
		{name: "ListEntries_emptyStore", storeID: "", entryID: "", fn: func() error { _, err := api.ListEntries(ctx, "", &types.ListParams{}); return err }},
		{name: "CreateEntry_emptyStore", storeID: "", entryID: "", fn: func() error { _, err := api.CreateEntry(ctx, "", validEntryReq); return err }},
		{name: "GetEntry_emptyStore", storeID: "", entryID: "entry_001", fn: func() error { _, err := api.GetEntry(ctx, "", "entry_001"); return err }},
		{name: "UpdateEntry_emptyStore", storeID: "", entryID: "entry_001", fn: func() error { _, err := api.UpdateEntry(ctx, "", "entry_001", validUpdateReq); return err }},
		{name: "DeleteEntry_emptyStore", storeID: "", entryID: "entry_001", fn: func() error { _, err := api.DeleteEntry(ctx, "", "entry_001"); return err }},
		// Invalid storeID with bad characters
		{name: "GetEntry_slashStore", storeID: "a/b", entryID: "entry_001", fn: func() error { _, err := api.GetEntry(ctx, "a/b", "entry_001"); return err }},
		{name: "UpdateEntry_dotdotStore", storeID: "a..b", entryID: "entry_001", fn: func() error { _, err := api.UpdateEntry(ctx, "a..b", "entry_001", validUpdateReq); return err }},
		// Invalid entryID
		{name: "GetEntry_emptyEntry", storeID: "store_001", entryID: "", fn: func() error { _, err := api.GetEntry(ctx, "store_001", ""); return err }},
		{name: "UpdateEntry_slashEntry", storeID: "store_001", entryID: "a/b", fn: func() error { _, err := api.UpdateEntry(ctx, "store_001", "a/b", validUpdateReq); return err }},
		{name: "DeleteEntry_dotdotEntry", storeID: "store_001", entryID: "a..b", fn: func() error { _, err := api.DeleteEntry(ctx, "store_001", "a..b"); return err }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("expected error for invalid ID (store=%q, entry=%q), got nil", tt.storeID, tt.entryID)
			}
		})
	}
}

// --- Builder tests ---

func TestCreateStoreRequest_Builder(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"env": "production", "team": "platform"}

	req := NewCreateStoreRequest("my-store").
		WithDescription("store description").
		WithMetadata(meta)

	if req.Name != "my-store" {
		t.Errorf("expected Name 'my-store', got %q", req.Name)
	}
	if req.Description != "store description" {
		t.Errorf("expected Description 'store description', got %q", req.Description)
	}
	if len(req.Metadata) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(req.Metadata))
	}
	if req.Metadata["env"] != "production" {
		t.Errorf("expected metadata env='production', got %q", req.Metadata["env"])
	}
	if req.Metadata["team"] != "platform" {
		t.Errorf("expected metadata team='platform', got %q", req.Metadata["team"])
	}

	// Verify chaining returns the same request.
	req2 := NewCreateStoreRequest("x")
	if req2.WithDescription("d") != req2 {
		t.Error("WithDescription should return the same pointer for chaining")
	}
	if req2.WithMetadata(nil) != req2 {
		t.Error("WithMetadata should return the same pointer for chaining")
	}
}

func TestUpdateEntryRequest_Builder(t *testing.T) {
	t.Parallel()

	meta := types.Metadata{"source": "api"}

	req := NewUpdateEntryRequest("new content").
		WithContentSHA256("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855").
		WithMetadata(meta)

	if req.Content != "new content" {
		t.Errorf("expected Content 'new content', got %q", req.Content)
	}
	if req.ContentSHA256 != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("expected ContentSHA256, got %q", req.ContentSHA256)
	}
	if len(req.Metadata) != 1 {
		t.Errorf("expected 1 metadata entry, got %d", len(req.Metadata))
	}
	if req.Metadata["source"] != "api" {
		t.Errorf("expected metadata source='api', got %q", req.Metadata["source"])
	}

	// Verify chaining returns the same request.
	req2 := NewUpdateEntryRequest("x")
	if req2.WithContentSHA256("abc") != req2 {
		t.Error("WithContentSHA256 should return the same pointer for chaining")
	}
	if req2.WithMetadata(nil) != req2 {
		t.Error("WithMetadata should return the same pointer for chaining")
	}
}

// --- HTTP error paths ---

func TestStore_Create_Error(t *testing.T) {
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

	_, err := api.Create(context.Background(), NewCreateStoreRequest("test-store"))
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

func TestEntry_Get_Error(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "not_found_error",
				"message": "entry not found",
			},
		})
	}))
	defer srv.Close()

	c := qoderhttp.NewClient(&qoderhttp.Config{BaseURL: srv.URL, Token: "test-token", Timeout: 5 * time.Second})
	api := NewAPI(c)

	_, err := api.GetEntry(context.Background(), "store_001", "entry_001")
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
