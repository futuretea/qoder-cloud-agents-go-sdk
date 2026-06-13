//go:build e2e

package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

const (
	e2ePrefix          = "qoder-sdk-e2e-"
	resourcesFile      = ".e2e-resources.jsonl"
	retryMaxAttempts   = 3
	cleanupRetryMax    = 2
	retryBaseInterval  = 500 * time.Millisecond
	retryBackoffFactor = 2
	eventsReadTimeout  = 15 * time.Second
)

var (
	sharedClient     *qoder.Client
	sharedClientOnce sync.Once
	enabledModelID   string
	enabledModelOnce sync.Once

	sensitiveWords     []string
	sensitiveWordsOnce sync.Once
)

// resourceRecord represents a single created resource for cleanup tracking.
type resourceRecord struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// newClientWithTimeout returns a Qoder client with a custom HTTP timeout.
func newClientWithTimeout(t *testing.T, timeout time.Duration) *qoder.Client {
	t.Helper()

	token := os.Getenv("QODER_PAT")
	if token == "" {
		t.Skipf("QODER_PAT is not set; skipping e2e test")
	}

	setSensitiveWords(token)

	hc := &http.Client{Timeout: timeout}
	return qoder.New(token, qoder.WithHTTPClient(hc))
}

// newClient returns a shared Qoder client configured from environment variables.
// It skips the test if QODER_PAT is not set.
func newClient(t *testing.T) *qoder.Client {
	t.Helper()

	token := os.Getenv("QODER_PAT")
	if token == "" {
		t.Skipf("QODER_PAT is not set; skipping e2e test")
	}

	setSensitiveWords(token)

	sharedClientOnce.Do(func() {
		sharedClient = qoder.New(token)
	})

	return sharedClient
}

// requireAck skips the test if the user has not acknowledged e2e writes.
func requireAck(t *testing.T) {
	t.Helper()
	if os.Getenv("QODER_E2E_ACK") != "1" {
		t.Skipf("QODER_E2E_ACK=1 is required to run e2e write tests")
	}
}

// requireProdOk skips the test if the base URL points to production and the
// user has not explicitly confirmed production writes.
func requireProdOk(t *testing.T) {
	t.Helper()

	baseURL := os.Getenv("QODER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.qoder.com/api/v1/cloud"
	}

	if strings.Contains(baseURL, "api.qoder.com") && os.Getenv("QODER_E2E_PROD_OK") != "1" {
		t.Skipf("base URL points to production (api.qoder.com); set QODER_E2E_PROD_OK=1 after confirming an isolated test account")
	}
}

// setSensitiveWords records strings that must be redacted from diagnostic output.
// It is safe for concurrent use and initializes the slice exactly once per process.
func setSensitiveWords(token string) {
	sensitiveWordsOnce.Do(func() {
		sensitiveWords = append(sensitiveWords, token)
		if len(token) > 8 {
			sensitiveWords = append(sensitiveWords, token[:len(token)/2])
		}
	})
}

// redact removes sensitive substrings from a string.
func redact(s string) string {
	for _, w := range sensitiveWords {
		if w == "" {
			continue
		}
		s = strings.ReplaceAll(s, w, "<redacted>")
	}
	s = regexp.MustCompile(`(?i)authorization:\s*Bearer\s+\S+`).ReplaceAllString(s, "Authorization: Bearer <redacted>")
	return s
}

// logError prints a redacted error message.
func logError(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Log(redact(fmt.Sprintf(format, args...)))
}

// closeOrLog closes c and logs any error. It is intended for deferred cleanup
// of response bodies and streams where a close failure should not hide a test
// failure but must still be observed.
func closeOrLog(t *testing.T, c io.Closer) {
	t.Helper()
	if err := c.Close(); err != nil {
		logError(t, "close failed: %v", err)
	}
}

// randomHex returns a random hex string of the requested byte length.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to read random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}

// sanitize makes a test name safe to embed in a resource name.
func sanitize(name string) string {
	s := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(name, "-")
	if len(s) > 32 {
		s = s[:32]
	}
	return strings.Trim(s, "-")
}

// newE2EResourceName returns a unique resource name with the required prefix.
func newE2EResourceName(t *testing.T, kind string) string {
	t.Helper()
	name := fmt.Sprintf("%s%s-%s-%s", e2ePrefix, kind, sanitize(t.Name()), randomHex(4))
	if !strings.HasPrefix(name, e2ePrefix) {
		t.Fatalf("invalid e2e resource name: %s", name)
	}
	return name
}

// cleanupRetry executes fn with backoff for cleanup operations.
func cleanupRetry(ctx context.Context, fn func() error) error {
	return doRetry(ctx, fn, cleanupRetryMax)
}

// createWithRetry executes a create-style fn with retry for transient errors.
func createWithRetry(ctx context.Context, fn func() error) error {
	return doRetry(ctx, fn, retryMaxAttempts)
}

// createWithRetryValue executes a create-style fn that returns a value, with retry.
func createWithRetryValue[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	var result T
	err := createWithRetry(ctx, func() error {
		var err error
		result, err = fn()
		return err
	})
	return result, err
}

func doRetry(ctx context.Context, fn func() error, maxAttempts int) error {
	var lastErr error
	delay := retryBaseInterval
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
				delay *= retryBackoffFactor
			}
		}
	}
	return lastErr
}

// isRetriable reports whether an error is likely transient.
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
	// Retry temporary network errors.
	var temporary interface{ Temporary() bool }
	if errors.As(err, &temporary) {
		return temporary.Temporary()
	}
	return false
}

// isNotFoundOrConflict reports whether err represents a resource that is already
// gone or in a terminal state, which is acceptable for cleanup.
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

// ignoreNotFoundOrConflict returns nil for errors that indicate a resource has
// already been cleaned up, so cleanup can be treated as successful.
func ignoreNotFoundOrConflict(err error) error {
	if isNotFoundOrConflict(err) {
		return nil
	}
	return err
}

// recordResource appends a created resource to the inventory file.
func recordResource(t *testing.T, resourceType, id, name string) {
	t.Helper()
	if id == "" {
		t.Fatalf("cannot record resource with empty id (type=%s)", resourceType)
	}

	rec := resourceRecord{Type: resourceType, ID: id, Name: name}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("failed to marshal resource record: %v", err)
	}

	f, err := os.OpenFile(resourcesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("failed to open resource inventory %s: %v", resourcesFile, err)
	}
	defer closeOrLog(t, f)

	if _, err := fmt.Fprintln(f, string(data)); err != nil {
		t.Fatalf("failed to write resource record: %v", err)
	}
	if err := f.Sync(); err != nil {
		t.Fatalf("failed to sync resource inventory: %v", err)
	}
}

// unrecordResource removes the line with the given resource id from the
// inventory file. If the file becomes empty it is removed. Errors are logged
// rather than failing the test so that cleanup failures remain the primary
// signal.
func unrecordResource(t *testing.T, id string) {
	t.Helper()
	if id == "" {
		return
	}

	data, err := os.ReadFile(resourcesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logError(t, "failed to read inventory to unrecord %s: %v", id, err)
		return
	}

	lines := bytes.Split(data, []byte("\n"))
	kept := make([][]byte, 0, len(lines))
	removed := false
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var rec resourceRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			kept = append(kept, line)
			continue
		}
		if rec.ID == id {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return
	}
	if len(kept) == 0 {
		if err := os.Remove(resourcesFile); err != nil {
			logError(t, "failed to remove empty inventory: %v", err)
		}
		return
	}

	out := append(bytes.Join(kept, []byte("\n")), '\n')
	if err := os.WriteFile(resourcesFile, out, 0o600); err != nil {
		logError(t, "failed to rewrite inventory after unrecording %s: %v", id, err)
	}
}

// safeDelete dispatches cleanup based on resource type.
// The name is used as a guard: cleanup is refused if the name does not start
// with the e2e prefix, except for memoryentry where name holds the parent store
// name and id holds the entry id.
func safeDelete(t *testing.T, resourceType, id, name string) {
	t.Helper()
	if id == "" {
		t.Fatalf("refusing to delete resource with empty id: type=%s", resourceType)
	}
	if resourceType != "memoryentry" && !strings.HasPrefix(name, e2ePrefix) {
		t.Fatalf("refusing to delete non-e2e resource: type=%s id=%s name=%s", resourceType, id, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	token := os.Getenv("QODER_PAT")
	if token == "" {
		logError(t, "QODER_PAT not set during cleanup of %s %s", resourceType, id)
		return
	}
	setSensitiveWords(token)

	opts := []qoder.Option{}
	if baseURL := os.Getenv("QODER_BASE_URL"); baseURL != "" {
		opts = append(opts, qoder.WithBaseURL(baseURL))
	}
	c := qoder.New(token, opts...)
	var err error

	switch resourceType {
	case "agent":
		err = cleanupRetry(ctx, func() error { _, err := c.Agents().Archive(ctx, id); return ignoreNotFoundOrConflict(err) })
	case "environment":
		err = cleanupRetry(ctx, func() error { _, err := c.Environments().Archive(ctx, id); return ignoreNotFoundOrConflict(err) })
	case "session":
		_ = cleanupRetry(ctx, func() error { _, err := c.Sessions().Cancel(ctx, id); return ignoreNotFoundOrConflict(err) })
		err = cleanupRetry(ctx, func() error { _, err := c.Sessions().Archive(ctx, id); return ignoreNotFoundOrConflict(err) })
	case "file":
		err = cleanupRetry(ctx, func() error { return ignoreNotFoundOrConflict(c.Files().Delete(ctx, id)) })
	case "skill":
		err = cleanupRetry(ctx, func() error { return ignoreNotFoundOrConflict(c.Skills().Delete(ctx, id)) })
	case "memorystore":
		err = cleanupRetry(ctx, func() error { return ignoreNotFoundOrConflict(c.MemoryStores().Delete(ctx, id)) })
	case "memoryentry":
		err = cleanupRetry(ctx, func() error {
			_, err := c.MemoryStores().DeleteEntry(ctx, name, id)
			return ignoreNotFoundOrConflict(err)
		})
	case "vault":
		err = cleanupRetry(ctx, func() error { _, err := c.Vaults().Archive(ctx, id); return ignoreNotFoundOrConflict(err) })
	default:
		t.Fatalf("unknown resource type: %s", resourceType)
	}

	if err != nil {
		logError(t, "cleanup failed for %s %s (%s): %v", resourceType, id, name, err)
		return
	}
	unrecordResource(t, id)
}

// pickEnabledModel returns the ID of an enabled model, caching the result.
func pickEnabledModel(t *testing.T) string {
	t.Helper()

	enabledModelOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		c := newClient(t)
		models, err := c.Models().List(ctx)
		if err != nil {
			t.Fatalf("failed to list models: %v", redact(err.Error()))
		}
		for _, m := range models {
			if m.IsEnabled {
				enabledModelID = m.ID
				return
			}
		}
		t.Skip("no enabled model found")
	})

	if enabledModelID == "" {
		t.Skip("no enabled model available")
	}
	return enabledModelID
}

// newMinimalSkillZip returns the bytes of a minimal custom skill zip.
func newMinimalSkillZip(t *testing.T) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for _, entry := range []struct {
		name string
		data string
	}{
		{"SKILL.md", "---\nname: minimal\ndescription: A minimal skill for e2e tests\nversion: 1.0.0\n---\n\n# Minimal\n\nA minimal skill for e2e tests.\n"},
		{"skill.json", `{"name":"minimal","version":"1.0.0"}`},
	} {
		f, err := zw.Create(entry.name)
		if err != nil {
			t.Fatalf("failed to create %s in zip: %v", entry.name, err)
		}
		if _, err := f.Write([]byte(entry.data)); err != nil {
			t.Fatalf("failed to write %s: %v", entry.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	return buf.Bytes()
}

// TestE2EZInventoryEmpty verifies that successful cleanups leave the inventory
// file empty or non-existent. It is named with a leading Z so that it runs
// after the other e2e tests when tests are executed in sorted order.
func TestE2EZInventoryEmpty(t *testing.T) {
	info, err := os.Stat(resourcesFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Log("inventory file does not exist; no resources to verify")
			return
		}
		t.Fatalf("failed to stat inventory file: %v", err)
	}
	if info.Size() == 0 {
		t.Log("inventory file is empty")
		return
	}

	data, err := os.ReadFile(resourcesFile)
	if err != nil {
		t.Fatalf("failed to read inventory file: %v", err)
	}
	t.Fatalf("inventory file %s is not empty after cleanup:\n%s", resourcesFile, string(data))
}
