# Qoder Cloud Agent API 文档

> 基于 Qoder 官方 Cloud Agents 文档整理：<https://docs.qoder.com/cloud-agents/overview> 及其 API 页面。  
> 生成日期：2026-06-13

## 1. 概述

Qoder Cloud Agents 是一个托管式 AI Agent 运行时。调用方通过 API 定义可复用的 `Agent`，配置运行用的 `Environment`，创建 `Session`，向 Session 发送事件，并通过事件流实时接收 Agent 的消息、工具调用、思考过程和状态变更。

核心概念：

- `Agent`：可复用的 Agent 配置模板，包含模型、系统提示词、工具、MCP server、Skill 等。
- `Environment`：Session 运行的云端执行环境。
- `Session`：一次具体的 Agent 运行或对话任务。
- `Event`：Session 内的用户消息、Agent 输出、工具调用、状态变化等事件。

官方 API 覆盖资源包括：

- Agents
- Environments
- Sessions
- Events
- Files
- Vaults
- Skills
- Memory Stores
- Models

---

## 2. 全局约定

### 2.1 Base URL

```text
https://api.qoder.com/api/v1/cloud
```

后续接口路径均以该 Base URL 为前缀。例如：

```text
GET https://api.qoder.com/api/v1/cloud/agents
```

### 2.2 鉴权

所有请求都需要 Personal Access Token，放在 `Authorization` 请求头中。

```http
Authorization: Bearer $QODER_PAT
```

推荐本地通过环境变量保存：

```bash
export QODER_PAT="pt-your-token-here"
```

### 2.3 请求格式

JSON 请求使用：

```http
Content-Type: application/json
```

文件和 Skill 上传接口使用 `multipart/form-data`。

### 2.4 幂等性

写操作可使用：

```http
Idempotency-Key: <UUID v4>
```

相同 key 和相同 body 会返回首次结果，不会重复执行。相同 key 但不同 body 通常返回 `409 conflict_error`。幂等 key 适合用于关键资源创建和批量写入。

### 2.5 分页

列表接口使用游标分页。常见参数：

| 参数 | 说明 |
|---|---|
| `limit` | 返回数量，通常默认 20，范围 1–100 |
| `after_id` | 返回指定 ID 之后的数据 |
| `before_id` | 返回指定 ID 之前的数据 |

常见响应结构：

```json
{
  "data": [],
  "first_id": null,
  "last_id": null,
  "has_more": false
}
```

部分资源可能使用 `after` / `before` 命名，但语义相同。

### 2.6 metadata

`metadata` 是调用方自定义的扁平字符串对象。常见约束：

- 最多 16 个键。
- 键最长 64 个 Unicode 字符。
- 值最长 512 个 Unicode 字符。
- 省略时默认为 `{}`。

示例：

```json
{
  "project": "cloud-agent-demo",
  "team": "platform"
}
```

### 2.7 错误结构

错误响应使用统一 envelope。

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "Missing required field: name",
    "param": "name"
  }
}
```

常见错误类型：

| error.type | 说明 |
|---|---|
| `invalid_request_error` | 请求参数无效 |
| `authentication_error` | 鉴权失败 |
| `permission_error` | 权限不足 |
| `not_found_error` | 资源不存在 |
| `conflict_error` | 状态冲突、版本冲突或幂等冲突 |
| `api_error` | 服务端错误 |

---

## 3. 快速调用流程

典型流程：

1. 创建 Environment。
2. 创建 Agent。
3. 创建 Session。
4. 向 Session 发送 `user.message`。
5. 监听 SSE 事件流。

```bash
BASE_URL="https://api.qoder.com/api/v1/cloud"

# 1. 创建 Agent
AGENT_ID=$(curl -s -X POST "$BASE_URL/agents" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "doc-agent",
    "model": "ultimate",
    "system": "You are a documentation assistant.",
    "tools": [
      {
        "type": "agent_toolset_20260401",
        "enabled_tools": ["Read", "Write", "Edit", "Bash", "WebSearch", "WebFetch"]
      }
    ]
  }' | jq -r '.id')

# 2. 创建 Environment
ENV_ID=$(curl -s -X POST "$BASE_URL/environments" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "default-cloud-env",
    "description": "Default cloud execution environment",
    "config": {
      "type": "cloud",
      "networking": { "type": "limited" },
      "packages": { "apt": ["curl"], "pip": [], "npm": [] }
    }
  }' | jq -r '.id')

# 3. 创建 Session
SESSION_ID=$(curl -s -X POST "$BASE_URL/sessions" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Content-Type: application/json" \
  -d "{
    \"agent\": \"$AGENT_ID\",
    \"environment_id\": \"$ENV_ID\",
    \"title\": \"Generate API docs\"
  }" | jq -r '.id')

# 4. 发送用户消息
curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/events" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "type": "user.message",
        "content": "Generate API documentation for this project."
      }
    ]
  }'

# 5. 监听事件流
curl -N "$BASE_URL/sessions/$SESSION_ID/events/stream" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Accept: text/event-stream"
```

---

## 4. Agents API

Agent 是可复用的配置模板，包含模型、系统提示词、工具、MCP server、Skill 绑定和 metadata。

### 4.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/agents` | 列出 Agents |
| `POST` | `/agents` | 创建 Agent |
| `GET` | `/agents/{agent_id}` | 获取单个 Agent |
| `PUT` | `/agents/{agent_id}` | 更新 Agent |
| `POST` | `/agents/{agent_id}/archive` | 归档 Agent |
| `GET` | `/agents/{agent_id}/versions` | 获取 Agent 版本历史 |

### 4.2 创建 Agent

```http
POST /agents
```

请求体示例：

```json
{
  "name": "doc-agent",
  "model": "ultimate",
  "system": "You are a documentation assistant.",
  "description": "Generates API documentation",
  "tools": [
    {
      "type": "agent_toolset_20260401",
      "enabled_tools": ["Bash", "Read", "Write", "Edit", "Glob", "Grep", "WebFetch", "WebSearch"]
    }
  ],
  "mcp_servers": [
    {
      "type": "http",
      "name": "weather-service",
      "url": "https://mcp.example.com/mcp"
    }
  ],
  "skills": [
    {
      "type": "custom",
      "skill_id": "skill_xxx",
      "version": 1
    }
  ],
  "metadata": {
    "project": "docs"
  }
}
```

常见字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---:|---:|---|
| `name` | string | 是 | Agent 名称 |
| `model` | string | 是 | 模型 ID，例如 `ultimate`；建议通过 Models API 获取 |
| `system` | string | 否 | 系统提示词 |
| `description` | string | 否 | Agent 描述 |
| `tools` | array | 否 | 内置工具配置 |
| `mcp_servers` | array | 否 | MCP server 配置 |
| `skills` | array | 否 | Skill 绑定 |
| `metadata` | object | 否 | 自定义 metadata |

### 4.3 Agent 对象

```json
{
  "id": "agent_xxx",
  "type": "agent",
  "name": "doc-agent",
  "model": "ultimate",
  "system": "You are a documentation assistant.",
  "description": "Generates API documentation",
  "tools": [],
  "mcp_servers": [],
  "skills": [],
  "metadata": {},
  "version": 1,
  "archived": false,
  "created_at": "2026-06-13T00:00:00Z",
  "updated_at": "2026-06-13T00:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | `agent_` 前缀 |
| `type` | string | 固定为 `agent` |
| `name` | string | Agent 名称 |
| `model` | string | 模型 ID |
| `system` | string | 系统提示词 |
| `tools` | array | 工具配置 |
| `mcp_servers` | array | MCP server 配置 |
| `skills` | array | Skill 绑定 |
| `metadata` | object | 自定义 metadata |
| `version` | integer | Agent 配置版本 |
| `archived` | boolean | 是否归档 |
| `created_at` | string | 创建时间，UTC |
| `updated_at` | string | 更新时间，UTC |

### 4.4 更新 Agent

```http
PUT /agents/{agent_id}
```

更新 Agent 时需要带当前 `version`，用于乐观并发控制。成功更新后版本号递增。

```json
{
  "version": 1,
  "name": "doc-agent-v2",
  "system": "You are a senior API documentation assistant.",
  "metadata": {
    "project": "docs",
    "stage": "v2"
  }
}
```

### 4.5 内置工具

`agent_toolset_20260401` 常见内置工具：

| 工具 | 说明 |
|---|---|
| `Bash` | 执行 shell 命令 |
| `Read` | 读取文件 |
| `Write` | 写入文件 |
| `Edit` | 编辑文件 |
| `Glob` | 文件路径匹配 |
| `Grep` | 内容搜索 |
| `WebFetch` | 获取网页内容 |
| `WebSearch` | Web 搜索 |
| `DeliverArtifacts` | 交付制品 |

常见权限策略：

| 策略 | 说明 |
|---|---|
| `always_allow` | 始终允许 |
| `always_ask` | 每次询问 |
| `always_deny` | 始终拒绝 |

---

## 5. Environments API

Environment 定义 Session 运行的云容器环境，包括网络策略和依赖包。

### 5.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/environments` | 列出 Environments |
| `POST` | `/environments` | 创建 Environment |
| `GET` | `/environments/{environment_id}` | 获取单个 Environment |
| `PUT` | `/environments/{environment_id}` | 更新 Environment |
| `POST` | `/environments/{environment_id}/archive` | 归档 Environment |

### 5.2 创建 Environment

```http
POST /environments
```

```json
{
  "name": "default-cloud-env",
  "description": "Default environment",
  "config": {
    "type": "cloud",
    "networking": {
      "type": "limited"
    },
    "packages": {
      "apt": ["curl"],
      "pip": ["requests"],
      "npm": []
    }
  },
  "metadata": {
    "team": "platform"
  }
}
```

### 5.3 Environment 对象

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | `env_` 前缀 |
| `type` | string | `environment` |
| `name` | string | 环境名称 |
| `description` | string | 描述 |
| `status` | string | 例如 `ready`、`archived` |
| `config` | object | 云环境配置 |
| `metadata` | object | 自定义 metadata |
| `archived_at` | string/null | 归档时间 |
| `created_at` | string | 创建时间，UTC |
| `updated_at` | string | 更新时间，UTC |

---

## 6. Sessions 与 Events API

Session 是绑定 Agent 和 Environment 后启动的一次具体对话或任务运行实例。Session 创建后可以通过 Events API 发送用户消息，并通过事件列表或 SSE 流读取 Agent 的实时输出。

### 6.1 Sessions 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/sessions` | 列出 Sessions |
| `POST` | `/sessions` | 创建 Session |
| `GET` | `/sessions/{session_id}` | 获取 Session |
| `POST` | `/sessions/{session_id}` | 更新 Session |
| `POST` | `/sessions/{session_id}/archive` | 归档 Session |
| `POST` | `/sessions/{session_id}/cancel` | 取消正在运行的 Session |
| `POST` | `/sessions/{session_id}/resources` | 向 Session 附加资源 |

### 6.2 创建 Session

```http
POST /sessions
```

```json
{
  "agent": "agent_xxx",
  "environment_id": "env_xxx",
  "title": "Implement feature X",
  "metadata": {
    "ticket": "ABC-123"
  },
  "delta_flush_interval_ms": 100,
  "resources": [
    {
      "type": "file",
      "file_id": "file_xxx",
      "path": "/data/input.md"
    },
    {
      "type": "github_repository",
      "url": "https://github.com/example/repo",
      "mount_path": "/workspace/repo"
    }
  ],
  "vault_ids": ["vault_xxx"],
  "memory_store_ids": ["memstore_xxx"],
  "environment_variables": "NODE_ENV=production\nFEATURE_FLAG=true"
}
```

### 6.3 Session 创建字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| `agent` | 是 | Agent ID 字符串，或包含 `id` / `version` 的对象 |
| `environment_id` | 通常是 | 已创建的 Environment ID |
| `environment` | 否 | inline 环境对象；若同时提供 `environment_id`，通常以 `environment_id` 为准 |
| `title` | 否 | Session 标题 |
| `metadata` | 否 | 自定义 metadata |
| `delta_flush_interval_ms` | 否 | SSE 增量刷新间隔；常见范围 50–5000，默认 100 |
| `resources` | 否 | 文件或 GitHub 仓库资源 |
| `vault_ids` | 否 | 绑定 Vault |
| `memory_store_ids` | 否 | 绑定 Memory Store |
| `environment_variables` | 否 | 环境变量字符串 |

### 6.4 Events 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/sessions/{session_id}/events` | 发送事件 |
| `GET` | `/sessions/{session_id}/events` | 按时间顺序列出事件 |
| `GET` | `/sessions/{session_id}/events/stream` | 通过 SSE 获取完整历史事件和后续实时事件 |

### 6.5 发送用户消息

```http
POST /sessions/{session_id}/events
```

```json
{
  "events": [
    {
      "type": "user.message",
      "content": "Please analyze the repository and propose a refactor plan."
    }
  ]
}
```

通常响应 `202 Accepted`，表示事件已接收并异步处理。

### 6.6 常见事件类型

| 类型 | 说明 |
|---|---|
| `user.message` | 用户输入 |
| `user.interrupt` | 中断当前任务 |
| `user.tool_confirmation` | 人工确认工具调用 |
| `user.custom_tool_result` | 返回自定义工具结果 |
| `agent.message` | Agent 输出消息 |
| `agent.thinking` | Agent 思考过程事件 |
| `agent.tool_use` | Agent 内置工具调用 |
| `agent.tool_result` | Agent 内置工具结果 |
| `agent.custom_tool_use` | 自定义工具调用 |
| `agent.mcp_tool_use` | MCP 工具调用 |
| `agent.mcp_tool_result` | MCP 工具结果 |
| `session.status_running` | Session 开始处理 |
| `session.status_idle` | Session 空闲 |
| `session.error` | Session 错误 |
| `agent.artifact_delivered` | Agent 交付文件制品 |

### 6.7 SSE 事件流

```http
GET /sessions/{session_id}/events/stream
Accept: text/event-stream
```

示例：

```bash
curl -N "$BASE_URL/sessions/$SESSION_ID/events/stream" \
  -H "Authorization: Bearer $QODER_PAT" \
  -H "Accept: text/event-stream"
```

SSE 使用标准格式：

```text
id: evt_xxx
event: agent.message
data: {"type":"agent.message","content":"..."}
```

可使用 `Last-Event-ID` 请求头进行断点续传。

---

## 7. Files API

Files API 用于上传文本文件作为 Session 上下文、读取文件 metadata，以及下载 Agent 或工具产物。

### 7.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/files` | 列出文件，可按 `purpose` 过滤 |
| `POST` | `/files` | 上传文本文件 |
| `GET` | `/files/{file_id}` | 获取文件 metadata |
| `GET` | `/files/{file_id}/content` | 获取预签名下载 URL |

### 7.2 上传文件

```bash
curl -X POST "$BASE_URL/files" \
  -H "Authorization: Bearer $QODER_PAT" \
  -F "purpose=user_upload" \
  -F 'metadata={"project":"docs"}' \
  -F "file=@README.md"
```

### 7.3 File 对象

| 字段 | 说明 |
|---|---|
| `id` | `file_` 前缀 |
| `type` | `file` |
| `filename` / `name` | 文件名 |
| `purpose` | `user_upload`、`tool_output`、`skill_output`、`session_resource`、`agent_output` 等 |
| `status` | `uploading`、`ready`、`error`、`deleted` 等 |
| `size` | 字节大小 |
| `metadata` | 自定义 metadata |
| `created_at` | 创建时间，UTC |
| `updated_at` | 更新时间，UTC |

---

## 8. Vaults API

Vault 用于安全保存 MCP server 凭证，并在 Session 中通过 `vault_ids` 绑定使用。创建 Vault 时可一次性写入 credentials；返回对象不会回显明文 `access_token`。

### 8.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/vaults` | 列出 Vaults |
| `POST` | `/vaults` | 创建 Vault |
| `GET` | `/vaults/{vault_id}` | 获取 Vault |
| `POST` | `/vaults/{vault_id}/archive` | 归档 Vault |
| `POST` | `/vaults/{vault_id}/credentials` | 创建 Vault credential |
| `GET` | `/vaults/{vault_id}/credentials` | 列出 credentials；不返回 secret |
| `POST` | `/vaults/{vault_id}/credentials/{credential_id}/archive` | 归档 credential |

### 8.2 创建 Vault

```json
{
  "display_name": "prod-mcp-credentials",
  "credentials": [
    {
      "mcp_server_url": "https://mcp.example.com/mcp",
      "protocol": "streamable_http",
      "type": "static_bearer",
      "access_token": "secret-token"
    }
  ]
}
```

### 8.3 Vault 对象

| 字段 | 说明 |
|---|---|
| `id` | `vault_` 前缀 |
| `type` | `vault` |
| `display_name` | 显示名称 |
| `status` | `active`、`archived` 等 |
| `metadata` | 自定义 metadata |
| `credentials` | credential 列表，不包含明文 token |
| `archived_at` | 归档时间 |
| `created_at` | 创建时间，UTC |
| `updated_at` | 更新时间，UTC |

---

## 9. Skills API

Skill 是 `.zip` 包形式的能力包，必须包含 `SKILL.md`。`SKILL.md` 的 frontmatter 通常需要 `name` 和 `description`。

### 9.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/skills` | 列出 Skills |
| `POST` | `/skills` | 上传 `.zip` 创建 Skill |
| `GET` | `/skills/{skill_id}` | 获取 Skill；可加 `include_content=true` |
| `PUT` | `/skills/{skill_id}` | 更新 metadata，例如 `name`、`description` |
| `DELETE` | `/skills/{skill_id}` | 永久删除 Skill 和所有版本 |
| `GET` | `/skills/{skill_id}/versions` | 获取 Skill 版本描述 |

### 9.2 创建 Skill

```bash
mkdir my-skill
cat > my-skill/SKILL.md <<'SKILL'
---
name: my-custom-skill
description: Custom skill example
version: 1.0.0
---

# My Custom Skill

## Steps
1. Perform action A.
2. Verify the result.
SKILL

cd my-skill && zip ../my-skill.zip SKILL.md && cd ..

curl -X POST "$BASE_URL/skills" \
  -H "Authorization: Bearer $QODER_PAT" \
  -F "type=custom" \
  -F "file=@my-skill.zip"
```

### 9.3 Skill 包约束

| 规则 | 值 |
|---|---|
| 文件格式 | `.zip` |
| 最大大小 | 约 5 MB |
| 必需文件 | `SKILL.md` |
| 必需 frontmatter | `name`、`description` |
| Skill 名称 | 小写字母、数字、连字符、下划线；最长 64 字符 |

### 9.4 Skill 对象

| 字段 | 说明 |
|---|---|
| `id` | `skill_` 前缀 |
| `type` | `skill` |
| `name` | Skill 名称 |
| `description` | Skill 描述 |
| `skill_type` | `prebuilt` 或 `custom` |
| `content_size` | 上传内容大小 |
| `content_sha256` | 内容 SHA-256 |
| `version` | Skill 版本 |
| `status` | 通常为 `active` |
| `content` | 仅 `include_content=true` 时返回，base64 |
| `content_encoding` | 通常为 `base64` |

---

## 10. Memory Stores API

Memory Store 用于跨 Session 持久保存小型文本记忆。Store 下包含 Memory Entry；每次创建、更新、删除 Entry 都会产生不可变 Version。

### 10.1 Memory Store 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/memory_stores` | 列出 Memory Stores |
| `POST` | `/memory_stores` | 创建 Memory Store |
| `GET` | `/memory_stores/{memory_store_id}` | 获取 Memory Store |
| `POST` | `/memory_stores/{memory_store_id}/archive` | 归档 Memory Store |
| `DELETE` | `/memory_stores/{memory_store_id}` | 永久删除 Store 和全部 entries / versions |

### 10.2 Memory Entry 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/memory_stores/{memory_store_id}/memories` | 列出 active entries；通常不返回 `content` |
| `POST` | `/memory_stores/{memory_store_id}/memories` | 创建 Memory Entry |
| `GET` | `/memory_stores/{memory_store_id}/memories/{memory_id}` | 获取 Entry；返回 `content` |
| `PUT` | `/memory_stores/{memory_store_id}/memories/{memory_id}` | 更新 Entry 内容；自动创建新版本 |
| `DELETE` | `/memory_stores/{memory_store_id}/memories/{memory_id}` | 删除 Entry；产生 `deleted` version 记录 |

### 10.3 Memory Version 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/memory_stores/{memory_store_id}/versions` | 列出版本历史；通常不返回 `content` |
| `GET` | `/memory_stores/{memory_store_id}/versions/{version_id}` | 获取版本；未脱敏时返回 `content` |
| `POST` | `/memory_stores/{memory_store_id}/versions/{version_id}/redact` | 永久脱敏版本内容 |

### 10.4 创建 Memory Store

```json
{
  "name": "project-memory",
  "description": "Persistent project notes and decisions",
  "metadata": {
    "project": "cloud-agent-demo"
  }
}
```

### 10.5 创建 Memory Entry

```json
{
  "path": "notes/architecture.md",
  "content": "The service uses event streaming for session updates.",
  "metadata": {
    "source": "design-review"
  }
}
```

### 10.6 更新 Memory Entry

```json
{
  "content": "Updated architecture notes...",
  "content_sha256": "previous-current-content-sha256",
  "metadata": {
    "source": "design-review"
  }
}
```

### 10.7 Memory Store 对象

| 字段 | 说明 |
|---|---|
| `id` | `memstore_` 前缀 |
| `type` | `memory_store` |
| `name` | 名称 |
| `description` | 描述 |
| `status` | `active` 或 `archived` |
| `entry_count` | active entries 数量 |
| `total_size` | active entries 总字节数 |
| `metadata` | 自定义 metadata |
| `archived_at` | 归档时间 |
| `created_at` | 创建时间，UTC |
| `updated_at` | 更新时间，UTC |

### 10.8 Memory Entry 约束

| 字段/规则 | 说明 |
|---|---|
| `path` | 相对路径，不能以 `/` 开头，不能包含 `..` |
| `content` | 必填；trim 后非空；通常最大 100 KB |
| `path` 唯一性 | 同一 Store 下 active entries 的 `path` 唯一 |
| `version` | 初始为 1，每次更新递增 |
| `content_sha256` | 可在更新时作为乐观并发检查 |

---

## 11. Models API

Models API 用于列出当前账号可用的模型 ID。创建或更新 Agent 时的 `model` 字段应使用该接口返回的 `id`。

### 11.1 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/models` | 列出可用模型 |

### 11.2 Model 对象

| 字段 | 说明 |
|---|---|
| `id` | 传给 Agent `model` 的模型 ID |
| `type` | `model` |
| `display_name` | 展示名称 |
| `source` | `system` 或 `user` |
| `is_enabled` | 是否启用 |
| `is_new` | 是否标记为新模型 |
| `price_factor` | 可选价格系数 |
| `efforts` | 可选 reasoning effort，例如 `none`、`low`、`medium`、`high`、`xhigh`、`max` |
| `default_effort` | 默认 effort |

---

## 12. 端点总览

| 资源 | 方法与路径 |
|---|---|
| Agents | `GET /agents` |
| Agents | `POST /agents` |
| Agents | `GET /agents/{agent_id}` |
| Agents | `PUT /agents/{agent_id}` |
| Agents | `POST /agents/{agent_id}/archive` |
| Agents | `GET /agents/{agent_id}/versions` |
| Environments | `GET /environments` |
| Environments | `POST /environments` |
| Environments | `GET /environments/{environment_id}` |
| Environments | `PUT /environments/{environment_id}` |
| Environments | `POST /environments/{environment_id}/archive` |
| Sessions | `GET /sessions` |
| Sessions | `POST /sessions` |
| Sessions | `GET /sessions/{session_id}` |
| Sessions | `POST /sessions/{session_id}` |
| Sessions | `POST /sessions/{session_id}/archive` |
| Sessions | `POST /sessions/{session_id}/cancel` |
| Sessions | `POST /sessions/{session_id}/resources` |
| Events | `POST /sessions/{session_id}/events` |
| Events | `GET /sessions/{session_id}/events` |
| Events | `GET /sessions/{session_id}/events/stream` |
| Files | `GET /files` |
| Files | `POST /files` |
| Files | `GET /files/{file_id}` |
| Files | `GET /files/{file_id}/content` |
| Vaults | `GET /vaults` |
| Vaults | `POST /vaults` |
| Vaults | `GET /vaults/{vault_id}` |
| Vaults | `POST /vaults/{vault_id}/archive` |
| Vault Credentials | `POST /vaults/{vault_id}/credentials` |
| Vault Credentials | `GET /vaults/{vault_id}/credentials` |
| Vault Credentials | `POST /vaults/{vault_id}/credentials/{credential_id}/archive` |
| Skills | `GET /skills` |
| Skills | `POST /skills` |
| Skills | `GET /skills/{skill_id}` |
| Skills | `PUT /skills/{skill_id}` |
| Skills | `DELETE /skills/{skill_id}` |
| Skills | `GET /skills/{skill_id}/versions` |
| Memory Stores | `GET /memory_stores` |
| Memory Stores | `POST /memory_stores` |
| Memory Stores | `GET /memory_stores/{memory_store_id}` |
| Memory Stores | `POST /memory_stores/{memory_store_id}/archive` |
| Memory Stores | `DELETE /memory_stores/{memory_store_id}` |
| Memory Entries | `GET /memory_stores/{memory_store_id}/memories` |
| Memory Entries | `POST /memory_stores/{memory_store_id}/memories` |
| Memory Entries | `GET /memory_stores/{memory_store_id}/memories/{memory_id}` |
| Memory Entries | `PUT /memory_stores/{memory_store_id}/memories/{memory_id}` |
| Memory Entries | `DELETE /memory_stores/{memory_store_id}/memories/{memory_id}` |
| Memory Versions | `GET /memory_stores/{memory_store_id}/versions` |
| Memory Versions | `GET /memory_stores/{memory_store_id}/versions/{version_id}` |
| Memory Versions | `POST /memory_stores/{memory_store_id}/versions/{version_id}/redact` |
| Models | `GET /models` |

---

## 13. 推荐 SDK 封装结构

```text
qoder-cloud-agent-client/
  client.ts
  resources/
    agents.ts
    environments.ts
    sessions.ts
    events.ts
    files.ts
    vaults.ts
    skills.ts
    memory-stores.ts
    models.ts
  types/
    agent.ts
    environment.ts
    session.ts
    event.ts
    file.ts
    vault.ts
    skill.ts
    memory.ts
    model.ts
    common.ts
```

### 13.1 TypeScript 客户端骨架

```ts
export type QoderClientOptions = {
  apiKey: string;
  baseUrl?: string;
};

export class QoderCloudAgentClient {
  readonly baseUrl: string;
  readonly apiKey: string;

  constructor(options: QoderClientOptions) {
    this.apiKey = options.apiKey;
    this.baseUrl = options.baseUrl ?? "https://api.qoder.com/api/v1/cloud";
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        "Content-Type": "application/json",
        ...(init.headers ?? {})
      }
    });

    const text = await response.text();
    const payload = text ? JSON.parse(text) : null;

    if (!response.ok) {
      const message = payload?.error?.message ?? response.statusText;
      throw new Error(`Qoder API error ${response.status}: ${message}`);
    }

    return payload as T;
  }

  listAgents() {
    return this.request("/agents");
  }

  createAgent(body: unknown, idempotencyKey?: string) {
    return this.request("/agents", {
      method: "POST",
      headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {},
      body: JSON.stringify(body)
    });
  }

  createSession(body: unknown, idempotencyKey?: string) {
    return this.request("/sessions", {
      method: "POST",
      headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : {},
      body: JSON.stringify(body)
    });
  }

  sendEvents(sessionId: string, events: unknown[]) {
    return this.request(`/sessions/${sessionId}/events`, {
      method: "POST",
      body: JSON.stringify({ events })
    });
  }
}
```

### 13.2 SDK 实现建议

客户端建议统一封装以下能力：

- Bearer Token 注入。
- JSON 序列化与反序列化。
- `multipart/form-data` 上传。
- 错误 envelope 解析。
- 游标分页。
- 幂等 key。
- SSE 事件解析。
- 5xx 指数退避重试。
- `409 conflict_error` 的版本冲突提示。

---

## 14. 参考链接

- Qoder Cloud Agents Overview: <https://docs.qoder.com/cloud-agents/overview>
- API Conventions: <https://docs.qoder.com/cloud-agents/api/conventions/overview>
- Authentication: <https://docs.qoder.com/cloud-agents/api/conventions/authentication>
- Pagination: <https://docs.qoder.com/cloud-agents/api/conventions/pagination>
- Agents API: <https://docs.qoder.com/cloud-agents/api/agents/create>
- Environments API: <https://docs.qoder.com/cloud-agents/api/environments/create>
- Sessions API: <https://docs.qoder.com/cloud-agents/api/sessions/create>
- Files API: <https://docs.qoder.com/cloud-agents/api/files/list>
- Vaults API: <https://docs.qoder.com/cloud-agents/api/vaults/create>
- Skills API: <https://docs.qoder.com/cloud-agents/api/skills/create>
- Memory Stores API: <https://docs.qoder.com/cloud-agents/api/memory-stores/schemas>
- Models API: <https://docs.qoder.com/cloud-agents/api/models/list>
