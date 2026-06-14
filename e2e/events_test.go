//go:build e2e

package e2e

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
)

func TestE2EEvents(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Events streaming may hold the connection open; use a longer HTTP timeout.
	c := newClientWithTimeout(t, 5*time.Minute)
	modelID := pickEnabledModel(t)

	// Create environment.
	envName := newE2EResourceName(t, "environment")
	env, err := c.Environments().Create(ctx, environments.NewCreateRequest(envName, environments.EnvConfig{
		Type: "cloud",
		Networking: environments.Networking{
			Type: "unrestricted",
		},
	}))
	if err != nil {
		t.Fatalf("failed to create environment for events: %v", redact(err.Error()))
	}
	recordResource(t, "environment", env.ID, env.Name)
	t.Cleanup(func() { safeDelete(t, "environment", env.ID, env.Name) })

	// Create agent.
	agentName := newE2EResourceName(t, "agent")
	agent, err := c.Agents().Create(ctx, agents.NewCreateRequest(agentName, modelID).WithDescription("e2e agent for events"))
	if err != nil {
		t.Fatalf("failed to create agent for events: %v", redact(err.Error()))
	}
	recordResource(t, "agent", agent.ID, agent.Name)
	t.Cleanup(func() { safeDelete(t, "agent", agent.ID, agent.Name) })

	// Create session.
	sessionTitle := newE2EResourceName(t, "session")
	session, err := c.Sessions().Create(ctx, sessions.NewCreateRequest(agent.ID).
		WithEnvironment(env.ID).
		WithTitle(sessionTitle))
	if err != nil {
		t.Fatalf("failed to create session for events: %v", redact(err.Error()))
	}
	recordResource(t, "session", session.ID, sessionTitle)
	t.Cleanup(func() { safeDelete(t, "session", session.ID, sessionTitle) })

	// Send a short user message.
	if err := c.Events().SendMessage(ctx, session.ID, "hi"); err != nil {
		t.Fatalf("failed to send message: %v", redact(err.Error()))
	}

	// Stream events with a bounded read.
	resp, err := c.Events().Stream(ctx, session.ID)
	if err != nil {
		t.Fatalf("failed to open event stream: %v", redact(err.Error()))
	}
	defer closeOrLog(t, resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("unexpected stream status code: got %d, want 200", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("unexpected stream content type: got %q, want text/event-stream", contentType)
	}

	stream := qoderhttp.NewSSEStream(resp)
	defer closeOrLog(t, stream)

	readCtx, readCancel := context.WithTimeout(ctx, eventsReadTimeout)
	defer readCancel()

	eventCount := 0
	for {
		evt, err := stream.Next(readCtx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				break
			}
			t.Fatalf("stream read failed: %v", redact(err.Error()))
		}
		if evt == nil {
			continue
		}
		if len(evt.Data) == 0 {
			t.Fatalf("event data is empty at event %d", eventCount)
		}
		eventCount++
		if eventCount >= 5 {
			break
		}
	}

	if eventCount == 0 {
		t.Fatalf("no events read before timeout; connection was established but stream delivered zero events")
	}

	t.Logf("event stream passed: read %d event(s)", eventCount)
}
