# End-to-End Tests

This directory contains end-to-end tests for `qoder-cloud-agents-go-sdk`. They
exercise the SDK against the real Qoder Cloud Agents API using a Personal Access
Token (`QODER_PAT`).

## ⚠️ 警告：请在非生产/独立测试账号上运行

运行这些测试会在目标账号中创建并删除真实资源，可能产生：

- 账号事件与审计日志
- API 调用计费
- 资源配额消耗
- Webhook 或其他副作用事件

**前置条件**：使用与生产工作负载隔离的组织/项目/账号；账号内无关键生产数据；可独立计费与审计。

## 运行方式

```bash
# 1. 设置 Personal Access Token
export QODER_PAT="pt-your-token-here"

# 2. 确认授权 e2e 写入测试
export QODER_E2E_ACK=1

# 3. 若指向生产域名 api.qoder.com，必须额外确认
export QODER_E2E_PROD_OK=1

# 4. 运行端到端测试（跨包串行，降低限流与配额风险）
go test -tags=e2e -p=1 ./e2e/...

# 5. 默认测试命令不会运行 e2e
go test ./...
```

## 环境变量

| 变量 | 是否必须 | 说明 |
|------|----------|------|
| `QODER_PAT` | 是 | Personal Access Token，用于鉴权 |
| `QODER_E2E_ACK` | 写入用例必须 | 设置为 `1` 表示已了解测试会写入真实资源 |
| `QODER_E2E_PROD_OK` | 生产域名必须 | 当 BaseURL 包含 `api.qoder.com` 时，设置为 `1` 表示已确认使用独立测试账号 |
| `QODER_BASE_URL` | 否 | 覆盖默认 BaseURL，用于代理或未来沙箱端点 |

## 资源命名与清理

- 所有 e2e 创建的可识别名称字段以 `qoder-sdk-e2e-` 为前缀。
- 每次 `Create` 成功后立即将 `{type, id, name}` 追加到 `e2e/.e2e-resources.jsonl`。
- `t.Cleanup` 会在测试结束时尝试清理对应资源。
- 若 Cleanup 失败，资源 ID 仍保留在 `e2e/.e2e-resources.jsonl` 中，可用手动或脚本方式二次清理。

## 残留资源清理脚本

`scripts/cleanup-e2e.sh` 从仓库根目录运行，通过 `go run scripts/cleanup-e2e.go` 读取 `e2e/.e2e-resources.jsonl` 并按资源类型调用 SDK 清理接口。成功清理的资源会从清单中移除；失败的资源保留以便重试。

```bash
# 仅查看待清理资源
./scripts/cleanup-e2e.sh --dry-run

# 真实清理
./scripts/cleanup-e2e.sh
```

脚本要求：

- 从仓库根目录执行。
- 已设置 `QODER_PAT` 与 `QODER_E2E_ACK=1`。
- 依赖 Go 工具链（用于 `go run`），不依赖 Python、`tail -r` 等平台专有命令。
- 幂等：对不存在、已归档或已删除资源视为成功。

## 按前缀手动清理

若 `e2e/.e2e-resources.jsonl` 丢失，可按名称前缀人工定位并清理以 `qoder-sdk-e2e-` 开头的资源。请仔细核对，避免误删非测试资源。

## 故障排查

- 若测试因权限不足跳过：检查 `QODER_PAT` 是否具备 agents/environments/sessions/files/vaults/skills/memorystores 的读写权限，以及 models 的读取权限。
- 若测试因限流失败：降低并发（已默认 `-p=1`），或增加重试退避（修改 `e2e/suite_test.go`）。
- 若 events 测试读取不到事件：只验证 SSE 连接建立，读取超时属于可接受行为。
