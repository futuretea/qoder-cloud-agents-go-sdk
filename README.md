# qoder-cloud-agents-go-sdk

Go SDK for the [Qoder Cloud Agents API](https://docs.qoder.com/cloud-agents/overview).

[中文 README](README.zh-CN.md)

[![CI](https://github.com/futuretea/qoder-cloud-agents-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/futuretea/qoder-cloud-agents-go-sdk/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/futuretea/qoder-cloud-agents-go-sdk)](go.mod)
[![Codecov](https://codecov.io/gh/futuretea/qoder-cloud-agents-go-sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/futuretea/qoder-cloud-agents-go-sdk)
[![License](https://img.shields.io/github/license/futuretea/qoder-cloud-agents-go-sdk)](LICENSE)

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Common Use Cases](#common-use-cases)
  - [Upload a file and attach it to a session](#upload-a-file-and-attach-it-to-a-session)
  - [Create a vault and bind it to a session](#create-a-vault-and-bind-it-to-a-session)
  - [Persist memory across sessions](#persist-memory-across-sessions)
- [SSE Streaming](#sse-streaming)
- [Resources](#resources)
- [Error Handling](#error-handling)
- [Advanced Configuration](#advanced-configuration)
- [Development](#development)
- [Project Structure](#project-structure)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [API Documentation](#api-documentation)
- [License](#license)

## Requirements

- Go 1.24.1 or later
- A Qoder Personal Access Token. Create one in your [Qoder account settings](https://docs.qoder.com/cloud-agents/api/conventions/authentication).

## Installation

```bash
go get github.com/futuretea/qoder-cloud-agents-go-sdk
```

## Quick Start

This minimal example creates an environment, an agent, a session, and sends a message.

```go
package main

import (
    "context"
    "log"

    qoder "github.com/futuretea/qoder-cloud-agents-go-sdk"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
)

func main() {
    ctx := context.Background()
    client := qoder.New("pt-your-token-here") // replace with your token

    // Create an environment
    env, err := client.Environments().Create(ctx,
        environments.NewCreateRequest("default-cloud-env", environments.EnvConfig{
            Type: "cloud",
            Networking: environments.Networking{Type: "limited"},
        }).WithDescription("Default cloud execution environment"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create an agent
    agent, err := client.Agents().Create(ctx,
        agents.NewCreateRequest("doc-agent", "ultimate").
            WithSystem("You are a documentation assistant.").
            WithTool(agents.Tool{
                Type:         "agent_toolset_20260401",
                EnabledTools: []string{"Read", "Write", "Edit", "Bash"},
            }),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create a session
    session, err := client.Sessions().Create(ctx,
        sessions.NewCreateRequest(agent.ID).
            WithEnvironment(env.ID).
            WithTitle("Generate docs"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Send a message
    err = client.Events().SendMessage(ctx, session.ID, "Generate API documentation.")
    if err != nil {
        log.Fatal(err)
    }
}
```

For more runnable examples, see [`example_test.go`](example_test.go) and the [Common Use Cases](#common-use-cases) section below.

## Common Use Cases

### Upload a file and attach it to a session

```go
import (
    "github.com/futuretea/qoder-cloud-agents-go-sdk/files"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
)

// Upload a file.
file, err := client.Files().Upload(ctx, &files.UploadFileRequest{
    Filename: "input.md",
    Data:     []byte("# Requirements\n"),
    Purpose:  "user_upload",
})
if err != nil {
    log.Fatal(err)
}

// Attach it when creating a session.
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("Review document").
        WithResource(sessions.NewResourceFile(file.ID, "/data/input.md")),
)
```

### Create a vault and bind it to a session

```go
import (
    "github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/vaults"
)

// Create a vault with a static bearer credential.
vault, err := client.Vaults().Create(ctx,
    vaults.NewCreateRequest("prod-mcp-credentials").
        WithCredential(vaults.CreateCredential{
            MCPServerURL: "https://mcp.example.com/mcp",
            Protocol:     "streamable_http",
            Type:         "static_bearer",
            AccessToken:  "secret-token",
        }),
)
if err != nil {
    log.Fatal(err)
}

// Bind the vault to a session so the agent can call the MCP server.
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("MCP session").
        WithVault(vault.ID),
)
```

### Persist memory across sessions

```go
import "github.com/futuretea/qoder-cloud-agents-go-sdk/memorystores"

// Create a memory store.
store, err := client.MemoryStores().Create(ctx,
    memorystores.NewCreateStoreRequest("project-memory").
        WithDescription("Persistent project notes"),
)
if err != nil {
    log.Fatal(err)
}

// Add an entry.
entry, err := client.MemoryStores().CreateEntry(ctx, store.ID,
    memorystores.NewCreateEntryRequest("notes/arch.md",
        "The service uses event streaming for session updates."),
)
if err != nil {
    log.Fatal(err)
}

// Bind the store to a session for recall.
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("Memory-aware session").
        WithMemoryStore(store.ID),
)
```

## SSE Streaming

Receive agent events in real time using Server-Sent Events (SSE).

```go
import (
    "io"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
)

resp, err := client.Events().Stream(ctx, sessionID)
if err != nil {
    log.Fatal(err)
}
stream := qoderhttp.NewSSEStream(resp)
defer stream.Close()

for {
    evt, err := stream.Next(ctx)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("[%s] %s: %s", evt.ID, evt.Event, string(evt.Data))
}
```

## Resources

| Resource | Package | Operations |
|---|---|---|
| Agents | `agents` | List, Create, Get, Update, Archive, Delete, ListVersions |
| Environments | `environments` | List, Create, Get, Update, Archive, Delete |
| Sessions | `sessions` | List, Create, Get, Update, Archive, Cancel, AddResources, Delete |
| Events | `events` | Send, SendMessage, List, Stream (SSE) |
| Files | `files` | List, Upload, Get, GetContent, Delete |
| Vaults | `vaults` | List, Create, Get, Archive, CreateCredential, ListCredentials, ArchiveCredential |
| Skills | `skills` | List, Create, Get, Update, Delete, ListVersions |
| Memory Stores | `memorystores` | Store/Entry/Version CRUD |
| Models | `models` | List |

## Error Handling

API errors are parsed into `*qoderhttp.APIError`, which provides helpers for common HTTP status codes.

```go
import "github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"

agent, err := client.Agents().Get(ctx, "agent_xxx")
if err != nil {
    if apiErr, ok := qoderhttp.IsAPIError(err); ok {
        switch {
        case apiErr.IsNotFound():
            log.Println("Agent not found")
        case apiErr.IsConflict():
            log.Println("Version conflict — re-fetch and retry")
        case apiErr.IsUnauthorized():
            log.Println("Invalid or expired token")
        default:
            log.Printf("API error: %s", apiErr.Error())
        }
    }
    return
}
```

## Advanced Configuration

### Custom HTTP Client

```go
import (
    "net/http"
    "time"
)

client := qoder.New("pt-your-token-here",
    qoder.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

### Custom Base URL

```go
client := qoder.New("pt-your-token-here",
    qoder.WithBaseURL("https://custom-proxy.example.com/api/v1/cloud"),
)
```

## Development

Run tests, lint and build locally with `make`:

```bash
make all        # lint + test + build
make test       # run all tests with race detector and coverage
make lint       # run golangci-lint
make ci         # full CI pipeline
```

Additional runnable examples are in [`example_test.go`](example_test.go).

## Project Structure

```
qoder-cloud-agents-go-sdk/
├── qoder.go              # Main Client
├── qoderhttp/            # HTTP helpers built on github.com/futuretea/go-http-client
│   ├── client.go         # Client factory, Config, Option, and fluent request helpers
│   ├── client_test.go    # HTTP client tests
│   ├── errors.go         # API error parsing
│   └── sse.go            # SSE stream parser
├── types/                # Common types (pagination, metadata)
├── agents/               # Agents resource
├── environments/         # Environments resource
├── sessions/             # Sessions resource
├── events/               # Events resource + SSE streaming
├── files/                # Files resource + multipart upload
├── vaults/               # Vaults resource + credentials
├── skills/               # Skills resource + .zip upload
├── memorystores/         # Memory Stores (Store/Entry/Version)
└── models/               # Models resource
```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `401 Unauthorized` | Invalid or expired token | Create a new Personal Access Token and update `qoder.New("pt-...")`. |
| `404 Not Found` | Resource ID does not exist or is archived | Verify the ID and check whether the resource has been archived. |
| `409 Conflict` | Optimistic concurrency version mismatch | Re-fetch the resource, then retry the update with the latest `Version`. |
| SSE stream stops suddenly | Context cancelled or response body closed | Ensure the stream is not closed early; handle `ctx.Err()` gracefully. |

## Contributing

Contributions are welcome.

1. Open an issue to discuss large changes.
2. Fork the repository and create a feature branch.
3. Run `make ci` locally to ensure lint, tests, and build pass.
4. Submit a pull request with a clear description.

Please follow the existing code style and keep changes focused.

## API Documentation

- [Qoder Cloud Agents Overview](https://docs.qoder.com/cloud-agents/overview)
- [API Conventions](https://docs.qoder.com/cloud-agents/api/conventions/overview)
- [Authentication](https://docs.qoder.com/cloud-agents/api/conventions/authentication)

## License

MIT License
