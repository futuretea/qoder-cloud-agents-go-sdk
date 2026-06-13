// cleanup-e2e.go is the implementation backing scripts/cleanup-e2e.sh.
//
// It reads e2e/.e2e-resources.jsonl and deletes/archives/cancels each recorded
// resource using the SDK. Successfully cleaned resources are removed from the
// inventory; failures are kept so they can be retried manually.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

const resourcesFile = "e2e/.e2e-resources.jsonl"

// resourceRecord mirrors the inventory schema used by the e2e test suite.
type resourceRecord struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

func main() {
	dryRun := len(os.Args) > 1 && os.Args[1] == "--dry-run"

	if os.Getenv("QODER_PAT") == "" {
		fmt.Fprintln(os.Stderr, "Error: QODER_PAT is not set")
		os.Exit(1)
	}
	if os.Getenv("QODER_E2E_ACK") != "1" {
		fmt.Fprintln(os.Stderr, "Error: QODER_E2E_ACK must be set to 1")
		os.Exit(1)
	}

	baseURL := os.Getenv("QODER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.qoder.com/api/v1/cloud"
	}

	if _, err := os.Stat(resourcesFile); os.IsNotExist(err) {
		fmt.Printf("No resource inventory found at %s\n", resourcesFile)
		return
	}

	records, err := readInventory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read inventory: %v\n", err)
		os.Exit(1)
	}

	if dryRun {
		for i := len(records) - 1; i >= 0; i-- {
			rec := records[i]
			fmt.Printf("would clean %s %s\n", rec.Type, rec.ID)
		}
		return
	}

	client := qoder.New(os.Getenv("QODER_PAT"), qoder.WithBaseURL(baseURL))
	var remaining []resourceRecord
	failed := false

	// Process in reverse order so child resources are cleaned before parents.
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		if err := cleanupResource(context.Background(), client, rec); err != nil {
			fmt.Fprintf(os.Stderr, "failed to clean %s %s: %v\n", rec.Type, rec.ID, err)
			remaining = append(remaining, rec)
			failed = true
			continue
		}
		fmt.Printf("cleaned %s %s\n", rec.Type, rec.ID)
	}

	if err := writeInventory(remaining); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to update inventory: %v\n", err)
		os.Exit(1)
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("Cleanup complete.")
}

func readInventory() ([]resourceRecord, error) {
	f, err := os.Open(resourcesFile)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // best effort

	var records []resourceRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var rec resourceRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return nil, fmt.Errorf("invalid inventory line %q: %w", line, err)
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func writeInventory(records []resourceRecord) error {
	if len(records) == 0 {
		if err := os.Remove(resourcesFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove empty inventory: %w", err)
		}
		return nil
	}

	f, err := os.Create(resourcesFile)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // best effort

	enc := json.NewEncoder(f)
	for _, rec := range records {
		if err := enc.Encode(rec); err != nil {
			return err
		}
	}
	return nil
}

func cleanupResource(ctx context.Context, client *qoder.Client, rec resourceRecord) error {
	return cleanupRetry(ctx, func() error {
		var err error
		switch rec.Type {
		case "agent":
			_, err = client.Agents().Archive(ctx, rec.ID)
		case "environment":
			_, err = client.Environments().Archive(ctx, rec.ID)
		case "session":
			_ = ignoreNotFound(cancelSession(ctx, client, rec.ID))
			_, err = client.Sessions().Archive(ctx, rec.ID)
		case "file":
			err = client.Files().Delete(ctx, rec.ID)
		case "skill":
			err = client.Skills().Delete(ctx, rec.ID)
		case "memorystore":
			err = client.MemoryStores().Delete(ctx, rec.ID)
		case "memoryentry":
			_, err = client.MemoryStores().DeleteEntry(ctx, rec.Name, rec.ID)
		case "vault":
			_, err = client.Vaults().Archive(ctx, rec.ID)
		default:
			return fmt.Errorf("unknown resource type: %s", rec.Type)
		}
		if isNotFoundOrConflict(err) {
			return nil
		}
		return err
	})
}

func cancelSession(ctx context.Context, client *qoder.Client, id string) error {
	_, err := client.Sessions().Cancel(ctx, id)
	return err
}

func cleanupRetry(ctx context.Context, fn func() error) error {
	const (
		maxAttempts = 3
		baseDelay   = 500 * time.Millisecond
		backoff     = 2
	)
	var lastErr error
	delay := baseDelay
	for i := 0; i < maxAttempts; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !isRetriable(lastErr) {
			return lastErr
		}
		if i < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= time.Duration(backoff)
			}
		}
	}
	return lastErr
}

func isRetriable(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := qoderhttp.IsAPIError(err); ok {
		switch apiErr.StatusCode {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
	}
	var temporary interface{ Temporary() bool }
	if errors.As(err, &temporary) {
		return temporary.Temporary()
	}
	return false
}

func isNotFoundOrConflict(err error) bool {
	if err == nil {
		return false
	}
	apiErr, ok := qoderhttp.IsAPIError(err)
	if !ok {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusConflict
}

func ignoreNotFound(err error) error {
	if isNotFoundOrConflict(err) {
		return nil
	}
	return err
}
