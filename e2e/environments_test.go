//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
)

func TestE2EEnvironments(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	name := newE2EResourceName(t, "environment")

	env, err := createWithRetryValue(ctx, func() (*environments.Environment, error) {
		return c.Environments().Create(ctx, environments.NewCreateRequest(name, environments.EnvConfig{
			Type: "cloud",
			Networking: environments.Networking{
				Type: "unrestricted",
			},
		}).WithDescription("e2e environment"))
	})
	if err != nil {
		t.Fatalf("failed to create environment: %v", redact(err.Error()))
	}
	recordResource(t, "environment", env.ID, env.Name)
	t.Cleanup(func() { safeDelete(t, "environment", env.ID, env.Name) })

	if env.Name != name {
		t.Fatalf("environment name mismatch: got %q, want %q", env.Name, name)
	}

	got, err := c.Environments().Get(ctx, env.ID)
	if err != nil {
		t.Fatalf("failed to get environment: %v", redact(err.Error()))
	}
	if got.ID != env.ID {
		t.Fatalf("environment id mismatch: got %q, want %q", got.ID, env.ID)
	}

	list, err := c.Environments().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list environments: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("environments list is empty")
	}

	updatedName := name + "-updated"
	updated, err := c.Environments().Update(ctx, env.ID, environments.NewUpdateRequest().WithName(updatedName).WithDescription("updated by e2e"))
	if err != nil {
		t.Fatalf("failed to update environment: %v", redact(err.Error()))
	}
	if updated.Name != updatedName {
		t.Fatalf("updated environment name mismatch: got %q, want %q", updated.Name, updatedName)
	}

	got2, err := c.Environments().Get(ctx, env.ID)
	if err != nil {
		t.Fatalf("failed to get environment after update: %v", redact(err.Error()))
	}
	if got2.Name != updatedName {
		t.Fatalf("environment name after update mismatch: got %q, want %q", got2.Name, updatedName)
	}
	if got2.Description != "updated by e2e" {
		t.Fatalf("environment description after update mismatch: got %q, want %q", got2.Description, "updated by e2e")
	}

	t.Logf("environment lifecycle passed: %s", env.ID)
}
