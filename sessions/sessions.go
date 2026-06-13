// Package sessions provides the Sessions resource for creating and managing agent sessions.
// See: https://docs.qoder.com/cloud-agents/api/sessions/create
package sessions

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Session represents an agent run or conversation instance.
type Session struct {
	ID            string         `json:"id"`
	Title         string         `json:"title,omitempty"`
	Agent         string         `json:"agent"`
	EnvironmentID string         `json:"environment_id,omitempty"`
	Status        string         `json:"status"`
	Metadata      types.Metadata `json:"metadata,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

// Resource represents a file or repository attached to a session.
type Resource struct {
	Type      string `json:"type"`
	FileID    string `json:"file_id,omitempty"`
	Path      string `json:"path,omitempty"`
	URL       string `json:"url,omitempty"`
	MountPath string `json:"mount_path,omitempty"`
}

// NewResourceFile creates a file resource attachment.
func NewResourceFile(fileID, path string) Resource {
	return Resource{Type: "file", FileID: fileID, Path: path}
}

// NewResourceGitHub creates a GitHub repository resource attachment.
func NewResourceGitHub(url, mountPath string) Resource {
	return Resource{Type: "github_repository", URL: url, MountPath: mountPath}
}

// CreateSessionRequest is the builder for creating a session.
type CreateSessionRequest struct {
	Agent                string         `json:"agent"`
	EnvironmentID        string         `json:"environment_id,omitempty"`
	Title                string         `json:"title,omitempty"`
	Metadata             types.Metadata `json:"metadata,omitempty"`
	DeltaFlushIntervalMs int            `json:"delta_flush_interval_ms,omitempty"`
	Resources            []Resource     `json:"resources,omitempty"`
	VaultIDs             []string       `json:"vault_ids,omitempty"`
	MemoryStoreIDs       []string       `json:"memory_store_ids,omitempty"`
	EnvironmentVariables string         `json:"environment_variables,omitempty"`
}

// NewCreateRequest creates a new CreateSessionRequest with the required agent ID.
func NewCreateRequest(agentID string) *CreateSessionRequest {
	return &CreateSessionRequest{Agent: agentID}
}

// WithEnvironment sets the environment by ID.
func (r *CreateSessionRequest) WithEnvironment(envID string) *CreateSessionRequest {
	r.EnvironmentID = envID
	return r
}

// WithTitle sets the session title.
func (r *CreateSessionRequest) WithTitle(title string) *CreateSessionRequest {
	r.Title = title
	return r
}

// WithMetadata sets custom metadata.
func (r *CreateSessionRequest) WithMetadata(metadata types.Metadata) *CreateSessionRequest {
	r.Metadata = metadata
	return r
}

// WithDeltaFlushInterval sets the SSE delta flush interval in milliseconds.
func (r *CreateSessionRequest) WithDeltaFlushInterval(ms int) *CreateSessionRequest {
	r.DeltaFlushIntervalMs = ms
	return r
}

// WithResource adds a file or repository resource.
func (r *CreateSessionRequest) WithResource(res Resource) *CreateSessionRequest {
	r.Resources = append(r.Resources, res)
	return r
}

// WithVault binds a vault for MCP credential access.
func (r *CreateSessionRequest) WithVault(vaultID string) *CreateSessionRequest {
	r.VaultIDs = append(r.VaultIDs, vaultID)
	return r
}

// WithMemoryStore binds a memory store for persistent memory.
func (r *CreateSessionRequest) WithMemoryStore(storeID string) *CreateSessionRequest {
	r.MemoryStoreIDs = append(r.MemoryStoreIDs, storeID)
	return r
}

// WithEnvironmentVariables sets environment variables as a newline-separated string.
func (r *CreateSessionRequest) WithEnvironmentVariables(vars string) *CreateSessionRequest {
	r.EnvironmentVariables = vars
	return r
}

// UpdateSessionRequest is the builder for updating a session.
type UpdateSessionRequest struct {
	Title    string         `json:"title,omitempty"`
	Metadata types.Metadata `json:"metadata,omitempty"`
}

// NewUpdateRequest creates a new UpdateSessionRequest.
func NewUpdateRequest() *UpdateSessionRequest {
	return &UpdateSessionRequest{}
}

// WithTitle sets the new title.
func (r *UpdateSessionRequest) WithTitle(title string) *UpdateSessionRequest {
	r.Title = title
	return r
}

// WithMetadata sets new metadata.
func (r *UpdateSessionRequest) WithMetadata(metadata types.Metadata) *UpdateSessionRequest {
	r.Metadata = metadata
	return r
}

// API provides access to the Sessions resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Sessions API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of sessions.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[Session], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/sessions"), params)
	var result types.PaginatedResponse[Session]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new session.
func (a *API) Create(ctx context.Context, req *CreateSessionRequest, idempotencyKey ...string) (*Session, error) {
	r := qoderhttp.ApplyIdempotencyKey(a.client.POST("/sessions").WithJSON(req), idempotencyKey...)
	var session Session
	if err := r.WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Get retrieves a single session by ID.
func (a *API) Get(ctx context.Context, id string) (*Session, error) {
	var session Session
	if err := a.client.GET("/sessions/" + id).WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Update updates an existing session.
func (a *API) Update(ctx context.Context, id string, req *UpdateSessionRequest) (*Session, error) {
	var session Session
	if err := a.client.POST("/sessions/" + id).WithJSON(req).WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Archive archives a session.
func (a *API) Archive(ctx context.Context, id string) (*Session, error) {
	var session Session
	if err := a.client.POST("/sessions/" + id + "/archive").WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// Cancel cancels a running session.
func (a *API) Cancel(ctx context.Context, id string) (*Session, error) {
	var session Session
	if err := a.client.POST("/sessions/" + id + "/cancel").WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

// AddResources attaches file or repository resources to a session.
func (a *API) AddResources(ctx context.Context, id string, resources []Resource) (*Session, error) {
	body := map[string][]Resource{"resources": resources}
	var session Session
	if err := a.client.POST("/sessions/" + id + "/resources").WithJSON(body).WithContext(ctx).Do(&session); err != nil {
		return nil, err
	}
	return &session, nil
}
