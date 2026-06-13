//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/vaults"
)

func TestE2EVaults(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	displayName := newE2EResourceName(t, "vault")

	vault, err := createWithRetryValue(ctx, func() (*vaults.Vault, error) {
		return c.Vaults().Create(ctx, vaults.NewCreateRequest(displayName).
			WithCredential(vaults.CreateCredential{
				MCPServerURL: "https://example.com/mcp",
				Protocol:     "sse",
				Type:         "static_bearer",
				AccessToken:  "e2e-token-" + randomHex(8),
			}))
	})
	if err != nil {
		t.Fatalf("failed to create vault: %v", redact(err.Error()))
	}
	recordResource(t, "vault", vault.ID, vault.DisplayName)
	t.Cleanup(func() { safeDelete(t, "vault", vault.ID, vault.DisplayName) })

	if vault.DisplayName != displayName {
		t.Fatalf("vault display_name mismatch: got %q, want %q", vault.DisplayName, displayName)
	}

	got, err := c.Vaults().Get(ctx, vault.ID)
	if err != nil {
		t.Fatalf("failed to get vault: %v", redact(err.Error()))
	}
	if got.ID != vault.ID {
		t.Fatalf("vault id mismatch: got %q, want %q", got.ID, vault.ID)
	}

	list, err := c.Vaults().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list vaults: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("vaults list is empty")
	}

	credReq := vaults.NewStaticBearerCredential(
		"https://example.com/mcp",
		"sse",
		"e2e-credential-"+randomHex(8),
	)
	cred, err := c.Vaults().CreateCredential(ctx, vault.ID, &credReq)
	if err != nil {
		t.Fatalf("failed to create credential: %v", redact(err.Error()))
	}
	if cred.ID == "" {
		t.Fatal("credential id is empty")
	}

	credList, err := c.Vaults().ListCredentials(ctx, vault.ID, nil)
	if err != nil {
		t.Fatalf("failed to list credentials: %v", redact(err.Error()))
	}
	found := false
	for _, existing := range credList.Data {
		if existing.ID == cred.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created credential %s not found in list", cred.ID)
	}

	archivedCred, err := c.Vaults().ArchiveCredential(ctx, vault.ID, cred.ID)
	if err != nil {
		t.Fatalf("failed to archive credential: %v", redact(err.Error()))
	}
	if archivedCred.ID != cred.ID {
		t.Fatalf("archived credential id mismatch: got %q, want %q", archivedCred.ID, cred.ID)
	}

	// Vaults do not support Delete; Archive is performed by t.Cleanup.
	t.Logf("vault lifecycle passed: %s", vault.ID)
}
