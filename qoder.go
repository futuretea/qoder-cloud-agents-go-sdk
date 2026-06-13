// Package qoder provides a Go client for the Qoder Cloud Agents API.
// See: https://docs.qoder.com/cloud-agents/overview
//
// Usage:
//
//	client := qoder.New("pt-your-token-here")
//	agents, err := client.Agents().List(ctx, nil)
package qoder

import (
	"net/http"
	"sync"

	httpclient "github.com/futuretea/go-http-client"

	"github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/events"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/files"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/memorystores"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/models"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/skills"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/vaults"
)

const (
	defaultBaseURL = "https://api.qoder.com/api/v1/cloud"
)

// Client is the main entry point for the Qoder Cloud Agents API.
type Client struct {
	http       httpclient.Client
	token      string
	baseURL    string
	httpClient httpclient.Doer // custom HTTP client preserved across option rebuilds

	agentsOnce       sync.Once
	agents           *agents.API
	environmentsOnce sync.Once
	environments     *environments.API
	sessionsOnce     sync.Once
	sessions         *sessions.API
	eventsOnce       sync.Once
	events           *events.API
	filesOnce        sync.Once
	files            *files.API
	vaultsOnce       sync.Once
	vaults           *vaults.API
	skillsOnce       sync.Once
	skills           *skills.API
	memoryStoresOnce sync.Once
	memoryStores     *memorystores.API
	modelsOnce       sync.Once
	models           *models.API
}

// Option configures a Client.
type Option func(*Client)

// rebuildHTTP recreates the internal HTTP client from the current Client state.
// It is called by Option functions after mutating configuration fields.
func (c *Client) rebuildHTTP() {
	c.http = qoderhttp.NewClient(&qoderhttp.Config{
		BaseURL:    c.baseURL,
		Token:      c.token,
		HTTPClient: c.httpClient,
	})
}

// WithHTTPClient sets a custom *http.Client for connection pooling or testing.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
		c.rebuildHTTP()
	}
}

// WithBaseURL sets a custom base URL for testing or proxy scenarios.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
		c.rebuildHTTP()
	}
}

// New creates a new Qoder Cloud Agents client with the given Personal Access Token.
// The token can be created at https://docs.qoder.com/cloud-agents/overview
func New(token string, opts ...Option) *Client {
	c := &Client{
		token:   token,
		baseURL: defaultBaseURL,
		http: qoderhttp.NewClient(&qoderhttp.Config{
			BaseURL: defaultBaseURL,
			Token:   token,
		}),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Agents returns the Agents API client (lazy-initialized via sync.Once).
func (c *Client) Agents() *agents.API {
	c.agentsOnce.Do(func() {
		c.agents = agents.NewAPI(c.http)
	})
	return c.agents
}

// Environments returns the Environments API client (lazy-initialized via sync.Once).
func (c *Client) Environments() *environments.API {
	c.environmentsOnce.Do(func() {
		c.environments = environments.NewAPI(c.http)
	})
	return c.environments
}

// Sessions returns the Sessions API client (lazy-initialized via sync.Once).
func (c *Client) Sessions() *sessions.API {
	c.sessionsOnce.Do(func() {
		c.sessions = sessions.NewAPI(c.http)
	})
	return c.sessions
}

// Events returns the Events API client (lazy-initialized via sync.Once).
func (c *Client) Events() *events.API {
	c.eventsOnce.Do(func() {
		c.events = events.NewAPI(c.http)
	})
	return c.events
}

// Files returns the Files API client (lazy-initialized via sync.Once).
func (c *Client) Files() *files.API {
	c.filesOnce.Do(func() {
		c.files = files.NewAPI(c.http)
	})
	return c.files
}

// Vaults returns the Vaults API client (lazy-initialized via sync.Once).
func (c *Client) Vaults() *vaults.API {
	c.vaultsOnce.Do(func() {
		c.vaults = vaults.NewAPI(c.http)
	})
	return c.vaults
}

// Skills returns the Skills API client (lazy-initialized via sync.Once).
func (c *Client) Skills() *skills.API {
	c.skillsOnce.Do(func() {
		c.skills = skills.NewAPI(c.http)
	})
	return c.skills
}

// MemoryStores returns the Memory Stores API client (lazy-initialized via sync.Once).
func (c *Client) MemoryStores() *memorystores.API {
	c.memoryStoresOnce.Do(func() {
		c.memoryStores = memorystores.NewAPI(c.http)
	})
	return c.memoryStores
}

// Models returns the Models API client (lazy-initialized via sync.Once).
func (c *Client) Models() *models.API {
	c.modelsOnce.Do(func() {
		c.models = models.NewAPI(c.http)
	})
	return c.models
}
