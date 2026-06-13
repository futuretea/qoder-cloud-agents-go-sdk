package sessions

import (
	"encoding/json"
	"testing"
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
