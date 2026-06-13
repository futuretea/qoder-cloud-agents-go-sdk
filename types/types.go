// Package types provides common types used across all Qoder Cloud Agents API resources.
//
//nolint:revive // Package name 'types' is appropriate for shared type definitions
package types

import (
	"fmt"
	"net/url"
)

// PaginatedResponse is a generic cursor-paginated list response.
// The Qoder API returns lists in this envelope:
//
//	{"data": [...], "first_id": null, "last_id": null, "has_more": false}
type PaginatedResponse[T any] struct {
	Data    []T     `json:"data"`
	FirstID *string `json:"first_id,omitempty"`
	LastID  *string `json:"last_id,omitempty"`
	HasMore bool    `json:"has_more"`
}

// ListParams holds cursor-based pagination parameters for list endpoints.
// Common parameters: limit (1-100, default 20), after_id, before_id.
type ListParams struct {
	Limit    int
	AfterID  string
	BeforeID string
}

// ToQuery converts ListParams to URL query parameters.
// Zero/empty values are omitted.
func (p ListParams) ToQuery() url.Values {
	q := url.Values{}
	if p.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", p.Limit))
	}
	if p.AfterID != "" {
		q.Set("after_id", p.AfterID)
	}
	if p.BeforeID != "" {
		q.Set("before_id", p.BeforeID)
	}
	return q
}

// Metadata is a flat key-value map for attaching custom metadata to resources.
// Constraints: max 16 keys, key max 64 chars, value max 512 chars.
type Metadata map[string]string
