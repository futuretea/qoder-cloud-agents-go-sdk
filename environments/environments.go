// Package environments provides the Environments resource for managing cloud execution environments.
// See: https://docs.qoder.com/cloud-agents/api/environments/create
package environments

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Environment represents a cloud execution environment.
type Environment struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Config      EnvConfig      `json:"config"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
	ArchivedAt  *string        `json:"archived_at,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// EnvConfig holds the cloud environment configuration.
type EnvConfig struct {
	Type       string     `json:"type"`
	Networking Networking `json:"networking,omitempty"`
	Packages   Packages   `json:"packages,omitempty"`
}

// Networking defines the network access policy.
type Networking struct {
	Type string `json:"type"`
}

// Packages defines the system packages to install.
type Packages struct {
	Apt []string `json:"apt,omitempty"`
	Pip []string `json:"pip,omitempty"`
	Npm []string `json:"npm,omitempty"`
}

// CreateEnvRequest is the builder for creating an environment.
type CreateEnvRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Config      EnvConfig      `json:"config"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
}

// NewCreateRequest creates a new CreateEnvRequest with the required name and config.
func NewCreateRequest(name string, config EnvConfig) *CreateEnvRequest {
	return &CreateEnvRequest{Name: name, Config: config}
}

// WithDescription sets the environment description.
func (r *CreateEnvRequest) WithDescription(desc string) *CreateEnvRequest {
	r.Description = desc
	return r
}

// WithConfig overrides the environment configuration.
func (r *CreateEnvRequest) WithConfig(config EnvConfig) *CreateEnvRequest {
	r.Config = config
	return r
}

// WithMetadata sets custom metadata.
func (r *CreateEnvRequest) WithMetadata(metadata types.Metadata) *CreateEnvRequest {
	r.Metadata = metadata
	return r
}

// UpdateEnvRequest is the builder for updating an environment.
type UpdateEnvRequest struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Config      *EnvConfig     `json:"config,omitempty"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
}

// NewUpdateRequest creates a new UpdateEnvRequest.
func NewUpdateRequest() *UpdateEnvRequest {
	return &UpdateEnvRequest{}
}

// WithName sets the new name.
func (r *UpdateEnvRequest) WithName(name string) *UpdateEnvRequest {
	r.Name = name
	return r
}

// WithDescription sets the new description.
func (r *UpdateEnvRequest) WithDescription(desc string) *UpdateEnvRequest {
	r.Description = desc
	return r
}

// WithConfig sets the new configuration.
func (r *UpdateEnvRequest) WithConfig(config EnvConfig) *UpdateEnvRequest {
	r.Config = &config
	return r
}

// WithMetadata sets new metadata.
func (r *UpdateEnvRequest) WithMetadata(metadata types.Metadata) *UpdateEnvRequest {
	r.Metadata = metadata
	return r
}

// API provides access to the Environments resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Environments API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of environments.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[Environment], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/environments"), params)
	var result types.PaginatedResponse[Environment]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new environment.
func (a *API) Create(ctx context.Context, req *CreateEnvRequest, idempotencyKey ...string) (*Environment, error) {
	r := qoderhttp.ApplyIdempotencyKey(a.client.POST("/environments").WithJSON(req), idempotencyKey...)
	var env Environment
	if err := r.WithContext(ctx).Do(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Get retrieves a single environment by ID.
func (a *API) Get(ctx context.Context, id string) (*Environment, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var env Environment
	if err := a.client.GET("/environments/" + id).WithContext(ctx).Do(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Update updates an existing environment.
func (a *API) Update(ctx context.Context, id string, req *UpdateEnvRequest) (*Environment, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var env Environment
	if err := a.client.PUT("/environments/" + id).WithJSON(req).WithContext(ctx).Do(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Archive archives an environment.
func (a *API) Archive(ctx context.Context, id string) (*Environment, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var env Environment
	if err := a.client.POST("/environments/" + id + "/archive").WithContext(ctx).Do(&env); err != nil {
		return nil, err
	}
	return &env, nil
}

// Delete deletes an environment.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := qoderhttp.ValidateID(id); err != nil {
		return err
	}
	return a.client.DELETE("/environments/" + id).WithContext(ctx).Do(nil)
}
