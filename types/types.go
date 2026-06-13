// Package types provides common types used across all Qoder Cloud Agents API resources.
//
//nolint:revive // Package name 'types' is appropriate for shared type definitions
package types

import (
	"fmt"
	"net/url"
	"strconv"
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

// Validate checks that ListParams values are within allowed ranges.
// Limit=0 (the zero value) is treated as "not set" / "use API default" and is valid.
// Returns an error if Limit is outside the valid range of 0-100.
func (p ListParams) Validate() error {
	if p.Limit < 0 || p.Limit > 100 {
		return fmt.Errorf("types: ListParams.Limit must be between 0 and 100 (got %d)", p.Limit)
	}
	return nil
}

// ToQuery converts ListParams to URL query parameters.
// Zero/empty values are omitted.
func (p ListParams) ToQuery() url.Values {
	q := url.Values{}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
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

// Validate checks that the Metadata map satisfies the documented constraints.
// Returns an error if the number of keys exceeds 16, or any key/value exceeds
// the maximum length.
func (m Metadata) Validate() error {
	if len(m) > 16 {
		return fmt.Errorf("types: Metadata has %d keys, max is 16", len(m))
	}
	for k, v := range m {
		if len(k) > 64 {
			return fmt.Errorf("types: Metadata key %q exceeds 64 chars", k)
		}
		if len(v) > 512 {
			return fmt.Errorf("types: Metadata value for key %q exceeds 512 chars", k)
		}
	}
	return nil
}
