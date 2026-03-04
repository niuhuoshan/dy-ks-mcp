package service

import (
	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/platform"
)

const (
	StatusSuccess = "success"
	StatusPartial = "partial"
	StatusBlocked = "blocked"
	StatusFailed  = "failed"
)

type ToolIssue struct {
	Stage         string         `json:"stage"`
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	Retriable     bool           `json:"retriable"`
	RequiresAgent bool           `json:"requires_agent"`
	AgentHints    []string       `json:"agent_hints,omitempty"`
	Artifacts     map[string]any `json:"artifacts,omitempty"`
}

type RunTaskResponse struct {
	Status     string           `json:"status"`
	Result     engine.RunResult `json:"result"`
	Error      *ToolIssue       `json:"error,omitempty"`
	AgentHints []string         `json:"agent_hints,omitempty"`
	Artifacts  map[string]any   `json:"artifacts,omitempty"`
}

type SearchPostsRequest struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sort_by"`
	TimeRange string `json:"time_range"`
	Limit     int    `json:"limit"`
}

type SearchPostsResponse struct {
	Status     string          `json:"status"`
	Platform   string          `json:"platform"`
	AccountID  string          `json:"account_id"`
	Keyword    string          `json:"keyword"`
	SortBy     string          `json:"sort_by"`
	TimeRange  string          `json:"time_range"`
	Posts      []platform.Post `json:"posts,omitempty"`
	Error      *ToolIssue      `json:"error,omitempty"`
	AgentHints []string        `json:"agent_hints,omitempty"`
	Artifacts  map[string]any  `json:"artifacts,omitempty"`
}

type PrepareCommentTargetRequest struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sort_by"`
	TimeRange string `json:"time_range"`
	Limit     int    `json:"limit"`
	Index     int    `json:"index"`
}

type PrepareCommentTargetResponse struct {
	Status     string         `json:"status"`
	Platform   string         `json:"platform"`
	AccountID  string         `json:"account_id"`
	Keyword    string         `json:"keyword"`
	Selected   *platform.Post `json:"selected,omitempty"`
	Candidates int            `json:"candidates"`
	Error      *ToolIssue     `json:"error,omitempty"`
	AgentHints []string       `json:"agent_hints,omitempty"`
}

type SubmitCommentRequest struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
	PostID    string `json:"post_id"`
	PostURL   string `json:"post_url"`
	Content   string `json:"content"`
	Keyword   string `json:"keyword"`
}

type SubmitCommentResponse struct {
	Status     string     `json:"status"`
	Platform   string     `json:"platform"`
	AccountID  string     `json:"account_id"`
	PostID     string     `json:"post_id"`
	Submitted  bool       `json:"submitted"`
	Error      *ToolIssue `json:"error,omitempty"`
	AgentHints []string   `json:"agent_hints,omitempty"`
}

type VerifyCommentResponse struct {
	Status    string     `json:"status"`
	Platform  string     `json:"platform"`
	AccountID string     `json:"account_id"`
	PostID    string     `json:"post_id"`
	Exists    bool       `json:"exists"`
	Error     *ToolIssue `json:"error,omitempty"`
}
