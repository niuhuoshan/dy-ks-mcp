package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeRPCError(w, nil, -32600, "only POST is supported")
		return
	}
	defer r.Body.Close()

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, nil, -32700, "invalid json")
		return
	}

	switch req.Method {
	case "initialize":
		writeRPCResult(w, req.ID, map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo": map[string]any{
				"name":    "dy-ks-mcp",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		})
	case "tools/list":
		writeRPCResult(w, req.ID, map[string]any{
			"tools": []any{
				map[string]any{
					"name":        "check_login_status",
					"description": "Check login status by platform/account.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"platform":   map[string]any{"type": "string"},
							"account_id": map[string]any{"type": "string"},
						},
						"required": []string{"platform"},
					},
				},
				map[string]any{
					"name":        "start_login",
					"description": "Start login flow by platform/account.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"platform":   map[string]any{"type": "string"},
							"account_id": map[string]any{"type": "string"},
						},
						"required": []string{"platform"},
					},
				},
				map[string]any{
					"name":        "run_comment_task",
					"description": "Run one search-comment pipeline task.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"platform":   map[string]any{"type": "string"},
							"account_id": map[string]any{"type": "string"},
							"keyword":    map[string]any{"type": "string"},
							"sort_by": map[string]any{
								"type":        "string",
								"enum":        []string{"comprehensive", "latest"},
								"default":     "comprehensive",
								"description": "search sort mode",
							},
							"time_range": map[string]any{
								"type":    "string",
								"enum":    []string{"all", "day", "week", "month", "year"},
								"default": "all",
							},
							"limit": map[string]any{"type": "integer", "minimum": 1},
						},
						"required": []string{"platform", "keyword"},
					},
				},
			},
		})
	case "tools/call":
		result, err := h.callTool(r.Context(), req.Params)
		if err != nil {
			writeRPCError(w, req.ID, -32000, err.Error())
			return
		}
		writeRPCResult(w, req.ID, result)
	default:
		writeRPCError(w, req.ID, -32601, "method not found")
	}
}

func (h *Handler) callTool(ctx context.Context, raw json.RawMessage) (any, error) {
	var params toolCallParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params")
	}

	switch params.Name {
	case "check_login_status":
		platformName, err := readStringArg(params.Arguments, "platform", true)
		if err != nil {
			return nil, err
		}
		accountID, _ := readStringArg(params.Arguments, "account_id", false)
		status, err := h.svc.CheckLoginStatus(ctx, platformName, accountID)
		if err != nil {
			return nil, err
		}
		return toolResult(status), nil
	case "start_login":
		platformName, err := readStringArg(params.Arguments, "platform", true)
		if err != nil {
			return nil, err
		}
		accountID, _ := readStringArg(params.Arguments, "account_id", false)
		if err := h.svc.StartLogin(ctx, platformName, accountID); err != nil {
			return nil, err
		}
		return toolResult(map[string]any{
			"platform":   platformName,
			"account_id": defaultAccount(accountID),
			"started":    true,
		}), nil
	case "run_comment_task":
		platformName, err := readStringArg(params.Arguments, "platform", true)
		if err != nil {
			return nil, err
		}
		keyword, err := readStringArg(params.Arguments, "keyword", true)
		if err != nil {
			return nil, err
		}
		accountID, _ := readStringArg(params.Arguments, "account_id", false)
		sortBy, err := readStringArg(params.Arguments, "sort_by", false)
		if err != nil {
			return nil, err
		}
		timeRange, err := readStringArg(params.Arguments, "time_range", false)
		if err != nil {
			return nil, err
		}
		limit, err := readIntArg(params.Arguments, "limit", false)
		if err != nil {
			return nil, err
		}
		result, err := h.svc.RunCommentTask(ctx, engine.RunRequest{
			Platform:  platformName,
			AccountID: accountID,
			Keyword:   keyword,
			SortBy:    sortBy,
			TimeRange: timeRange,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		return toolResult(result), nil
	default:
		return nil, fmt.Errorf("unknown tool %q", params.Name)
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func toolResult(v any) map[string]any {
	b, _ := json.Marshal(v)
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": string(b),
			},
		},
	}
}

func readStringArg(args map[string]any, key string, required bool) (string, error) {
	v, ok := args[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing argument %q", key)
		}
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be string", key)
	}
	if s == "" && required {
		return "", fmt.Errorf("argument %q cannot be empty", key)
	}
	return s, nil
}

func readIntArg(args map[string]any, key string, required bool) (int, error) {
	v, ok := args[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing argument %q", key)
		}
		return 0, nil
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	default:
		return 0, fmt.Errorf("argument %q must be integer", key)
	}
}

func defaultAccount(accountID string) string {
	if accountID == "" {
		return "default"
	}
	return accountID
}

func writeRPCResult(w http.ResponseWriter, id any, result any) {
	writeRPC(w, rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeRPCError(w http.ResponseWriter, id any, code int, msg string) {
	writeRPC(w, rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: msg,
		},
	})
}

func writeRPC(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
