package skills

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

// skillResponse returns a sample Skill used in handler responses.
func skillResponse() Skill {
	return Skill{
		ID:          "skill_abc123",
		Type:        "custom",
		Name:        "my-skill",
		Description: "A test skill",
		SkillType:   "tool",
		ContentSize: 1024,
		Version:     1,
		Status:      "ready",
		CreatedAt:   "2026-06-13T12:00:00Z",
		UpdatedAt:   "2026-06-13T12:00:00Z",
	}
}

// newTestClientAndAPI creates a Skills API backed by the given httptest.Server URL.
func newTestClientAndAPI(baseURL string) *API {
	c := qoderhttp.NewClient(&qoderhttp.Config{
		BaseURL: baseURL,
		Token:   "test-token",
		Timeout: 5 * time.Second,
	})
	return NewAPI(c)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestSkill_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/skills" {
			t.Errorf("expected path /skills, got %s", r.URL.Path)
		}

		// Parse multipart form (10 MB max).
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify the file field.
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("expected file field: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() { _ = file.Close() }()

		content, _ := io.ReadAll(file)
		if string(content) != "zip-bytes-here" {
			t.Errorf("unexpected file content: %s", string(content))
		}
		if header.Filename != "mypackage.zip" {
			t.Errorf("expected filename mypackage.zip, got %s", header.Filename)
		}

		// Verify the type form field.
		if r.FormValue("type") != "custom" {
			t.Errorf("expected type=custom, got %s", r.FormValue("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skillResponse())
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	skill, err := api.Create(context.Background(), &CreateSkillRequest{
		Filename: "mypackage.zip",
		Data:     []byte("zip-bytes-here"),
		Type:     "custom",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected non-nil skill")
	}
	if skill.ID != "skill_abc123" {
		t.Errorf("expected ID skill_abc123, got %s", skill.ID)
	}
	if skill.Name != "my-skill" {
		t.Errorf("expected Name my-skill, got %s", skill.Name)
	}
	if skill.Status != "ready" {
		t.Errorf("expected Status ready, got %s", skill.Status)
	}
}

func TestSkill_Create_WithTypePrebuilt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		if r.FormValue("type") != "prebuilt" {
			t.Errorf("expected type=prebuilt, got %s", r.FormValue("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skillResponse())
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	skill, err := api.Create(context.Background(), &CreateSkillRequest{
		Filename: "prebuilt.zip",
		Data:     []byte("prebuilt-content"),
		Type:     "prebuilt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected non-nil skill")
	}
}

func TestSkill_Create_WithIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "my-key-001" {
			t.Errorf("expected Idempotency-Key=my-key-001, got %s", r.Header.Get("Idempotency-Key"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skillResponse())
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	skill, err := api.Create(context.Background(), &CreateSkillRequest{
		Filename: "mypackage.zip",
		Data:     []byte("zip-bytes-here"),
		Type:     "custom",
	}, "my-key-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected non-nil skill")
	}
}

func TestSkill_Create_EmptyIdempotencyKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "" {
			t.Errorf("expected no Idempotency-Key header for empty key, got %q", r.Header.Get("Idempotency-Key"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skillResponse())
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	skill, err := api.Create(context.Background(), &CreateSkillRequest{
		Filename: "mypackage.zip",
		Data:     []byte("zip-bytes-here"),
		Type:     "custom",
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected non-nil skill")
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestSkill_Get(t *testing.T) {
	t.Run("includeContent=false", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/skills/skill_abc123"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}
			// When includeContent is false, the query param should be absent.
			if r.URL.Query().Has("include_content") {
				t.Errorf("expected no include_content query param, got %s", r.URL.Query().Get("include_content"))
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(skillResponse())
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		skill, err := api.Get(context.Background(), "skill_abc123", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skill == nil {
			t.Fatal("expected non-nil skill")
		}
		if skill.ID != "skill_abc123" {
			t.Errorf("expected ID skill_abc123, got %s", skill.ID)
		}
		if skill.Content != "" {
			t.Errorf("expected empty Content when includeContent is false, got %s", skill.Content)
		}
	})

	t.Run("includeContent=true", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Query().Get("include_content") != "true" {
				t.Errorf("expected include_content=true, got %s", r.URL.Query().Get("include_content"))
			}

			resp := skillResponse()
			resp.Content = "YmFzZTY0LWVuY29kZWQtY29udGVudA=="
			resp.ContentEncoding = "base64"

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		skill, err := api.Get(context.Background(), "skill_abc123", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skill == nil {
			t.Fatal("expected non-nil skill")
		}
		if skill.Content == "" {
			t.Error("expected non-empty Content when includeContent is true")
		}
		if skill.ContentEncoding != "base64" {
			t.Errorf("expected ContentEncoding base64, got %s", skill.ContentEncoding)
		}
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestSkill_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/skills/skill_abc123"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Parse JSON body.
		var body UpdateSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode JSON body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Name != "updated-name" {
			t.Errorf("expected Name=updated-name, got %s", body.Name)
		}
		if body.Description != "updated-description" {
			t.Errorf("expected Description=updated-description, got %s", body.Description)
		}

		resp := skillResponse()
		resp.Name = "updated-name"
		resp.Description = "updated-description"

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	updated, err := api.Update(context.Background(), "skill_abc123",
		NewUpdateRequest().WithName("updated-name").WithDescription("updated-description"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected non-nil skill")
	}
	if updated.Name != "updated-name" {
		t.Errorf("expected Name updated-name, got %s", updated.Name)
	}
	if updated.Description != "updated-description" {
		t.Errorf("expected Description updated-description, got %s", updated.Description)
	}
}

func TestSkill_Update_NameOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body UpdateSkillRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode JSON body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body.Name != "only-name-update" {
			t.Errorf("expected Name=only-name-update, got %s", body.Name)
		}
		if body.Description != "" {
			t.Errorf("expected Description to be empty, got %s", body.Description)
		}

		resp := skillResponse()
		resp.Name = "only-name-update"

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	updated, err := api.Update(context.Background(), "skill_abc123",
		NewUpdateRequest().WithName("only-name-update"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected non-nil skill")
	}
	if updated.Name != "only-name-update" {
		t.Errorf("expected Name only-name-update, got %s", updated.Name)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestSkill_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/skills/skill_abc123"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	if err := api.Delete(context.Background(), "skill_abc123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestSkill_List(t *testing.T) {
	t.Run("no params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/skills" {
				t.Errorf("expected path /skills, got %s", r.URL.Path)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("expected no query params, got %s", r.URL.RawQuery)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Skill]{
				Data:    []Skill{skillResponse()},
				HasMore: false,
			})
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		resp, err := api.List(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Data) != 1 {
			t.Errorf("expected 1 skill, got %d", len(resp.Data))
		}
		if resp.HasMore {
			t.Error("expected HasMore to be false")
		}
		if resp.Data[0].ID != "skill_abc123" {
			t.Errorf("expected ID skill_abc123, got %s", resp.Data[0].ID)
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("limit") != "10" {
				t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("after_id") != "cursor_001" {
				t.Errorf("expected after_id=cursor_001, got %s", r.URL.Query().Get("after_id"))
			}

			w.Header().Set("Content-Type", "application/json")
			lastID := "cursor_001"
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Skill]{
				Data:    []Skill{skillResponse()},
				LastID:  &lastID,
				HasMore: true,
			})
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		wantLastID := "cursor_001"
		resp, err := api.List(context.Background(), &types.ListParams{
			Limit:   10,
			AfterID: "cursor_001",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.LastID == nil || *resp.LastID != wantLastID {
			t.Errorf("expected LastID to be %q, got %v", wantLastID, resp.LastID)
		}
	})

	t.Run("invalid params returns error", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		api := newTestClientAndAPI("http://localhost")
		_, err := api.List(context.Background(), &types.ListParams{Limit: -1})
		if err == nil {
			t.Error("expected error for invalid Limit")
		}
	})
}

// ---------------------------------------------------------------------------
// ListVersions
// ---------------------------------------------------------------------------

func TestSkill_ListVersions(t *testing.T) {
	t.Run("no params", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/skills/skill_abc123/versions"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			v1 := skillResponse()
			v2 := skillResponse()
			v2.Version = 2

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Skill]{
				Data:    []Skill{v2, v1},
				HasMore: false,
			})
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		resp, err := api.ListVersions(context.Background(), "skill_abc123", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if len(resp.Data) != 2 {
			t.Errorf("expected 2 versions, got %d", len(resp.Data))
		}
		if resp.Data[0].Version != 2 {
			t.Errorf("expected first version to be 2, got %d", resp.Data[0].Version)
		}
		if resp.HasMore {
			t.Error("expected HasMore to be false")
		}
	})

	t.Run("with pagination", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/skills/skill_abc123/versions"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}
			if r.URL.Query().Get("limit") != "5" {
				t.Errorf("expected limit=5, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("before_id") != "v3" {
				t.Errorf("expected before_id=v3, got %s", r.URL.Query().Get("before_id"))
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(types.PaginatedResponse[Skill]{
				Data:    []Skill{skillResponse()},
				HasMore: true,
			})
		}))
		defer srv.Close()

		api := newTestClientAndAPI(srv.URL)

		resp, err := api.ListVersions(context.Background(), "skill_abc123", &types.ListParams{
			Limit:    5,
			BeforeID: "v3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if !resp.HasMore {
			t.Error("expected HasMore to be true")
		}
	})

	t.Run("invalid params returns error", func(t *testing.T) {
		// No server needed - validation fails client-side before HTTP call.
		api := newTestClientAndAPI("http://localhost")
		_, err := api.ListVersions(context.Background(), "skill_abc123", &types.ListParams{Limit: 200})
		if err == nil {
			t.Error("expected error for invalid Limit > 100")
		}
	})
}

// ---------------------------------------------------------------------------
// Invalid ID
// ---------------------------------------------------------------------------

func TestSkill_InvalidID(t *testing.T) {
	// ValidateID should reject before any HTTP call. Use httptest so that if
	// a call does slip through, the test fails loudly instead of hitting a
	// live endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP call for invalid skill ID")
	}))
	defer srv.Close()
	c := qoderhttp.NewClient(&qoderhttp.Config{
		BaseURL: srv.URL,
		Token:   "test-token",
		Timeout: 5 * time.Second,
	})
	api := NewAPI(c)

	invalidIDs := []string{
		"",
		"/",
		"a/b",
		"a\\b",
		"..",
		"a..b",
		"%2F",
		"a%2fb",
		"a%2e%2eb",
	}

	t.Run("Get", func(t *testing.T) {
		for _, id := range invalidIDs {
			_, err := api.Get(context.Background(), id, false)
			if err == nil {
				t.Errorf("expected error for ID %q", id)
			}
		}
	})

	t.Run("Update", func(t *testing.T) {
		for _, id := range invalidIDs {
			_, err := api.Update(context.Background(), id, NewUpdateRequest().WithName("x"))
			if err == nil {
				t.Errorf("expected error for ID %q", id)
			}
		}
	})

	t.Run("Delete", func(t *testing.T) {
		for _, id := range invalidIDs {
			err := api.Delete(context.Background(), id)
			if err == nil {
				t.Errorf("expected error for ID %q", id)
			}
		}
	})

	t.Run("ListVersions", func(t *testing.T) {
		for _, id := range invalidIDs {
			_, err := api.ListVersions(context.Background(), id, nil)
			if err == nil {
				t.Errorf("expected error for ID %q", id)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Builder
// ---------------------------------------------------------------------------

func TestUpdateSkillRequest_Builder(t *testing.T) {
	t.Run("chain", func(t *testing.T) {
		req := NewUpdateRequest().
			WithName("my-skill").
			WithDescription("A skill description")

		if req.Name != "my-skill" {
			t.Errorf("expected Name=my-skill, got %s", req.Name)
		}
		if req.Description != "A skill description" {
			t.Errorf("expected Description='A skill description', got %s", req.Description)
		}
	})

	t.Run("marshal JSON", func(t *testing.T) {
		req := NewUpdateRequest().
			WithName("json-name").
			WithDescription("json-desc")

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		expected := `{"name":"json-name","description":"json-desc"}`
		if string(data) != expected {
			t.Errorf("expected JSON %s, got %s", expected, string(data))
		}
	})

	t.Run("marshal JSON omitempty", func(t *testing.T) {
		req := NewUpdateRequest().WithName("name-only")
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		expected := `{"name":"name-only"}`
		if string(data) != expected {
			t.Errorf("expected JSON %s, got %s", expected, string(data))
		}

		req2 := NewUpdateRequest().WithDescription("desc-only")
		data2, err := json.Marshal(req2)
		if err != nil {
			t.Fatalf("unexpected marshal error: %v", err)
		}
		expected2 := `{"description":"desc-only"}`
		if string(data2) != expected2 {
			t.Errorf("expected JSON %s, got %s", expected2, string(data2))
		}
	})

	t.Run("existing test", func(t *testing.T) {
		// Legacy test from original codebase; keep for backwards-compat confidence.
		req := NewUpdateRequest().WithName("test-skill").WithDescription("updated description")
		body, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		bodyStr := string(body)
		if !strings.Contains(bodyStr, `"name":"test-skill"`) {
			t.Errorf("expected name field in JSON, got %s", bodyStr)
		}
		if !strings.Contains(bodyStr, `"description":"updated description"`) {
			t.Errorf("expected description field in JSON, got %s", bodyStr)
		}
	})
}

// ---------------------------------------------------------------------------
// Create — Large file content
// ---------------------------------------------------------------------------

func TestSkill_Create_LargeFile(t *testing.T) {
	// Use a moderately large payload to verify streaming integrity.
	content := make([]byte, 1024*1024) // 1 MB
	for i := range content {
		content[i] = byte(i % 256)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("file field missing: %v", err)
		}
		defer func() { _ = file.Close() }()

		received, _ := io.ReadAll(file)
		if len(received) != len(content) {
			t.Errorf("expected %d bytes, got %d", len(content), len(received))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skillResponse())
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	skill, err := api.Create(context.Background(), &CreateSkillRequest{
		Filename: "large.zip",
		Data:     content,
		Type:     "custom",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected non-nil skill")
	}
}

// ---------------------------------------------------------------------------
// HTTP error paths
// ---------------------------------------------------------------------------

func TestSkill_Get_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "not_found_error",
				"message": "skill not found",
			},
		})
	}))
	defer srv.Close()
	api := newTestClientAndAPI(srv.URL)
	_, err := api.Get(context.Background(), "skill_abc123", false)
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

func TestSkill_Update_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "server_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()
	api := newTestClientAndAPI(srv.URL)
	_, err := api.Update(context.Background(), "skill_abc123", NewUpdateRequest().WithName("test"))
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

func TestSkill_Create_NilRequest(t *testing.T) {
	api := newTestClientAndAPI("http://localhost")

	_, err := api.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil CreateSkillRequest")
	}
	if err.Error() != "skills: CreateSkillRequest must not be nil" {
		t.Errorf("expected nil request error, got %q", err.Error())
	}
}

func TestSkill_Update_NilRequest(t *testing.T) {
	api := newTestClientAndAPI("http://localhost")

	_, err := api.Update(context.Background(), "skill_abc123", nil)
	if err == nil {
		t.Fatal("expected error for nil UpdateSkillRequest")
	}
	if err.Error() != "skills: UpdateSkillRequest must not be nil" {
		t.Errorf("expected nil request error, got %q", err.Error())
	}
}

func TestCreate_EmptyFilename(t *testing.T) {
	t.Parallel()

	api := newTestClientAndAPI("http://localhost")

	_, err := api.Create(context.Background(), &CreateSkillRequest{Filename: "", Data: []byte("x"), Type: "custom"})
	if err == nil {
		t.Fatal("expected error for empty Filename, got nil")
	}
	if err.Error() != "skills: CreateSkillRequest.Filename is required" {
		t.Errorf("expected 'skills: CreateSkillRequest.Filename is required', got %q", err.Error())
	}
}

func TestCreate_NilData(t *testing.T) {
	t.Parallel()

	api := newTestClientAndAPI("http://localhost")

	_, err := api.Create(context.Background(), &CreateSkillRequest{Filename: "x.zip", Data: nil, Type: "custom"})
	if err == nil {
		t.Fatal("expected error for nil Data, got nil")
	}
	if err.Error() != "skills: CreateSkillRequest.Data is required" {
		t.Errorf("expected 'skills: CreateSkillRequest.Data is required', got %q", err.Error())
	}
}

func TestSkill_Delete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "server_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	err := api.Delete(context.Background(), "skill_123")
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

func TestSkill_List_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "server_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	_, err := api.List(context.Background(), nil)
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

func TestSkill_Create_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "server_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	_, err := api.Create(context.Background(), &CreateSkillRequest{Filename: "skill.zip", Data: []byte("data"), Type: "custom"})
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

func TestSkill_ListVersions_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "error",
			"error": map[string]string{
				"type":    "server_error",
				"message": "internal server error",
			},
		})
	}))
	defer srv.Close()

	api := newTestClientAndAPI(srv.URL)

	_, err := api.ListVersions(context.Background(), "skill_123", nil)
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
