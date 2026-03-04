package douyin

import (
	"context"
	"fmt"
	"time"

	"dy-ks-mcp/internal/config"
	base "dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/platform/worker"
)

type Client struct {
	selectors map[string]string
	nodeBin   string
	script    string
	browser   worker.BrowserSettings
}

func NewClient(selectorPath string, browserCfg config.BrowserConfig) (*Client, error) {
	selectors, err := base.LoadSelectorFile(selectorPath)
	if err != nil {
		return nil, fmt.Errorf("load douyin selectors: %w", err)
	}

	actionTimeout, err := browserCfg.ActionTimeoutDuration()
	if err != nil {
		return nil, fmt.Errorf("parse browser action timeout: %w", err)
	}
	navTimeout, err := browserCfg.NavigationTimeoutDuration()
	if err != nil {
		return nil, fmt.Errorf("parse browser navigation timeout: %w", err)
	}
	loginTimeout, err := browserCfg.LoginTimeoutDuration()
	if err != nil {
		return nil, fmt.Errorf("parse browser login timeout: %w", err)
	}

	return &Client{
		selectors: selectors.Selectors,
		nodeBin:   browserCfg.NodeBinary,
		script:    browserCfg.ScriptPath,
		browser: worker.NewBrowserSettings(
			browserCfg.WSURL,
			browserCfg.Headless,
			browserCfg.ExecutablePath,
			browserCfg.UserDataRoot,
			actionTimeout,
			navTimeout,
			loginTimeout,
			browserCfg.PostLoadWaitDuration(),
		),
	}, nil
}

func (c *Client) Name() string {
	return "douyin"
}

func (c *Client) Login(ctx context.Context, accountID string) error {
	accountID = defaultAccount(accountID)
	_, err := c.run(ctx, worker.Request{
		Action:    "login",
		Platform:  c.Name(),
		AccountID: accountID,
	})
	return err
}

func (c *Client) CheckLogin(ctx context.Context, accountID string) (base.LoginStatus, error) {
	accountID = defaultAccount(accountID)
	resp, err := c.run(ctx, worker.Request{
		Action:    "check_login",
		Platform:  c.Name(),
		AccountID: accountID,
	})
	if err != nil {
		return base.LoginStatus{}, err
	}

	return base.LoginStatus{
		Platform:  c.Name(),
		AccountID: accountID,
		LoggedIn:  resp.LoggedIn,
		Message:   resp.Message,
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (c *Client) Search(ctx context.Context, accountID string, query base.SearchQuery) ([]base.Post, error) {
	if query.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	query.SortBy = base.NormalizeSortBy(query.SortBy)
	query.TimeRange = base.NormalizeTimeRange(query.TimeRange)

	resp, err := c.run(ctx, worker.Request{
		Action:    "search",
		Platform:  c.Name(),
		AccountID: defaultAccount(accountID),
		Keyword:   query.Keyword,
		SortBy:    query.SortBy,
		TimeRange: query.TimeRange,
		Limit:     query.Limit,
	})
	if err != nil {
		return nil, err
	}
	return resp.Posts, nil
}

func (c *Client) Comment(ctx context.Context, accountID string, req base.CommentRequest) error {
	if req.PostID == "" {
		return fmt.Errorf("post_id is required")
	}
	if req.Content == "" {
		return fmt.Errorf("comment content is empty")
	}
	_, err := c.run(ctx, worker.Request{
		Action:    "comment",
		Platform:  c.Name(),
		AccountID: defaultAccount(accountID),
		PostID:    req.PostID,
		Content:   req.Content,
	})
	return err
}

func (c *Client) run(ctx context.Context, req worker.Request) (worker.Response, error) {
	req.Selectors = c.selectors
	req.Browser = c.browser
	return worker.Run(ctx, c.nodeBin, c.script, req)
}

func defaultAccount(accountID string) string {
	if accountID == "" {
		return "default"
	}
	return accountID
}
