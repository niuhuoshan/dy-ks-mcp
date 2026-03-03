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
- Placeholder clients:
  - `internal/platform/douyin`
  - `internal/platform/kuaishou`
- Selector loading from `selectors/*.yaml`
- REST API endpoints:
  - `GET /health`
  - `POST /api/v1/run`
  - `GET /api/v1/login/status?platform=&account_id=`
  - `GET|POST /api/v1/login/start?platform=&account_id=`
- MCP endpoint:
  - `POST /mcp`
  - tools: `check_login_status`, `start_login`, `run_comment_task`
- Search params:
  - `sort_by`: `comprehensive` | `latest`
  - `time_range`: `all` | `day` | `week` | `month` | `year`

## Quick start

```bash
cd /home/zhang/.openclaw/workspace/dy-ks-mcp
go mod tidy
go run ./cmd/server -config ./config/config.yaml
```

Server default address: `0.0.0.0:8080`

## REST examples

```bash
curl -s http://127.0.0.1:8080/health
```

```bash
curl -s "http://127.0.0.1:8080/api/v1/login/start?platform=douyin&account_id=test"
```

```bash
curl -s "http://127.0.0.1:8080/api/v1/login/status?platform=douyin&account_id=test"
```

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/run \
  -H 'Content-Type: application/json' \
  -d '{
    "platform": "douyin",
    "account_id": "test",
    "keyword": "automation",
    "sort_by": "latest",
    "time_range": "week",
    "limit": 5
  }'
```

## MCP examples

List tools:

```bash
curl -s -X POST http://127.0.0.1:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call a tool:

```bash
curl -s -X POST http://127.0.0.1:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc":"2.0",
    "id":2,
    "method":"tools/call",
    "params":{
      "name":"run_comment_task",
      "arguments":{
        "platform":"douyin",
        "account_id":"test",
        "keyword":"automation",
        "sort_by":"latest",
        "time_range":"week",
        "limit":3
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

Reference report:

- `docs/browser-probe-2026-03-03.md`

## Notes for future go-rod integration

- Replace placeholder implementations in:
  - `internal/platform/douyin/client.go`
  - `internal/platform/kuaishou/client.go`
- Keep current `platform.Client` interface unchanged for minimal upper-layer impact.
- Map selectors from `selectors/*.yaml` to real rod element queries.
- Persist login session/cookies outside in-memory placeholder state.
- Extend error typing in platform layer so engine can classify retryable errors.
