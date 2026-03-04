# dy-ks-mcp Guide (English)

> Language switch:
>
> - 中文（默认）: `docs/说明文档.zh-CN.md`
> - English (current): `docs/Guide.en.md`

## 1. Overview

`dy-ks-mcp` is an MCP tool layer for upper AI agents to execute Douyin/Kuaishou comment tasks.

Key principles:

- Agent-first orchestration
- Structured errors for handoff/recovery
- Fail-fast guardrails instead of silent hanging

## 2. Execution policy

- Douyin: semi-automatic (agent/browser handles search+target, MCP handles submit+verify)
- Kuaishou: can run fully automated pipeline

## 3. Core tools

- `check_login_status`
- `start_login`
- `search_posts`
- `prepare_comment_target`
- `submit_comment`
- `verify_comment`
- `run_comment_task`

## 4. Status contract

Unified status values:

- `success`
- `partial`
- `blocked`
- `failed`

Error payload:

- `error.stage`
- `error.code`
- `error.message`
- `error.retriable`
- `error.requires_agent`
- `error.agent_hints`
- `error.artifacts`

## 5. Anti-hang guardrails

Built-in protections:

- Request-level hard timeouts
- Worker-level execution timeouts
- Relay fast-fail when no attached browser tab (`RELAY_NOT_ATTACHED`)

## 6. Recommended closure flow

1. `check_login_status`
2. Search & choose target (manual on Douyin, automated on Kuaishou)
3. `submit_comment`
4. `verify_comment`

A task is considered closed-loop when submit succeeds and `verify_comment.exists=true`.
