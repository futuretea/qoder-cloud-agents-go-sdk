// Package skills provides the Skills resource for uploading and managing custom skills.
// See: https://docs.qoder.com/cloud-agents/api/skills/create
package skills

import (
	"context"
	"fmt"

	httpclient "github.com/futuretea/go-http-client"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
)

// Skill represents a custom or prebuilt skill.
type Skill struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	SkillType       string `json:"skill_type"`
	ContentSize     int64  `json:"content_size"`
	ContentSHA256   string `json:"content_sha256,omitempty"`
	Version         int    `json:"version"`
	Status          string `json:"status"`
	Content         string `json:"content,omitempty"`
	ContentEncoding string `json:"content_encoding,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// CreateSkillRequest holds the data for creating a skill from a .zip file.
type CreateSkillRequest struct {
	Filename string
	Data     []byte
	Type     string // "custom" or "prebuilt"
}

// UpdateSkillRequest is the builder for updating a skill's metadata.
type UpdateSkillRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewUpdateRequest creates a new UpdateSkillRequest.
func NewUpdateRequest() *UpdateSkillRequest {
	return &UpdateSkillRequest{}
}

// WithName sets the new name.
func (r *UpdateSkillRequest) WithName(name string) *UpdateSkillRequest {
	r.Name = name
	return r
}

// WithDescription sets the new description.
func (r *UpdateSkillRequest) WithDescription(desc string) *UpdateSkillRequest {
	r.Description = desc
	return r
}

// API provides access to the Skills resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Skills API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns a paginated list of skills.
func (a *API) List(ctx context.Context, params *types.ListParams) (*types.PaginatedResponse[Skill], error) {
	req, err := qoderhttp.ApplyListParams(a.client.GET("/skills"), params)
	if err != nil {
		return nil, err
	}
	var result types.PaginatedResponse[Skill]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Create uploads a .zip skill package and creates a new skill.
func (a *API) Create(ctx context.Context, req *CreateSkillRequest, idempotencyKey ...string) (*Skill, error) {
	if req == nil {
		return nil, fmt.Errorf("skills: CreateSkillRequest must not be nil")
	}
	if req.Filename == "" {
		return nil, fmt.Errorf("skills: CreateSkillRequest.Filename is required")
	}
	if req.Data == nil {
		return nil, fmt.Errorf("skills: CreateSkillRequest.Data is required")
	}
	extraFields := map[string]string{"type": req.Type}

	var extraHeaders map[string]string
	if len(idempotencyKey) > 0 && idempotencyKey[0] != "" {
		extraHeaders = map[string]string{"Idempotency-Key": idempotencyKey[0]}
	}

	var skill Skill
	if err := qoderhttp.PostMultipart(ctx, a.client, "/skills", "file", req.Filename, req.Data, extraFields, extraHeaders, &skill); err != nil {
		return nil, err
	}
	return &skill, nil
}

// Get retrieves a skill by ID. Set includeContent to true to receive base64-encoded content.
func (a *API) Get(ctx context.Context, id string, includeContent bool) (*Skill, error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	req := a.client.GET("/skills/" + id)
	if includeContent {
		req = req.WithQuery("include_content", "true")
	}
	var skill Skill
	if err := req.WithContext(ctx).Do(&skill); err != nil {
		return nil, err
	}
	return &skill, nil
}

// Update updates a skill's metadata (name, description).
func (a *API) Update(ctx context.Context, id string, req *UpdateSkillRequest) (*Skill, error) {
	if req == nil {
		return nil, fmt.Errorf("skills: UpdateSkillRequest must not be nil")
	}
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	var skill Skill
	if err := a.client.PUT("/skills/" + id).WithJSON(req).WithContext(ctx).Do(&skill); err != nil {
		return nil, err
	}
	return &skill, nil
}

// Delete permanently deletes a skill and all its versions.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := qoderhttp.ValidateID(id); err != nil {
		return err
	}
	return a.client.DELETE("/skills/" + id).WithContext(ctx).Do(nil)
}

// ListVersions returns the version history of a skill.
func (a *API) ListVersions(ctx context.Context, id string, params *types.ListParams) (*types.PaginatedResponse[Skill], error) {
	if err := qoderhttp.ValidateID(id); err != nil {
		return nil, err
	}
	req, err := qoderhttp.ApplyListParams(a.client.GET("/skills/"+id+"/versions"), params)
	if err != nil {
		return nil, err
	}
	var result types.PaginatedResponse[Skill]
	if err := req.WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
