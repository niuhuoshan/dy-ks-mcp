# dy-ks-mcp

Go 1.22+ skeleton for a future Douyin + Kuaishou automated comment system.

## What is included

- HTTP server entrypoint: `cmd/server/main.go`
- YAML config loader with validation: `internal/config`
- SQLite store with comment dedupe key: `(platform, account_id, post_id)`
- Execution engine pipeline:
  - search results (keyword + sort + time range)
  - filter
  - dedupe check
  - rate limit / random pause / time window / circuit breaker checks
  - comment
  - persistence
- Platform abstraction `Login / CheckLogin / Search / Comment`
- Real browser automation clients (Playwright worker):
  - `internal/platform/douyin`
  - `internal/platform/kuaishou`
- Runtime policy:
  - no mock/generated post data
  - no in-memory fake login state
  - browser action failures return real runtime errors
- Selector loading from `selectors/*.yaml`
- REST API endpoints:
  - `GET /health`
  - `POST /api/v1/run` (best-effort orchestrator, structured status; default only prepares target, `auto_submit=true` to auto comment)
  - `POST /api/v1/search`
  - `POST /api/v1/comment/prepare`
  - `POST /api/v1/comment/submit`
  - `GET|POST /api/v1/comment/verify`
  - `GET /api/v1/login/status?platform=&account_id=`
  - `GET|POST /api/v1/login/start?platform=&account_id=`
- MCP endpoint:
  - `POST /mcp`
  - tools: `check_login_status`, `start_login`, `search_posts`, `prepare_comment_target`, `submit_comment`, `verify_comment`, `run_comment_task`
- Search params:
  - `sort_by`: `comprehensive` | `latest`
  - `time_range`: `all` | `day` | `week` | `month` | `year`

## Agent-first contract

This project is designed as an MCP tool layer for upper agents (OpenClaw/Codex/etc), not a fully autonomous script.

Core response fields:

- `status`: `success` | `partial` | `blocked` | `failed`
- `error`: `{stage, code, message, retriable, requires_agent, agent_hints, artifacts}`
- `agent_hints`: suggested next actions for orchestration logic
- `artifacts`: runtime context payload for troubleshooting/handoff
- built-in runtime guardrails: per-request hard timeouts + relay-attachment fast-fail (`TIMEOUT_BLOCKED`, `RELAY_NOT_ATTACHED`)

Platform policy:

- `douyin`: search/target selection is agent-browser-led; MCP handles `submit_comment`/`verify_comment` only.
- `kuaishou`: MCP can still provide search + target preparation.

## Reusable orchestrator skill

- skill: `../skills/dy-ks-agent-orchestrator/SKILL.md`
- flow script: `../skills/dy-ks-agent-orchestrator/scripts/orchestrate-comment.mjs`
- douyin SOP doc: `../skills/dy-ks-agent-orchestrator/SOP-douyin-semi-auto.md`
- douyin SOP script: `../skills/dy-ks-agent-orchestrator/scripts/douyin-semi-auto-sop.mjs`
- run example:
- douyin note: search is browser-agent manual; then call `submit_comment` with `post_url`/`post_id`

```bash
node ../skills/dy-ks-agent-orchestrator/scripts/orchestrate-comment.mjs \
  --platform kuaishou \
  --keyword "搞笑" \
  --content "你好啊" \
  --account-id default \
  --base-url http://127.0.0.1:18080
```

```bash
node ../skills/dy-ks-agent-orchestrator/scripts/douyin-semi-auto-sop.mjs \
  --mode prepare \
  --keyword openclaw \
  --account-id default \
  --base-url http://127.0.0.1:18080
```

```bash
node ../skills/dy-ks-agent-orchestrator/scripts/douyin-semi-auto-sop.mjs \
  --mode submit \
  --keyword openclaw \
  --content "你好啊" \
  --post-url "https://www.douyin.com/video/<id>" \
  --account-id default \
  --base-url http://127.0.0.1:18080
```

## Quick start

```bash
cd /home/zhang/.openclaw/workspace/dy-ks-mcp
go mod tidy
go run ./cmd/server -config ./config/config.yaml
```

Server default address: `0.0.0.0:18080`

## Browser runtime

Platform automation uses a Node Playwright worker script:

- worker script: `tools/platform-browser.mjs`
- config path: `platform.browser` in `config/config.yaml`

Recommended setup for your environment:

- keep `platform.browser.ws_url` set to OpenClaw relay CDP endpoint (`ws://127.0.0.1:18792/cdp`)
- keep the Chrome Relay extension connected on the target tab
- use the same logged-in browser profile for stable login state

## REST examples

```bash
curl -s http://127.0.0.1:18080/health
```

```bash
curl -s "http://127.0.0.1:18080/api/v1/login/start?platform=douyin&account_id=test"
```

```bash
curl -s "http://127.0.0.1:18080/api/v1/login/status?platform=douyin&account_id=test"
```

```bash
curl -s -X POST http://127.0.0.1:18080/api/v1/run \
  -H 'Content-Type: application/json' \
  -d '{
    "platform": "douyin",
    "account_id": "test",
    "keyword": "automation",
    "sort_by": "latest",
    "time_range": "week",
    "limit": 5,
    "target_index": 0,
    "auto_submit": false
  }'
```

```bash
curl -s -X POST http://127.0.0.1:18080/api/v1/search \
  -H 'Content-Type: application/json' \
  -d '{
    "platform": "kuaishou",
    "account_id": "test",
    "keyword": "automation",
    "sort_by": "latest",
    "time_range": "week",
    "limit": 3
  }'
```

```bash
curl -s -X POST http://127.0.0.1:18080/api/v1/comment/submit \
  -H 'Content-Type: application/json' \
  -d '{
    "platform": "douyin",
    "account_id": "test",
    "post_url": "https://www.douyin.com/video/7612984938296782954",
    "content": "你好啊"
  }'
```

## MCP examples

List tools:

```bash
curl -s -X POST http://127.0.0.1:18080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call a tool:

```bash
curl -s -X POST http://127.0.0.1:18080/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc":"2.0",
    "id":2,
    "method":"tools/call",
    "params":{
      "name":"run_comment_task",
      "arguments":{
        "platform":"kuaishou",
        "account_id":"test",
        "keyword":"automation",
        "sort_by":"latest",
        "time_range":"week",
        "limit":3,
        "target_index":0,
        "auto_submit":false
      }
    }
  }'
```

## Browser probing (local Playwright)

Use local browser automation probe (not OpenClaw browser relay):

```bash
cd /home/zhang/.openclaw/workspace/dy-ks-mcp
node ./tools/probe-selectors.mjs
```

Probe output is used to update `selectors/douyin.yaml` and `selectors/kuaishou.yaml`.

Reference docs:

- `docs/browser-probe-2026-03-03.md`
- `docs/agent-first-sop.md`
- `docs/architecture-flow.md`

## Notes for future go-rod integration

- Implement browser automation in:
  - `internal/platform/douyin/client.go`
  - `internal/platform/kuaishou/client.go`
- Keep current `platform.Client` interface unchanged for minimal upper-layer impact.
- Map selectors from `selectors/*.yaml` to real rod element queries.
- Persist login session/cookies in real storage.
- Extend error typing in platform layer so engine can classify retryable errors.
