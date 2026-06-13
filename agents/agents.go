// Package agents provides the Agents resource for managing reusable agent configurations.
// See: https://docs.qoder.com/cloud-agents/api/agents/create
package agents

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Agent represents a reusable agent configuration template.
type Agent struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Model       string         `json:"model"`
	System      string         `json:"system,omitempty"`
	Description string         `json:"description,omitempty"`
	Tools       []Tool         `json:"tools,omitempty"`
	MCPServers  []MCPServer    `json:"mcp_servers,omitempty"`
	Skills      []SkillRef     `json:"skills,omitempty"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
	Version     int            `json:"version"`
	Archived    bool           `json:"archived"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// AgentVersion represents a specific version of an agent configuration.
type AgentVersion struct {
	Version   int    `json:"version"`
	Model     string `json:"model"`
	System    string `json:"system,omitempty"`
	CreatedAt string `json:"created_at"`
}

// Tool configures built-in tools for an agent.
type Tool struct {
	Type         string   `json:"type"`
	EnabledTools []string `json:"enabled_tools,omitempty"`
}

// MCPServer configures an MCP server for an agent.
type MCPServer struct {
	Type string `json:"type"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// SkillRef references a skill to bind to an agent.
type SkillRef struct {
	Type    string `json:"type"`
	SkillID string `json:"skill_id"`
	Version int    `json:"version,omitempty"`
}

// CreateAgentRequest is the builder for creating an agent.
type CreateAgentRequest struct {
	Name        string         `json:"name"`
	Model       string         `json:"model"`
	System      string         `json:"system,omitempty"`
	Description string         `json:"description,omitempty"`
	Tools       []Tool         `json:"tools,omitempty"`
	MCPServers  []MCPServer    `json:"mcp_servers,omitempty"`
	Skills      []SkillRef     `json:"skills,omitempty"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
}

// NewCreateRequest creates a new CreateAgentRequest with the required name and model.
func NewCreateRequest(name, model string) *CreateAgentRequest {
	return &CreateAgentRequest{Name: name, Model: model}
}

// WithSystem sets the system prompt.
func (r *CreateAgentRequest) WithSystem(system string) *CreateAgentRequest {
	r.System = system
	return r
}

// WithDescription sets the agent description.
func (r *CreateAgentRequest) WithDescription(desc string) *CreateAgentRequest {
	r.Description = desc
	return r
}

// WithTool adds a built-in tool configuration.
func (r *CreateAgentRequest) WithTool(tool Tool) *CreateAgentRequest {
	r.Tools = append(r.Tools, tool)
	return r
}

// WithMCPServer adds an MCP server configuration.
func (r *CreateAgentRequest) WithMCPServer(server MCPServer) *CreateAgentRequest {
	r.MCPServers = append(r.MCPServers, server)
	return r
}

// WithSkill adds a skill binding.
func (r *CreateAgentRequest) WithSkill(skill SkillRef) *CreateAgentRequest {
	r.Skills = append(r.Skills, skill)
	return r
}

// WithMetadata sets custom metadata.
func (r *CreateAgentRequest) WithMetadata(metadata types.Metadata) *CreateAgentRequest {
	r.Metadata = metadata
	return r
}

// UpdateAgentRequest is the builder for updating an agent.
// Version is required for optimistic concurrency control.
type UpdateAgentRequest struct {
	Version     int            `json:"version"`
	Name        string         `json:"name,omitempty"`
	Model       string         `json:"model,omitempty"`
	System      string         `json:"system,omitempty"`
	Description string         `json:"description,omitempty"`
	Tools       []Tool         `json:"tools,omitempty"`
	MCPServers  []MCPServer    `json:"mcp_servers,omitempty"`
	Skills      []SkillRef     `json:"skills,omitempty"`
	Metadata    types.Metadata `json:"metadata,omitempty"`
}

// NewUpdateRequest creates a new UpdateAgentRequest with the current version.
func NewUpdateRequest(version int) *UpdateAgentRequest {
	return &UpdateAgentRequest{Version: version}
}

// WithName sets the new name.
func (r *UpdateAgentRequest) WithName(name string) *UpdateAgentRequest {
	r.Name = name
	return r
}

// WithModel sets the new model.
func (r *UpdateAgentRequest) WithModel(model string) *UpdateAgentRequest {
	r.Model = model
	return r
}

// WithSystem sets the new system prompt.
func (r *UpdateAgentRequest) WithSystem(system string) *UpdateAgentRequest {
	r.System = system
	return r
}

// WithDescription sets the new description.
func (r *UpdateAgentRequest) WithDescription(desc string) *UpdateAgentRequest {
	r.Description = desc
	return r
}

// WithMetadata sets the new metadata.
func (r *UpdateAgentRequest) WithMetadata(metadata types.Metadata) *UpdateAgentRequest {
	r.Metadata = metadata
	return r
}

// WithTool adds a built-in tool configuration.
func (r *UpdateAgentRequest) WithTool(tool Tool) *UpdateAgentRequest {
	r.Tools = append(r.Tools, tool)
	return r
}

// WithMCPServer adds an MCP server configuration.
func (r *UpdateAgentRequest) WithMCPServer(server MCPServer) *UpdateAgentRequest {
	r.MCPServers = append(r.MCPServers, server)
	return r
}

// WithSkill adds a skill binding.
func (r *UpdateAgentRequest) WithSkill(skill SkillRef) *UpdateAgentRequest {
	r.Skills = append(r.Skills, skill)
	return r
}

// API provides access to the Agents resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Agents API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of agents.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[Agent], error) {
	req := qoderhttp.ApplyListParams(a.client.GET("/agents"), params)
	var result types.PaginatedResponse[Agent]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create creates a new agent.
func (a *API) Create(ctx context.Context, req *CreateAgentRequest, idempotencyKey ...string) (*Agent, error) {
	r := qoderhttp.ApplyIdempotencyKey(a.client.POST("/agents").WithJSON(req), idempotencyKey...)
	var agent Agent
	if err := r.WithContext(ctx).Do(&agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Get retrieves a single agent by ID.
func (a *API) Get(ctx context.Context, id string) (*Agent, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.client.GET("/agents/" + id).WithContext(ctx).Do(&agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Update updates an existing agent. Requires the current version for optimistic concurrency.
func (a *API) Update(ctx context.Context, id string, req *UpdateAgentRequest) (*Agent, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.client.PUT("/agents/" + id).WithJSON(req).WithContext(ctx).Do(&agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Archive archives an agent.
func (a *API) Archive(ctx context.Context, id string) (*Agent, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var agent Agent
	if err := a.client.POST("/agents/" + id + "/archive").WithContext(ctx).Do(&agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// ListVersions returns the version history of an agent.
func (a *API) ListVersions(ctx context.Context, id string, params *types.ListParams) (*types.PaginatedResponse[AgentVersion], error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	req := qoderhttp.ApplyListParams(a.client.GET("/agents/"+id+"/versions"), params)
	var result types.PaginatedResponse[AgentVersion]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
