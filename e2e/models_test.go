//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"
)

func TestE2EModels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := newClient(t)
	models, err := c.Models().List(ctx)
	if err != nil {
		t.Fatalf("failed to list models: %v", redact(err.Error()))
	}
	if len(models) == 0 {
		t.Fatal("no models returned")
	}

	var enabled []string
	for _, m := range models {
		if m.IsEnabled {
			enabled = append(enabled, m.ID)
		}
	}
	if len(enabled) == 0 {
		t.Skip("no enabled models found")
	}

	t.Logf("found %d model(s), %d enabled", len(models), len(enabled))
}
