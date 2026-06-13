// Package models provides the Models resource for listing available AI models.
package models

import (
	"context"

	httpclient "github.com/futuretea/go-http-client"
)

// Model represents an available AI model.
type Model struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	DisplayName   string   `json:"display_name"`
	Source        string   `json:"source"`
	IsEnabled     bool     `json:"is_enabled"`
	IsNew         bool     `json:"is_new"`
	PriceFactor   *float64 `json:"price_factor,omitempty"`
	Efforts       []string `json:"efforts,omitempty"`
	DefaultEffort string   `json:"default_effort,omitempty"`
}

// API provides access to the Models resource.
type API struct {
	client httpclient.Client
}

// NewAPI creates a new Models API client.
func NewAPI(client httpclient.Client) *API {
	return &API{client: client}
}

// List returns all available models.
func (a *API) List(ctx context.Context) ([]Model, error) {
	var result struct {
		Data []Model `json:"data"`
	}
	if err := a.client.GET("/models").WithContext(ctx).Do(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
