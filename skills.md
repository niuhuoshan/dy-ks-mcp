# skills.md

本文件定义 `dy-ks-mcp` 给上层 Agent 的推荐技能使用方式。

## 1) 目标

- 让 Agent 在抖音/快手评论任务里按统一 SOP 执行。
- 避免每次临时拼接工具调用链。
- 遇到 `blocked/failed` 时，优先走可接管流程，不盲目重试。

## 2) 推荐技能

### `dy-ks-agent-orchestrator`

- 位置：`skills/dy-ks-agent-orchestrator/SKILL.md`
- 用途：对 MCP 工具做固定编排（登录检查、搜索、选目标、提交、验证）。
- 触发场景：
  - 需要执行抖音/快手评论链路
  - 需要结构化返回结果给上层 Agent
  - 需要 human-in-the-loop 接管

## 3) 执行策略（必须遵守）

- 抖音（Douyin）：
  - 搜索/选目标由浏览器 Agent 手动完成。
  - MCP 负责 `submit_comment` + `verify_comment`。
- 快手（Kuaishou）：
  - 可走 MCP 全链路：`search_posts -> prepare_comment_target -> submit_comment -> verify_comment`。

## 4) 状态与接管规则

- 统一状态：`success | partial | blocked | failed`
- 若 `error.requires_agent=true`：必须交还给 Agent/人工，不得继续硬重试。
- 若命中以下错误码：
  - `AGENT_BROWSER_REQUIRED`：切换浏览器手动步骤
  - `RELAY_NOT_ATTACHED`：先附着 Chrome Relay 再继续
  - `COMMENT_INPUT_NOT_FOUND`：人工打开评论框后重试

## 5) 快速调用示例

```bash
node skills/dy-ks-agent-orchestrator/scripts/orchestrate-comment.mjs \
  --platform kuaishou \
  --keyword "openclaw" \
  --content "你好啊" \
  --account-id default \
  --base-url http://127.0.0.1:18080
```

抖音如需提交评论（完成手动搜索后）：

```bash
node skills/dy-ks-agent-orchestrator/scripts/orchestrate-comment.mjs \
  --platform douyin \
  --content "你好啊" \
  --auto-submit true \
  --post-url "https://www.douyin.com/video/<id>" \
  --account-id default \
  --base-url http://127.0.0.1:18080
```

## 6) 给 Agent 的原则

- 先保证“可执行闭环”，再追求“全自动”。
- 输出必须保留 `error/agent_hints/artifacts` 原始结构，便于上层继续决策。
- 禁止把平台风控/登录异常吞掉，必须显式上报。
