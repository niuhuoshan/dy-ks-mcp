# dy-ks-mcp

> 默认语言：中文（可切换英文）
>
> - 中文（当前）：`README.md`
> - English: `README.en.md`

`dy-ks-mcp` 是一个面向上层 AI Agent（OpenClaw/Codex）的抖音/快手评论自动化 MCP 工具层。
核心目标不是“无脑全自动”，而是“可编排、可接管、可观测”。

## 项目定位

- **Agent-first**：把复杂/不稳定步骤交给 Agent 决策和接管。
- **结构化结果**：统一返回 `success|partial|blocked|failed`。
- **可恢复执行**：失败返回 `error + agent_hints + artifacts`，方便继续执行。

## 能力概览

- 登录相关：`check_login_status`、`start_login`
- 搜索与选目标：`search_posts`、`prepare_comment_target`
- 评论与校验：`submit_comment`、`verify_comment`
- 一体化编排：`run_comment_task`

## 平台策略

- **抖音（Douyin）**：搜索/选目标走浏览器 Agent 手动步骤，MCP 负责 `submit_comment + verify_comment`
- **快手（Kuaishou）**：支持 MCP 搜索、选目标、提交、验证全链路

## 目录结构

- 启动入口：`cmd/server/main.go`
- 配置：`config/config.yaml`、`internal/config/`
- REST API：`internal/httpapi/`
- MCP 协议层：`internal/mcp/`
- 业务编排层：`internal/service/`
- 执行引擎：`internal/engine/`
- 平台实现：`internal/platform/`
- 浏览器 Worker：`internal/platform/worker/`、`tools/platform-browser.mjs`
- 存储（去重）：`internal/store/`、`data/dy-ks-mcp.db`

## 快速开始

```bash
cd dy-ks-mcp
go mod tidy
go run ./cmd/server -config ./config/config.yaml
```

默认地址：`http://127.0.0.1:18080`

## 常用接口

- `GET /health`
- `POST /api/v1/run`
- `POST /api/v1/search`
- `POST /api/v1/comment/prepare`
- `POST /api/v1/comment/submit`
- `GET|POST /api/v1/comment/verify`
- `GET /api/v1/login/status`
- `GET|POST /api/v1/login/start`
- `POST /mcp`

## 文档（中英文切换）

- 中文总览：`docs/说明文档.zh-CN.md`
- English Guide: `docs/Guide.en.md`
- Agent-first SOP: `docs/agent-first-sop.md`
- 架构流程图：`docs/architecture-flow.md`
- Skill 规范：`skills.md`
- 安装规范（Agent）：`install.md`

## 说明

本项目已加入运行时保护：请求硬超时、relay 未附着快速失败，避免“卡死无响应”。
