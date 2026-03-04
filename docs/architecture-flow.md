# Architecture Flow

```mermaid
flowchart TD
    A[Client / Agent Request] --> B{Entry Type}
    B -->|REST| C[/api/v1/*]
    B -->|MCP| D[/mcp tools/call]

    C --> E[HTTP Handler]
    D --> F[MCP Handler]
    E --> G[Service Layer]
    F --> G

    G --> H{Platform}
    H -->|Douyin| I[Agent-led Search/Target Select]
    I --> J[submit_comment\n(post_id or post_url)]

    H -->|Kuaishou| K[search_posts]
    K --> L[prepare_comment_target]
    L --> J

    J --> M[Worker Adapter]
    M --> N[Node Playwright Worker]
    N --> O[Chrome Relay/CDP Browser]

    J --> P[Save Comment Record]
    P --> Q[SQLite Store]
    Q --> R[verify_comment]

    G --> S[Structured Response]
    S --> T[status: success|partial|blocked|failed]
    S --> U[error + agent_hints + artifacts]

    O -. runtime issue .-> V[Fast-fail Guardrails]
    V --> U
```
