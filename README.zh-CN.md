# qoder-cloud-agents-go-sdk

[Qoder Cloud Agents API](https://docs.qoder.com/cloud-agents/overview) 的 Go SDK。

[English README](README.md)

## 环境要求

- Go 1.24.1 或更高版本

## 安装

```bash
go get github.com/futuretea/qoder-cloud-agents-go-sdk
```

## 快速开始

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

    // 创建运行环境
    env, err := client.Environments().Create(ctx,
        environments.NewCreateRequest("default-cloud-env", environments.EnvConfig{
            Type: "cloud",
            Networking: environments.Networking{Type: "limited"},
        }).WithDescription("Default cloud execution environment"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 创建 Agent
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

    // 创建会话
    session, err := client.Sessions().Create(ctx,
        sessions.NewCreateRequest(agent.ID).
            WithEnvironment(env.ID).
            WithTitle("Generate docs"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 发送消息
    err = client.Events().SendMessage(ctx, session.ID, "Generate API documentation.")
    if err != nil {
        log.Fatal(err)
    }
}
```

## SSE 流式接收

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

## 资源与操作

| 资源 | 包 | 支持操作 |
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

## 错误处理

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

## 高级配置

### 自定义 HTTP Client

```go
import (
    "net/http"
    "time"
)

client := qoder.New("pt-your-token-here",
    qoder.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

### 自定义 Base URL

```go
client := qoder.New("pt-your-token-here",
    qoder.WithBaseURL("https://custom-proxy.example.com/api/v1/cloud"),
)
```

## 本地开发

使用 `make` 运行测试、lint 和构建：

```bash
make all        # lint + test + build
make test       # 运行所有测试（带 race detector 和覆盖率）
make lint       # 运行 golangci-lint
make ci         # 完整 CI 流水线
```

更多可运行示例见 [`example_test.go`](example_test.go)。

## 项目结构

```
qoder-cloud-agents-go-sdk/
├── qoder.go              # 主 Client
├── qoderhttp/            # 基于 github.com/futuretea/go-http-client 的 HTTP 辅助库
│   ├── client.go         # Client 工厂、Config、Option 与流式请求辅助函数
│   ├── client_test.go    # HTTP client 测试
│   ├── errors.go         # API 错误解析
│   └── sse.go            # SSE 流解析器
├── types/                # 公共类型（分页、元数据）
├── agents/               # Agents 资源
├── environments/         # Environments 资源
├── sessions/             # Sessions 资源
├── events/               # Events 资源 + SSE 流式
├── files/                # Files 资源 + 多文件上传
├── vaults/               # Vaults 资源 + 凭证管理
├── skills/               # Skills 资源 + .zip 上传
├── memorystores/         # Memory Stores（Store/Entry/Version）
└── models/               # Models 资源
```

## API 文档

- [Qoder Cloud Agents 概览](https://docs.qoder.com/cloud-agents/overview)
- [API 约定](https://docs.qoder.com/cloud-agents/api/conventions/overview)
- [认证方式](https://docs.qoder.com/cloud-agents/api/conventions/authentication)

## 许可证

MIT License
