//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
)

func TestE2ESessions(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c := newClient(t)
	modelID := pickEnabledModel(t)

	// Create environment.
	envName := newE2EResourceName(t, "environment")
	env, err := createWithRetryValue(ctx, func() (*environments.Environment, error) {
		return c.Environments().Create(ctx, environments.NewCreateRequest(envName, environments.EnvConfig{
			Type: "cloud",
			Networking: environments.Networking{
				Type: "unrestricted",
			},
		}))
	})
	if err != nil {
		t.Fatalf("failed to create environment for session: %v", redact(err.Error()))
	}
	recordResource(t, "environment", env.ID, env.Name)
	t.Cleanup(func() { safeDelete(t, "environment", env.ID, env.Name) })

	// Create agent.
	agentName := newE2EResourceName(t, "agent")
	agent, err := createWithRetryValue(ctx, func() (*agents.Agent, error) {
		return c.Agents().Create(ctx, agents.NewCreateRequest(agentName, modelID).WithDescription("e2e agent for session"))
	})
	if err != nil {
		t.Fatalf("failed to create agent for session: %v", redact(err.Error()))
	}
	recordResource(t, "agent", agent.ID, agent.Name)
	t.Cleanup(func() { safeDelete(t, "agent", agent.ID, agent.Name) })

	// Create session.
	sessionTitle := newE2EResourceName(t, "session")
	session, err := createWithRetryValue(ctx, func() (*sessions.Session, error) {
		return c.Sessions().Create(ctx, sessions.NewCreateRequest(agent.ID).
			WithEnvironment(env.ID).
			WithTitle(sessionTitle))
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", redact(err.Error()))
	}
	recordResource(t, "session", session.ID, sessionTitle)
	t.Cleanup(func() { safeDelete(t, "session", session.ID, sessionTitle) })

	if session.Title != sessionTitle {
		t.Fatalf("session title mismatch: got %q, want %q", session.Title, sessionTitle)
	}
	if session.EnvironmentID != env.ID {
		t.Fatalf("session environment_id mismatch: got %q, want %q", session.EnvironmentID, env.ID)
	}

	got, err := c.Sessions().Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to get session: %v", redact(err.Error()))
	}
	if got.ID != session.ID {
		t.Fatalf("session id mismatch: got %q, want %q", got.ID, session.ID)
	}

	list, err := c.Sessions().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list sessions: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("sessions list is empty")
	}

	cancelResp, err := c.Sessions().Cancel(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to cancel session: %v", redact(err.Error()))
	}
	if cancelResp.Status == "" {
		t.Fatal("cancel response status is empty")
	}

	got2, err := c.Sessions().Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to get session after cancel: %v", redact(err.Error()))
	}
	if got2.Status == "" {
		t.Fatal("session status is empty after cancel")
	}

	t.Logf("session lifecycle passed: %s", session.ID)
}
