package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
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
