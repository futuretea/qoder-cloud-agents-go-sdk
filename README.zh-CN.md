# qoder-cloud-agents-go-sdk

[Qoder Cloud Agents API](https://docs.qoder.com/cloud-agents/overview) 的 Go SDK。

[English README](README.md)

[![CI](https://github.com/futuretea/qoder-cloud-agents-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/futuretea/qoder-cloud-agents-go-sdk/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/futuretea/qoder-cloud-agents-go-sdk)](go.mod)
[![Codecov](https://codecov.io/gh/futuretea/qoder-cloud-agents-go-sdk/branch/main/graph/badge.svg)](https://codecov.io/gh/futuretea/qoder-cloud-agents-go-sdk)
[![License](https://img.shields.io/github/license/futuretea/qoder-cloud-agents-go-sdk)](LICENSE)

## 目录

- [环境要求](#环境要求)
- [安装](#安装)
- [快速开始](#快速开始)
- [常见用例](#常见用例)
  - [上传文件并附加到会话](#上传文件并附加到会话)
  - [创建 Vault 并绑定到会话](#创建-vault-并绑定到会话)
  - [跨会话持久化记忆](#跨会话持久化记忆)
- [SSE 流式接收](#sse-流式接收)
- [资源与操作](#资源与操作)
- [错误处理](#错误处理)
- [高级配置](#高级配置)
- [本地开发](#本地开发)
- [项目结构](#项目结构)
- [故障排查](#故障排查)
- [贡献指南](#贡献指南)
- [API 文档](#api-文档)
- [许可证](#许可证)

## 环境要求

- Go 1.24.1 或更高版本
- 一个 Qoder Personal Access Token。请在 [Qoder 账户设置](https://docs.qoder.com/cloud-agents/api/conventions/authentication) 中创建。

## 安装

```bash
go get github.com/futuretea/qoder-cloud-agents-go-sdk
```

## 快速开始

以下最小示例创建运行环境、Agent、会话，并发送一条消息。

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
    client := qoder.New("pt-your-token-here") // 替换为你的 token

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

更多可运行示例见 [`example_test.go`](example_test.go) 和下方的 [常见用例](#常见用例) 章节。

## 常见用例

### 上传文件并附加到会话

```go
import (
    "github.com/futuretea/qoder-cloud-agents-go-sdk/files"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
)

// 上传文件。
file, err := client.Files().Upload(ctx, &files.UploadFileRequest{
    Filename: "input.md",
    Data:     []byte("# Requirements\n"),
    Purpose:  "user_upload",
})
if err != nil {
    log.Fatal(err)
}

// 创建会话时附加文件。
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("Review document").
        WithResource(sessions.NewResourceFile(file.ID, "/data/input.md")),
)
```

### 创建 Vault 并绑定到会话

```go
import (
    "github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
    "github.com/futuretea/qoder-cloud-agents-go-sdk/vaults"
)

// 创建包含 static bearer 凭证的 Vault。
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

// 将会话绑定到 Vault，使 Agent 可以调用 MCP server。
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("MCP session").
        WithVault(vault.ID),
)
```

### 跨会话持久化记忆

```go
import "github.com/futuretea/qoder-cloud-agents-go-sdk/memorystores"

// 创建记忆存储。
store, err := client.MemoryStores().Create(ctx,
    memorystores.NewCreateStoreRequest("project-memory").
        WithDescription("Persistent project notes"),
)
if err != nil {
    log.Fatal(err)
}

// 添加一条记忆。
entry, err := client.MemoryStores().CreateEntry(ctx, store.ID,
    memorystores.NewCreateEntryRequest("notes/arch.md",
        "The service uses event streaming for session updates."),
)
if err != nil {
    log.Fatal(err)
}

// 绑定到会话以便回忆。
session, err := client.Sessions().Create(ctx,
    sessions.NewCreateRequest("agent_xxx").
        WithEnvironment("env_xxx").
        WithTitle("Memory-aware session").
        WithMemoryStore(store.ID),
)
```

## SSE 流式接收

使用 Server-Sent Events (SSE) 实时接收 Agent 事件。

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

API 错误会被解析为 `*qoderhttp.APIError`，并提供常见 HTTP 状态码的便捷判断方法。

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

## 故障排查

| 现象 | 可能原因 | 解决方法 |
|---|---|---|
| `401 Unauthorized` | Token 无效或已过期 | 创建新的 Personal Access Token 并更新 `qoder.New("pt-...")` 中的值。 |
| `404 Not Found` | 资源 ID 不存在或已被归档 | 确认 ID 正确，并检查资源是否已被归档。 |
| `409 Conflict` | 乐观锁版本冲突 | 重新 Get 资源，然后使用最新的 `Version` 重试更新。 |
| SSE 流突然中断 | Context 被取消或响应体被关闭 | 确保没有提前关闭流；优雅处理 `ctx.Err()`。 |

## 贡献指南

欢迎贡献。

1. 如有较大改动，请先开 Issue 讨论。
2. Fork 仓库并创建功能分支。
3. 本地运行 `make ci`，确保 lint、测试和构建均通过。
4. 提交 Pull Request，并写清楚改动描述。

请遵循现有代码风格，并保持改动聚焦。

## API 文档

- [Qoder Cloud Agents 概览](https://docs.qoder.com/cloud-agents/overview)
- [API 约定](https://docs.qoder.com/cloud-agents/api/conventions/overview)
- [认证方式](https://docs.qoder.com/cloud-agents/api/conventions/authentication)

## 许可证

MIT License
