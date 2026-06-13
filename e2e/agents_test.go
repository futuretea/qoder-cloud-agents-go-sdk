//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
)

func TestE2EAgents(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	modelID := pickEnabledModel(t)
	name := newE2EResourceName(t, "agent")

	agent, err := createWithRetryValue(ctx, func() (*agents.Agent, error) {
		return c.Agents().Create(ctx, agents.NewCreateRequest(name, modelID).WithDescription("e2e agent"))
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", redact(err.Error()))
	}
	recordResource(t, "agent", agent.ID, agent.Name)
	t.Cleanup(func() { safeDelete(t, "agent", agent.ID, agent.Name) })

	if agent.Name != name {
		t.Fatalf("agent name mismatch: got %q, want %q", agent.Name, name)
	}
	if agent.Model != modelID {
		t.Fatalf("agent model mismatch: got %q, want %q", agent.Model, modelID)
	}

	got, err := c.Agents().Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("failed to get agent: %v", redact(err.Error()))
	}
	if got.ID != agent.ID {
		t.Fatalf("agent id mismatch: got %q, want %q", got.ID, agent.ID)
	}

	list, err := c.Agents().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list agents: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("agents list is empty")
	}

	updatedName := name + "-updated"
	updated, err := c.Agents().Update(ctx, agent.ID, agents.NewUpdateRequest(got.Version).WithName(updatedName).WithDescription("updated by e2e"))
	if err != nil {
		t.Fatalf("failed to update agent: %v", redact(err.Error()))
	}
	if updated.Name != updatedName {
		t.Fatalf("updated agent name mismatch: got %q, want %q", updated.Name, updatedName)
	}

	got2, err := c.Agents().Get(ctx, agent.ID)
	if err != nil {
		t.Fatalf("failed to get agent after update: %v", redact(err.Error()))
	}
	if got2.Name != updatedName {
		t.Fatalf("agent name after update mismatch: got %q, want %q", got2.Name, updatedName)
	}
	if got2.Description != "updated by e2e" {
		t.Fatalf("agent description after update mismatch: got %q, want %q", got2.Description, "updated by e2e")
	}

	t.Logf("agent lifecycle passed: %s", agent.ID)
}
