# qoder-cloud-agents-go-sdk

Go SDK for the [Qoder Cloud Agents API](https://docs.qoder.com/cloud-agents/overview).

## Installation

```bash
go get github.com/futuretea/qoder-cloud-agents-go-sdk
```

## Quick Start

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
    client := qoder.New("pt-your-token-here")

    // Create an environment
    env, err := client.Environments().Create(ctx,
        environments.NewCreateRequest("default-cloud-env").
            WithDescription("Default cloud execution environment").
            WithConfig(environments.EnvConfig{
                Type: "cloud",
                Networking: environments.Networking{Type: "limited"},
            }),
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

## SSE Streaming

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
| Agents | `agents` | List, Create, Get, Update, Archive, ListVersions |
| Environments | `environments` | List, Create, Get, Update, Archive |
| Sessions | `sessions` | List, Create, Get, Update, Archive, Cancel, AddResources |
| Events | `events` | Send, SendMessage, List, Stream (SSE) |
| Files | `files` | List, Upload, Get, GetContent |
| Vaults | `vaults` | List, Create, Get, Archive, CreateCredential, ListCredentials, ArchiveCredential |
| Skills | `skills` | List, Create, Get, Update, Delete, ListVersions |
| Memory Stores | `memorystores` | Store/Entry/Version CRUD |
| Models | `models` | List |

## Error Handling

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
import "net/http"

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
├── qoderhttp/            # Internal HTTP client (zero deps)
│   ├── client.go         # Client, Config, Option, fluent request builder
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

## API Documentation

- [Qoder Cloud Agents Overview](https://docs.qoder.com/cloud-agents/overview)
- [API Conventions](https://docs.qoder.com/cloud-agents/api/conventions/overview)
- [Authentication](https://docs.qoder.com/cloud-agents/api/conventions/authentication)

## License

MIT License
