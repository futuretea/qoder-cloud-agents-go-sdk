//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/files"
)

func TestE2EFiles(t *testing.T) {
	requireAck(t)
	requireProdOk(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := newClient(t)
	filename := newE2EResourceName(t, "file") + ".txt"
	content := []byte("hello qoder e2e")

	file, err := createWithRetryValue(ctx, func() (*files.File, error) {
		return c.Files().Upload(ctx, &files.UploadFileRequest{
			Filename: filename,
			Data:     content,
			Purpose:  "session_resource",
		})
	})
	if err != nil {
		t.Fatalf("failed to upload file: %v", redact(err.Error()))
	}
	if file.ID == "" {
		t.Fatalf("file upload returned empty id: %+v", file)
	}
	recordResource(t, "file", file.ID, file.Filename)
	t.Cleanup(func() { safeDelete(t, "file", file.ID, file.Filename) })

	if file.Filename != filename {
		t.Fatalf("file filename mismatch: got %q, want %q", file.Filename, filename)
	}
	if file.Size != int64(len(content)) {
		t.Fatalf("file size mismatch: got %d, want %d", file.Size, len(content))
	}

	got, err := c.Files().Get(ctx, file.ID)
	if err != nil {
		t.Fatalf("failed to get file: %v", redact(err.Error()))
	}
	if got.ID != file.ID {
		t.Fatalf("file id mismatch: got %q, want %q", got.ID, file.ID)
	}

	list, err := c.Files().List(ctx, "", nil)
	if err != nil {
		t.Fatalf("failed to list files: %v", redact(err.Error()))
	}
	if len(list.Data) == 0 {
		t.Fatal("files list is empty")
	}

	contentResp, err := c.Files().GetContent(ctx, file.ID)
	if err != nil {
		t.Fatalf("failed to get file content: %v", redact(err.Error()))
	}
	if contentResp.URL == "" {
		t.Fatal("file content url is empty")
	}
	if contentResp.ExpiresAt == "" {
		t.Fatal("file content expires_at is empty")
	}

	t.Logf("file lifecycle passed: %s", file.ID)
}
