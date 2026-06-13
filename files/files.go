// Package files provides the Files resource for uploading and managing files.
// See: https://docs.qoder.com/cloud-agents/api/files/list
package files

import (
	"context"
	"fmt"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// File represents a file stored in the Qoder Cloud.
type File struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Filename  string         `json:"filename"`
	Name      string         `json:"name,omitempty"`
	Purpose   string         `json:"purpose"`
	Status    string         `json:"status"`
	Size      int64          `json:"size"`
	Metadata  types.Metadata `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

// UploadFileRequest holds the data for uploading a file.
type UploadFileRequest struct {
	Filename string
	Data     []byte
	Purpose  string
	Metadata types.Metadata
}

// API provides access to the Files resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Files API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of files, optionally filtered by purpose.
func (a *API) List(ctx context.Context, purpose string, params *types.ListParams) (*types.PaginatedResponse[File], error) {
	req := a.client.GET("/files")
	if purpose != "" {
		req = req.WithQuery("purpose", purpose)
	}
	req = qoderhttp.ApplyListParams(req, params)
	var result types.PaginatedResponse[File]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Upload uploads a file to Qoder Cloud.
// The file is sent as multipart/form-data.
func (a *API) Upload(ctx context.Context, req *UploadFileRequest, idempotencyKey ...string) (*File, error) {
	extraFields := map[string]string{"purpose": req.Purpose}
	if req.Metadata != nil {
		for k, v := range req.Metadata {
			extraFields[fmt.Sprintf("metadata[%s]", k)] = v
		}
	}

	var extraHeaders map[string]string
	if len(idempotencyKey) > 0 && idempotencyKey[0] != "" {
		extraHeaders = map[string]string{"Idempotency-Key": idempotencyKey[0]}
	}

	var file File
	if err := qoderhttp.PostMultipart(ctx, a.client, "/files", "file", req.Filename, req.Data, extraFields, extraHeaders, &file); err != nil {
		return nil, err
	}
	return &file, nil
}

// Get retrieves file metadata by ID.
func (a *API) Get(ctx context.Context, id string) (*File, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var file File
	if err := a.client.GET("/files/" + id).WithContext(ctx).Do(&file); err != nil {
		return nil, err
	}
	return &file, nil
}

// FileContentResponse holds the pre-signed download URL and its expiration time
// returned by GET /files/{id}/content.
type FileContentResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// GetContent returns a pre-signed download URL and expiration time for the file content.
func (a *API) GetContent(ctx context.Context, id string) (*FileContentResponse, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var result FileContentResponse
	if err := a.client.GET("/files/" + id + "/content").WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete permanently deletes a file by ID.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := qoderhttp.ValidateID(id); err != nil {
		return err
	}
	return a.client.DELETE("/files/" + id).WithContext(ctx).Do(nil)
}
