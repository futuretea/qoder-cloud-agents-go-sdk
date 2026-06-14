// Package vaults provides the Vaults resource for managing MCP server credentials.
// See: https://docs.qoder.com/cloud-agents/api/vaults/create
package vaults

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Vault represents a secure credential store.
type Vault struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	DisplayName string         `json:"display_name"`
	Status      string         `json:"status"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
	Credentials []Credential   `json:"credentials,omitempty"`
	ArchivedAt  *string        `json:"archived_at,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// Credential represents a single MCP server credential within a vault.
// The access_token is never returned in API responses after creation.
type Credential struct {
	ID           string `json:"id"`
	MCPServerURL string `json:"mcp_server_url"`
	Protocol     string `json:"protocol"`
	Type         string `json:"type"`
	Archived     bool   `json:"archived,omitempty"`
	CreatedAt    string `json:"created_at"`
}

// CreateVaultRequest is the builder for creating a vault.
type CreateVaultRequest struct {
	DisplayName string             `json:"display_name"`
	Credentials []CreateCredential `json:"credentials,omitempty"`
	Metadata    types.Metadata     `json:"metadata,omitempty"`
}

// CreateCredential represents a credential to create, used both as part of a
// vault creation request and as a standalone credential addition to an existing vault.
type CreateCredential struct {
	MCPServerURL string `json:"mcp_server_url"`
	Protocol     string `json:"protocol"`
	Type         string `json:"type"`
	AccessToken  string `json:"access_token"`
}

// CreateCredentialRequest is an alias for CreateCredential, used when adding a
// credential to an existing vault via API.CreateCredential. The two types are
// structurally identical; the distinct name documents the different use context.
type CreateCredentialRequest = CreateCredential

// NewCreateRequest creates a new CreateVaultRequest with the required display name.
func NewCreateRequest(displayName string) *CreateVaultRequest {
	return &CreateVaultRequest{DisplayName: displayName}
}

// WithCredential adds a credential to the vault creation request.
func (r *CreateVaultRequest) WithCredential(cred CreateCredential) *CreateVaultRequest {
	r.Credentials = append(r.Credentials, cred)
	return r
}

// WithMetadata sets custom metadata.
func (r *CreateVaultRequest) WithMetadata(metadata types.Metadata) *CreateVaultRequest {
	r.Metadata = metadata
	return r
}

// NewStaticBearerCredential creates a new static bearer credential for an MCP server.
func NewStaticBearerCredential(mcpServerURL, protocol, accessToken string) CreateCredentialRequest {
	return CreateCredentialRequest{
		MCPServerURL: mcpServerURL,
		Protocol:     protocol,
		Type:         "static_bearer",
		AccessToken:  accessToken,
	}
}

// API provides access to the Vaults resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Vaults API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of vaults.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[Vault], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/vaults"), params)
	var result types.PaginatedResponse[Vault]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new vault, optionally with initial credentials.
func (a *API) Create(ctx context.Context, req *CreateVaultRequest, idempotencyKey ...string) (*Vault, error) {
	r := qoderhttp.ApplyIdempotencyKey(a.client.POST("/vaults").WithJSON(req), idempotencyKey...)
	var vault Vault
	if err := r.WithContext(ctx).Do(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// Get retrieves a single vault by ID.
func (a *API) Get(ctx context.Context, id string) (*Vault, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var vault Vault
	if err := a.client.GET("/vaults/" + id).WithContext(ctx).Do(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// Archive archives a vault.
func (a *API) Archive(ctx context.Context, id string) (*Vault, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var vault Vault
	if err := a.client.POST("/vaults/" + id + "/archive").WithContext(ctx).Do(&vault); err != nil {
		return nil, err
	}
	return &vault, nil
}

// CreateCredential adds a new credential to an existing vault.
func (a *API) CreateCredential(ctx context.Context, vaultID string, req *CreateCredentialRequest) (*Credential, error) {
	if err := qoderhttp.ValidateID(vaultID); err != nil {
		return nil, err
	}
	var cred Credential
	if err := a.client.POST("/vaults/" + vaultID + "/credentials").WithJSON(req).WithContext(ctx).Do(&cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// ListCredentials returns the credentials in a vault (secrets are redacted).
func (a *API) ListCredentials(ctx context.Context, vaultID string, params *types.ListParams) (*types.PaginatedResponse[Credential], error) {
	if err := qoderhttp.ValidateID(vaultID); err != nil {
		return nil, err
	}
	req := qoderhttp.ApplyListParams(a.client.GET("/vaults/"+vaultID+"/credentials"), params)
	var result types.PaginatedResponse[Credential]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ArchiveCredential archives a credential within a vault.
func (a *API) ArchiveCredential(ctx context.Context, vaultID, credentialID string) (*Credential, error) {
	if err := qoderhttp.ValidateID(vaultID); err != nil {
		return nil, err
	}
	if err := qoderhttp.ValidateID(credentialID); err != nil {
		return nil, err
	}
	var cred Credential
	path := "/vaults/" + vaultID + "/credentials/" + credentialID + "/archive"
	if err := a.client.POST(path).WithContext(ctx).Do(&cred); err != nil {
		return nil, err
	}
	return &cred, nil
}
