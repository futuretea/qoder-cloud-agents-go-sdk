//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/memorystores"
)

func TestE2EMemoryStores(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	storeName := newE2EResourceName(t, "memorystore")

	store, err := createWithRetryValue(ctx, func() (*memorystores.MemoryStore, error) {
		return c.MemoryStores().Create(ctx, memorystores.NewCreateStoreRequest(storeName).WithDescription("e2e memory store"))
	})
	if err != nil {
		t.Fatalf("failed to create memory store: %v", redact(err.Error()))
	}
	recordResource(t, "memorystore", store.ID, store.Name)
	t.Cleanup(func() { safeDelete(t, "memorystore", store.ID, store.Name) })

	if store.Name != storeName {
		t.Fatalf("memory store name mismatch: got %q, want %q", store.Name, storeName)
	}

	gotStore, err := c.MemoryStores().Get(ctx, store.ID)
	if err != nil {
		t.Fatalf("failed to get memory store: %v", redact(err.Error()))
	}
	if gotStore.ID != store.ID {
		t.Fatalf("memory store id mismatch: got %q, want %q", gotStore.ID, store.ID)
	}

	storeList, err := c.MemoryStores().List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list memory stores: %v", redact(err.Error()))
	}
	if len(storeList.Data) == 0 {
		t.Fatal("memory stores list is empty")
	}

	entry, err := c.MemoryStores().CreateEntry(ctx, store.ID, memorystores.NewCreateEntryRequest("e2e/hello.md", "hello memory"))
	if err != nil {
		t.Fatalf("failed to create memory entry: %v", redact(err.Error()))
	}
	if entry.Path != "e2e/hello.md" {
		t.Fatalf("memory entry path mismatch: got %q, want %q", entry.Path, "e2e/hello.md")
	}
	recordResource(t, "memoryentry", entry.ID, store.ID)
	t.Cleanup(func() { safeDelete(t, "memoryentry", entry.ID, store.ID) })

	gotEntry, err := c.MemoryStores().GetEntry(ctx, store.ID, entry.ID)
	if err != nil {
		t.Fatalf("failed to get memory entry: %v", redact(err.Error()))
	}
	if gotEntry.ID != entry.ID {
		t.Fatalf("memory entry id mismatch: got %q, want %q", gotEntry.ID, entry.ID)
	}
	if gotEntry.Content != "hello memory" {
		t.Fatalf("memory entry content mismatch: got %q, want %q", gotEntry.Content, "hello memory")
	}

	updatedEntry, err := c.MemoryStores().UpdateEntry(ctx, store.ID, entry.ID, memorystores.NewUpdateEntryRequest("hello memory updated"))
	if err != nil {
		t.Fatalf("failed to update memory entry: %v", redact(err.Error()))
	}
	if updatedEntry.Content != "hello memory updated" {
		t.Fatalf("memory entry content after update mismatch: got %q, want %q", updatedEntry.Content, "hello memory updated")
	}
	if updatedEntry.Version <= entry.Version {
		t.Fatalf("memory entry version did not increase: got %d, want > %d", updatedEntry.Version, entry.Version)
	}

	entries, err := c.MemoryStores().ListEntries(ctx, store.ID, nil)
	if err != nil {
		t.Fatalf("failed to list memory entries: %v", redact(err.Error()))
	}
	if len(entries.Data) == 0 {
		t.Fatal("memory entries list is empty")
	}

	versions, err := c.MemoryStores().ListVersions(ctx, store.ID, nil)
	if err != nil {
		t.Fatalf("failed to list memory versions: %v", redact(err.Error()))
	}
	if len(versions.Data) == 0 {
		t.Fatal("memory versions list is empty")
	}

	t.Logf("memory store lifecycle passed: %s", store.ID)
}
