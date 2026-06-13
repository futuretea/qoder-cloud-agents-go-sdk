// Package memorystores provides the Memory Stores resource for persistent memory across sessions.
// See: https://docs.qoder.com/cloud-agents/api/memory-stores/schemas
//
//nolint:revive // Package name is descriptive
package memorystores

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// MemoryStore is a persistent memory store for cross-session memory.
type MemoryStore struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	EntryCount  int            `json:"entry_count"`
	TotalSize   int64          `json:"total_size"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
	ArchivedAt  *string        `json:"archived_at,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// MemoryEntry is a single memory entry within a store.
type MemoryEntry struct {
	ID            string         `json:"id"`
	Path          string         `json:"path"`
	Content       string         `json:"content,omitempty"`
	ContentSHA256 string         `json:"content_sha256"`
	Version       int            `json:"version"`
	Status        string         `json:"status"`
	Metadata      types.Metadata `json:"metadata,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
}

// Version represents an immutable version of a memory entry.
type Version struct {
	ID            string `json:"id"`
	EntryID       string `json:"memory_id"`
	Content       string `json:"content,omitempty"`
	ContentSHA256 string `json:"content_sha256"`
	Version       int    `json:"version"`
	Action        string `json:"action"`
	Redacted      bool   `json:"redacted"`
	CreatedAt     string `json:"created_at"`
}

// CreateStoreRequest is the builder for creating a memory store.
type CreateStoreRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
}

// NewCreateStoreRequest creates a new CreateStoreRequest with the required name.
func NewCreateStoreRequest(name string) *CreateStoreRequest {
	return &CreateStoreRequest{Name: name}
}

// WithDescription sets the store description.
func (r *CreateStoreRequest) WithDescription(desc string) *CreateStoreRequest {
	r.Description = desc
	return r
}

// WithMetadata sets custom metadata.
func (r *CreateStoreRequest) WithMetadata(metadata types.Metadata) *CreateStoreRequest {
	r.Metadata = metadata
	return r
}

// CreateEntryRequest is the builder for creating a memory entry.
type CreateEntryRequest struct {
	Path     string         `json:"path"`
	Content  string         `json:"content"`
	Metadata types.Metadata `json:"metadata,omitempty"`
}

// NewCreateEntryRequest creates a new CreateEntryRequest.
func NewCreateEntryRequest(path, content string) *CreateEntryRequest {
	return &CreateEntryRequest{Path: path, Content: content}
}

// WithMetadata sets custom metadata.
func (r *CreateEntryRequest) WithMetadata(metadata types.Metadata) *CreateEntryRequest {
	r.Metadata = metadata
	return r
}

// UpdateEntryRequest is the builder for updating a memory entry.
// ContentSHA256 is used for optimistic concurrency; set it to the current content's SHA-256.
type UpdateEntryRequest struct {
	Content       string         `json:"content"`
	ContentSHA256 string         `json:"content_sha256,omitempty"`
	Metadata      types.Metadata `json:"metadata,omitempty"`
}

// NewUpdateEntryRequest creates a new UpdateEntryRequest.
func NewUpdateEntryRequest(content string) *UpdateEntryRequest {
	return &UpdateEntryRequest{Content: content}
}

// WithContentSHA256 sets the expected current content SHA-256 for concurrency control.
func (r *UpdateEntryRequest) WithContentSHA256(sha256 string) *UpdateEntryRequest {
	r.ContentSHA256 = sha256
	return r
}

// WithMetadata sets new metadata.
func (r *UpdateEntryRequest) WithMetadata(metadata types.Metadata) *UpdateEntryRequest {
	r.Metadata = metadata
	return r
}

// API provides access to Memory Stores, Entries, and Versions.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Memory Stores API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// --- Store-level methods ---

// List returns a paginated list of memory stores.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[MemoryStore], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/memory_stores"), params)
	var result types.PaginatedResponse[MemoryStore]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new memory store.
func (a *API) Create(ctx context.Context, req *CreateStoreRequest, idempotencyKey ...string) (*MemoryStore, error) {
	r := qoderhttp.ApplyIdempotencyKey(a.client.POST("/memory_stores").WithJSON(req), idempotencyKey...)
	var store MemoryStore
	if err := r.WithContext(ctx).Do(&store); err != nil {
		return nil, err
	}
	return &store, nil
}

// Get retrieves a single memory store by ID.
func (a *API) Get(ctx context.Context, id string) (*MemoryStore, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var store MemoryStore
	if err := a.client.GET("/memory_stores/" + id).WithContext(ctx).Do(&store); err != nil {
		return nil, err
	}
	return &store, nil
}

// Archive archives a memory store.
func (a *API) Archive(ctx context.Context, id string) (*MemoryStore, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var store MemoryStore
	if err := a.client.POST("/memory_stores/" + id + "/archive").WithContext(ctx).Do(&store); err != nil {
		return nil, err
	}
	return &store, nil
}

// Delete permanently deletes a memory store and all entries/versions.
func (a *API) Delete(ctx context.Context, id string) error {
	return a.client.DELETE("/memory_stores/" + id).WithContext(ctx).Do(nil)
}

// --- Entry-level methods ---

// ListEntries returns active entries in a memory store (content is not returned in list).
func (a *API) ListEntries(ctx context.Context, storeID string, params *types.ListParams) (*types.PaginatedResponse[MemoryEntry], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/memory_stores/"+storeID+"/memories"), params)
	var result types.PaginatedResponse[MemoryEntry]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateEntry creates a new memory entry in a store.
func (a *API) CreateEntry(ctx context.Context, storeID string, req *CreateEntryRequest) (*MemoryEntry, error) {
	var entry MemoryEntry
	path := "/memory_stores/" + storeID + "/memories"
	if err := a.client.POST(path).WithJSON(req).WithContext(ctx).Do(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetEntry retrieves a memory entry by ID, including its content.
func (a *API) GetEntry(ctx context.Context, storeID, entryID string) (*MemoryEntry, error) {
	var entry MemoryEntry
	path := "/memory_stores/" + storeID + "/memories/" + entryID
	if err := a.client.GET(path).WithContext(ctx).Do(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// UpdateEntry updates a memory entry, creating a new version.
func (a *API) UpdateEntry(ctx context.Context, storeID, entryID string, req *UpdateEntryRequest) (*MemoryEntry, error) {
	var entry MemoryEntry
	path := "/memory_stores/" + storeID + "/memories/" + entryID
	if err := a.client.PUT(path).WithJSON(req).WithContext(ctx).Do(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// DeleteEntry deletes a memory entry, creating a deleted version record.
func (a *API) DeleteEntry(ctx context.Context, storeID, entryID string) (*MemoryEntry, error) {
	var entry MemoryEntry
	path := "/memory_stores/" + storeID + "/memories/" + entryID
	if err := a.client.DELETE(path).WithContext(ctx).Do(&entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// --- Version-level methods ---

// ListVersions returns version history for a memory store.
func (a *API) ListVersions(ctx context.Context, storeID string, params *types.ListParams) (*types.PaginatedResponse[Version], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/memory_stores/"+storeID+"/versions"), params)
	var result types.PaginatedResponse[Version]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVersion retrieves a specific version by ID. Returns content if not redacted.
func (a *API) GetVersion(ctx context.Context, storeID, versionID string) (*Version, error) {
	var version Version
	path := "/memory_stores/" + storeID + "/versions/" + versionID
	if err := a.client.GET(path).WithContext(ctx).Do(&version); err != nil {
		return nil, err
	}
	return &version, nil
}

// RedactVersion permanently redacts the content of a version.
func (a *API) RedactVersion(ctx context.Context, storeID, versionID string) (*Version, error) {
	var version Version
	path := "/memory_stores/" + storeID + "/versions/" + versionID + "/redact"
	if err := a.client.POST(path).WithContext(ctx).Do(&version); err != nil {
		return nil, err
	}
	return &version, nil
}
