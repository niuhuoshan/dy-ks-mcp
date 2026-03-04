# Agent-First Operation SOP

This project is designed for upper AI agents (OpenClaw/Codex) to orchestrate browser work, not for blind full automation.

## Platform policy

- Douyin: agent/browser performs search + target selection; MCP executes `submit_comment` + `verify_comment`.
- Kuaishou: MCP supports `search_posts` + `prepare_comment_target` + `submit_comment` + `verify_comment`.

## Standard flow

1. `check_login_status`
2. `search_posts` (Kuaishou) or manual search (Douyin)
3. `prepare_comment_target` (Kuaishou) or manual target pick (Douyin)
4. `submit_comment`
5. `verify_comment`

## Failure handling

- `AGENT_BROWSER_REQUIRED`: switch to browser-agent/manual step and continue.
- `COMMENT_INPUT_NOT_FOUND`: manually open comment panel, retry submit.
- `RELAY_NOT_ATTACHED`: attach Chrome relay tab, then retry.
- `TIMEOUT_BLOCKED`: retry once, then hand over to browser-agent manual path.

## Closure criteria

A task is closed-loop when submit succeeds and `verify_comment.exists=true`.
