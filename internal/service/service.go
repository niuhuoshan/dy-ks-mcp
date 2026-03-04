package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"dy-ks-mcp/internal/engine"
	"dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/platform/registry"
	"dy-ks-mcp/internal/store"
)

type Service struct {
	registry *registry.Registry
	runner   *engine.Runner
	repo     store.Repository
}

var (
	douyinVideoIDPattern   = regexp.MustCompile(`/video/([0-9A-Za-z_-]+)`)
	kuaishouShortIDPattern = regexp.MustCompile(`/short-video/([0-9A-Za-z_-]+)`)
	kuaishouVideoIDPattern = regexp.MustCompile(`/video/([0-9A-Za-z_-]+)`)
	kuaishouPhotoIDPattern = regexp.MustCompile(`[?&]photoId=([0-9A-Za-z_-]+)`)
)

func New(reg *registry.Registry, runner *engine.Runner, repo store.Repository) *Service {
	return &Service{registry: reg, runner: runner, repo: repo}
}

func (s *Service) RunCommentTask(ctx context.Context, req engine.RunRequest) (engine.RunResult, error) {
	client, err := s.client(req.Platform)
	if err != nil {
		return engine.RunResult{}, err
	}
	if req.AccountID == "" {
		req.AccountID = "default"
	}
	return s.runner.Run(ctx, client, req)
}

func (s *Service) RunCommentTaskWithStatus(ctx context.Context, req engine.RunRequest, autoSubmit bool, targetIndex int) RunTaskResponse {
	if isDouyin(req.Platform) {
		return s.planDouyinForAgent(req)
	}
	if !autoSubmit {
		return s.planRunForAgent(ctx, req, targetIndex)
	}

	result := RunTaskResponse{
		Result: engine.RunResult{
			Platform:  strings.ToLower(strings.TrimSpace(req.Platform)),
			AccountID: defaultAccount(req.AccountID),
			Keyword:   req.Keyword,
			SortBy:    platform.NormalizeSortBy(req.SortBy),
			TimeRange: platform.NormalizeTimeRange(req.TimeRange),
		},
	}

	runResult, err := s.RunCommentTask(ctx, req)
	if err != nil {
		issue := classifyIssue(stageFromErrorMessage(err.Error()), err)
		result.Error = issue
		result.Status = statusFrom(runResult, issue)
		result.AgentHints = issue.AgentHints
		result.Artifacts = issue.Artifacts
		return result
	}

	result.Result = runResult
	issue := issueFromRunResult(runResult)
	result.Error = issue
	result.Status = statusFrom(runResult, issue)
	if issue != nil {
		result.AgentHints = issue.AgentHints
		result.Artifacts = issue.Artifacts
	}
	if len(result.AgentHints) == 0 && result.Status == StatusPartial && runResult.Searched == 0 {
		result.AgentHints = []string{"调整关键词、排序或时间范围后重试 search_posts"}
	}
	return result
}

func (s *Service) planRunForAgent(ctx context.Context, req engine.RunRequest, targetIndex int) RunTaskResponse {
	result := RunTaskResponse{
		Result: engine.RunResult{
			Platform:  strings.ToLower(strings.TrimSpace(req.Platform)),
			AccountID: defaultAccount(req.AccountID),
			Keyword:   req.Keyword,
			SortBy:    platform.NormalizeSortBy(req.SortBy),
			TimeRange: platform.NormalizeTimeRange(req.TimeRange),
		},
	}

	search := s.SearchPosts(ctx, SearchPostsRequest{
		Platform:  req.Platform,
		AccountID: req.AccountID,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		TimeRange: req.TimeRange,
		Limit:     req.Limit,
	})

	if search.Error != nil {
		result.Status = search.Status
		result.Error = search.Error
		result.AgentHints = search.AgentHints
		result.Artifacts = search.Artifacts
		return result
	}

	result.Result.Searched = len(search.Posts)
	if len(search.Posts) == 0 {
		result.Status = StatusPartial
		result.AgentHints = []string{"未找到候选帖子，建议调整关键词或时间范围后重试 search_posts"}
		return result
	}

	if targetIndex < 0 {
		targetIndex = 0
	}
	if targetIndex >= len(search.Posts) {
		issue := &ToolIssue{
			Stage:         "prepare_target",
			Code:          "TARGET_INDEX_OUT_OF_RANGE",
			Message:       fmt.Sprintf("target_index %d out of range (candidates=%d)", targetIndex, len(search.Posts)),
			Retriable:     false,
			RequiresAgent: true,
			AgentHints:    []string{"将 target_index 调整到候选范围内后重试"},
		}
		result.Status = StatusFailed
		result.Error = issue
		result.AgentHints = issue.AgentHints
		return result
	}

	selected := search.Posts[targetIndex]
	preview := search.Posts
	if len(preview) > 5 {
		preview = preview[:5]
	}
	result.Status = StatusPartial
	result.AgentHints = []string{
		"已完成搜索和目标选择，请由 AI Agent/人工确认目标页面后调用 submit_comment",
		"若页面结构异常，优先使用 browser 工具手动定位评论输入框后再提交",
	}
	result.Artifacts = map[string]any{
		"selected_post":   selected,
		"candidate_posts": preview,
		"next_tool":       "submit_comment",
	}
	return result
}

func (s *Service) planDouyinForAgent(req engine.RunRequest) RunTaskResponse {
	platformName := strings.ToLower(strings.TrimSpace(req.Platform))
	if platformName == "" {
		platformName = "douyin"
	}
	keyword := strings.TrimSpace(req.Keyword)
	searchURL := "https://www.douyin.com/search?type=video"
	if keyword != "" {
		searchURL = fmt.Sprintf("https://www.douyin.com/search/%s?type=video", keyword)
	}

	result := RunTaskResponse{
		Status: StatusBlocked,
		Result: engine.RunResult{
			Platform:  platformName,
			AccountID: defaultAccount(req.AccountID),
			Keyword:   keyword,
			SortBy:    platform.NormalizeSortBy(req.SortBy),
			TimeRange: platform.NormalizeTimeRange(req.TimeRange),
		},
		Error: &ToolIssue{
			Stage:         "search",
			Code:          "AGENT_BROWSER_REQUIRED",
			Message:       "douyin search is agent-led; use browser tool to search and pick target post manually",
			Retriable:     false,
			RequiresAgent: true,
			AgentHints: []string{
				"在浏览器中打开抖音搜索页，并手动设置 最新 + 一周内",
				"从目标视频 URL 提取 post_id 后调用 submit_comment（或直接传 post_url）",
			},
			Artifacts: map[string]any{
				"search_url": searchURL,
				"next_tool":  "submit_comment",
			},
		},
		AgentHints: []string{
			"在浏览器中打开抖音搜索页，并手动设置 最新 + 一周内",
			"从目标视频 URL 提取 post_id 后调用 submit_comment（或直接传 post_url）",
		},
		Artifacts: map[string]any{
			"search_url": searchURL,
			"next_tool":  "submit_comment",
		},
	}
	return result
}

func (s *Service) SearchPosts(ctx context.Context, req SearchPostsRequest) SearchPostsResponse {
	resp := SearchPostsResponse{
		Platform:  strings.ToLower(strings.TrimSpace(req.Platform)),
		AccountID: defaultAccount(req.AccountID),
		Keyword:   req.Keyword,
		SortBy:    platform.NormalizeSortBy(req.SortBy),
		TimeRange: platform.NormalizeTimeRange(req.TimeRange),
	}

	if isDouyin(req.Platform) {
		searchURL := "https://www.douyin.com/search?type=video"
		if strings.TrimSpace(req.Keyword) != "" {
			searchURL = fmt.Sprintf("https://www.douyin.com/search/%s?type=video", strings.TrimSpace(req.Keyword))
		}
		issue := &ToolIssue{
			Stage:         "search",
			Code:          "AGENT_BROWSER_REQUIRED",
			Message:       "douyin search is disabled in MCP; use browser agent to perform search and select target",
			RequiresAgent: true,
			AgentHints: []string{
				"使用 browser 工具打开抖音搜索页，手动设为 最新 + 一周内",
				"确认目标视频后，将 post_url/post_id 交给 submit_comment",
			},
			Artifacts: map[string]any{
				"search_url": searchURL,
				"next_tool":  "submit_comment",
			},
		}
		resp.Status = StatusBlocked
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		resp.Artifacts = issue.Artifacts
		return resp
	}

	client, err := s.client(req.Platform)
	if err != nil {
		issue := classifyIssue("input", err)
		resp.Status = StatusFailed
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		resp.Artifacts = issue.Artifacts
		return resp
	}

	if strings.TrimSpace(req.Keyword) == "" {
		issue := classifyIssue("input", fmt.Errorf("keyword is required"))
		resp.Status = StatusFailed
		resp.Error = issue
		return resp
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	posts, err := client.Search(ctx, resp.AccountID, platform.SearchQuery{
		Keyword:   req.Keyword,
		SortBy:    resp.SortBy,
		TimeRange: resp.TimeRange,
		Limit:     limit,
	})
	if err != nil {
		issue := classifyIssue("search", err)
		resp.Status = statusFrom(engine.RunResult{}, issue)
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		resp.Artifacts = issue.Artifacts
		return resp
	}

	resp.Posts = posts
	if len(posts) == 0 {
		resp.Status = StatusPartial
		resp.AgentHints = []string{"没有搜索到帖子，建议调整 keyword/sort_by/time_range 后重试"}
		return resp
	}
	resp.Status = StatusSuccess
	return resp
}

func (s *Service) PrepareCommentTarget(ctx context.Context, req PrepareCommentTargetRequest) PrepareCommentTargetResponse {
	resp := PrepareCommentTargetResponse{
		Platform:  strings.ToLower(strings.TrimSpace(req.Platform)),
		AccountID: defaultAccount(req.AccountID),
		Keyword:   req.Keyword,
	}

	if isDouyin(req.Platform) {
		issue := &ToolIssue{
			Stage:         "prepare_target",
			Code:          "AGENT_BROWSER_REQUIRED",
			Message:       "douyin target selection is agent-led; pass selected post_url/post_id to submit_comment",
			RequiresAgent: true,
			AgentHints: []string{
				"在浏览器中手动选择抖音目标视频",
				"调用 submit_comment 时传 post_url 或 post_id",
			},
		}
		resp.Status = StatusBlocked
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		return resp
	}

	search := s.SearchPosts(ctx, SearchPostsRequest{
		Platform:  req.Platform,
		AccountID: req.AccountID,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		TimeRange: req.TimeRange,
		Limit:     req.Limit,
	})

	if search.Error != nil {
		resp.Status = search.Status
		resp.Error = search.Error
		resp.AgentHints = search.AgentHints
		return resp
	}

	resp.Candidates = len(search.Posts)
	if len(search.Posts) == 0 {
		issue := &ToolIssue{
			Stage:         "prepare_target",
			Code:          "NO_CANDIDATE_POSTS",
			Message:       "no candidate posts from search",
			Retriable:     true,
			RequiresAgent: true,
			AgentHints:    []string{"先调用 search_posts 调整检索条件，再选择目标帖子"},
		}
		resp.Status = StatusBlocked
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		return resp
	}

	index := req.Index
	if index < 0 {
		index = 0
	}
	if index >= len(search.Posts) {
		issue := &ToolIssue{
			Stage:         "prepare_target",
			Code:          "TARGET_INDEX_OUT_OF_RANGE",
			Message:       fmt.Sprintf("index %d out of range (candidates=%d)", index, len(search.Posts)),
			Retriable:     false,
			RequiresAgent: true,
			AgentHints:    []string{"将 index 调整到候选范围内后重试"},
		}
		resp.Status = StatusFailed
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		return resp
	}

	selected := search.Posts[index]
	resp.Selected = &selected
	resp.Status = StatusSuccess
	return resp
}

func (s *Service) SubmitComment(ctx context.Context, req SubmitCommentRequest) SubmitCommentResponse {
	resp := SubmitCommentResponse{
		Platform:  strings.ToLower(strings.TrimSpace(req.Platform)),
		AccountID: defaultAccount(req.AccountID),
		PostID:    req.PostID,
	}

	client, err := s.client(req.Platform)
	if err != nil {
		issue := classifyIssue("input", err)
		resp.Status = StatusFailed
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		return resp
	}
	resolvedPostID, resolveErr := resolvePostID(req.Platform, req.PostID, req.PostURL)
	if resolveErr != nil {
		issue := classifyIssue("input", resolveErr)
		resp.Status = StatusFailed
		resp.Error = issue
		return resp
	}
	resp.PostID = resolvedPostID

	if strings.TrimSpace(req.Content) == "" {
		issue := classifyIssue("input", fmt.Errorf("content is required"))
		resp.Status = StatusFailed
		resp.Error = issue
		return resp
	}

	err = client.Comment(ctx, resp.AccountID, platform.CommentRequest{
		PostID:  resolvedPostID,
		Content: req.Content,
	})
	if err != nil {
		issue := classifyIssue("comment", err)
		resp.Status = statusFrom(engine.RunResult{}, issue)
		resp.Error = issue
		resp.AgentHints = issue.AgentHints
		return resp
	}
	resp.Submitted = true
	resp.Status = StatusSuccess

	if s.repo != nil {
		saveErr := s.repo.SaveComment(ctx, store.CommentRecord{
			Platform:  resp.Platform,
			AccountID: resp.AccountID,
			PostID:    resolvedPostID,
			Keyword:   req.Keyword,
			Comment:   req.Content,
			CreatedAt: time.Now().UTC(),
		})
		if saveErr != nil && !errors.Is(saveErr, store.ErrDuplicate) {
			issue := classifyIssue("store", saveErr)
			resp.Status = StatusPartial
			resp.Error = issue
			resp.AgentHints = append(resp.AgentHints, "评论已提交但入库失败，可稍后 verify_comment")
		}
	}
	return resp
}

func (s *Service) VerifyComment(ctx context.Context, platformName string, accountID string, postID string) VerifyCommentResponse {
	resp := VerifyCommentResponse{
		Platform:  strings.ToLower(strings.TrimSpace(platformName)),
		AccountID: defaultAccount(accountID),
		PostID:    postID,
	}

	if strings.TrimSpace(platformName) == "" || strings.TrimSpace(postID) == "" {
		resp.Status = StatusFailed
		resp.Error = classifyIssue("input", fmt.Errorf("platform and post_id are required"))
		return resp
	}
	if s.repo == nil {
		resp.Status = StatusFailed
		resp.Error = classifyIssue("verify", fmt.Errorf("repository unavailable"))
		return resp
	}

	exists, err := s.repo.HasCommented(ctx, resp.Platform, resp.AccountID, postID)
	if err != nil {
		resp.Status = StatusFailed
		resp.Error = classifyIssue("verify", err)
		return resp
	}
	resp.Exists = exists
	if exists {
		resp.Status = StatusSuccess
	} else {
		resp.Status = StatusPartial
	}
	return resp
}

func (s *Service) CheckLoginStatus(ctx context.Context, platformName string, accountID string) (platform.LoginStatus, error) {
	client, err := s.client(platformName)
	if err != nil {
		return platform.LoginStatus{}, err
	}
	if accountID == "" {
		accountID = "default"
	}
	return client.CheckLogin(ctx, accountID)
}

func (s *Service) StartLogin(ctx context.Context, platformName string, accountID string) error {
	client, err := s.client(platformName)
	if err != nil {
		return err
	}
	if accountID == "" {
		accountID = "default"
	}
	return client.Login(ctx, accountID)
}

func (s *Service) SupportedPlatforms() []string {
	return s.registry.Names()
}

func (s *Service) client(platformName string) (platform.Client, error) {
	name := strings.ToLower(strings.TrimSpace(platformName))
	if name == "" {
		return nil, fmt.Errorf("platform is required")
	}
	return s.registry.Get(name)
}

func defaultAccount(accountID string) string {
	if accountID == "" {
		return "default"
	}
	return accountID
}

func isDouyin(platformName string) bool {
	return strings.EqualFold(strings.TrimSpace(platformName), "douyin")
}

func resolvePostID(platformName string, postID string, postURL string) (string, error) {
	if strings.TrimSpace(postID) != "" {
		return strings.TrimSpace(postID), nil
	}
	url := strings.TrimSpace(postURL)
	if url == "" {
		return "", fmt.Errorf("post_id or post_url is required")
	}

	platformName = strings.ToLower(strings.TrimSpace(platformName))
	if platformName == "douyin" {
		if match := douyinVideoIDPattern.FindStringSubmatch(url); len(match) == 2 {
			return match[1], nil
		}
		return "", fmt.Errorf("cannot parse douyin post_id from post_url")
	}
	if platformName == "kuaishou" {
		if match := kuaishouShortIDPattern.FindStringSubmatch(url); len(match) == 2 {
			return match[1], nil
		}
		if match := kuaishouVideoIDPattern.FindStringSubmatch(url); len(match) == 2 {
			return match[1], nil
		}
		if match := kuaishouPhotoIDPattern.FindStringSubmatch(url); len(match) == 2 {
			return match[1], nil
		}
		return "", fmt.Errorf("cannot parse kuaishou post_id from post_url")
	}
	return "", fmt.Errorf("unsupported platform %q", platformName)
}

func issueFromRunResult(result engine.RunResult) *ToolIssue {
	if len(result.Errors) == 0 {
		return nil
	}
	msg := result.Errors[0]
	return classifyIssue(stageFromErrorMessage(msg), errors.New(msg))
}

func statusFrom(result engine.RunResult, issue *ToolIssue) string {
	if issue != nil {
		if issue.RequiresAgent && result.Commented == 0 {
			return StatusBlocked
		}
		if result.Commented > 0 {
			return StatusPartial
		}
		return StatusFailed
	}
	if result.Failures > 0 || result.Commented < result.Attempted {
		if result.Commented > 0 {
			return StatusPartial
		}
		return StatusFailed
	}
	if result.Searched == 0 || result.Attempted == 0 {
		return StatusPartial
	}
	return StatusSuccess
}

func stageFromErrorMessage(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "search posts") || strings.Contains(lower, "search"):
		return "search"
	case strings.Contains(lower, "comment"):
		return "comment"
	case strings.Contains(lower, "login"):
		return "login"
	case strings.Contains(lower, "rate limit"):
		return "rate_limit"
	case strings.Contains(lower, "platform") || strings.Contains(lower, "keyword") || strings.Contains(lower, "post_id"):
		return "input"
	case strings.Contains(lower, "store") || strings.Contains(lower, "save"):
		return "store"
	default:
		return "run"
	}
}

func classifyIssue(stage string, err error) *ToolIssue {
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	issue := &ToolIssue{
		Stage:         stage,
		Code:          "RUNTIME_ERROR",
		Message:       msg,
		Retriable:     false,
		RequiresAgent: false,
	}

	switch {
	case strings.Contains(lower, "unsupported platform") || strings.Contains(lower, "platform is required"):
		issue.Code = "INVALID_PLATFORM"
	case strings.Contains(lower, "keyword is required") || strings.Contains(lower, "post_id is required") || strings.Contains(lower, "post_id or post_url is required") || strings.Contains(lower, "content is required") || strings.Contains(lower, "cannot parse"):
		issue.Code = "INVALID_INPUT"
	case strings.Contains(lower, "worker timed out") || strings.Contains(lower, "context deadline exceeded") || strings.Contains(lower, "tool call timeout"):
		issue.Code = "TIMEOUT_BLOCKED"
		issue.Retriable = true
		issue.RequiresAgent = true
		issue.AgentHints = []string{"当前步骤执行超时，请缩小任务范围后重试", "优先检查浏览器 relay 是否在线并附着标签页"}
	case strings.Contains(lower, "relay not attached") || strings.Contains(lower, "no browser tabs connected"):
		issue.Code = "RELAY_NOT_ATTACHED"
		issue.Retriable = true
		issue.RequiresAgent = true
		issue.AgentHints = []string{"请在目标页面点击 OpenClaw Browser Relay 工具栏按钮（ON）", "附着成功后再重试当前工具"}
	case strings.Contains(lower, "target page") || strings.Contains(lower, "has been closed"):
		issue.Code = "BROWSER_TARGET_CLOSED"
		issue.Retriable = true
		issue.RequiresAgent = true
		issue.AgentHints = []string{"重新聚焦 Chrome Relay 已附着标签页后重试", "必要时由上层 Agent 先调用 check_login_status"}
	case strings.Contains(lower, "comment input not found"):
		issue.Code = "COMMENT_INPUT_NOT_FOUND"
		issue.RequiresAgent = true
		issue.AgentHints = []string{"先确保评论面板已展开，再重试 submit_comment", "由 Agent 切换到手动选择器路径并回传新的 selector"}
	case strings.Contains(lower, "comment submit button not found"):
		issue.Code = "COMMENT_SUBMIT_NOT_FOUND"
		issue.RequiresAgent = true
		issue.AgentHints = []string{"由 Agent 检查页面是否触发风控/校验弹窗后再提交"}
	case strings.Contains(lower, "login timeout") || strings.Contains(lower, "not logged in"):
		issue.Code = "LOGIN_REQUIRED"
		issue.RequiresAgent = true
		issue.AgentHints = []string{"先执行 start_login 并等待人工完成验证码，再重试"}
	case strings.Contains(lower, "rate limit"):
		issue.Code = "RATE_LIMIT_BLOCKED"
		issue.Retriable = true
	}

	if issue.RequiresAgent {
		issue.Artifacts = map[string]any{
			"raw_error": msg,
		}
	}
	return issue
}
