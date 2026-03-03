package kuaishou

import (
	"context"
	"fmt"
	"strings"

	base "dy-ks-mcp/internal/platform"
)

type Client struct {
	selectors base.SelectorSet
}

func NewClient(selectorPath string) (*Client, error) {
	selectors, err := base.LoadSelectorFile(selectorPath)
	if err != nil {
		return nil, fmt.Errorf("load kuaishou selectors: %w", err)
	}
	if selectors.Platform == "" {
		selectors.Platform = "kuaishou"
	}
	return &Client{
		selectors: selectors,
	}, nil
}

func (c *Client) Name() string {
	return "kuaishou"
}

func (c *Client) Login(ctx context.Context, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return fmt.Errorf("account_id is required")
	}
	return c.loginWithBrowser(ctx, accountID)
}

func (c *Client) CheckLogin(ctx context.Context, accountID string) (base.LoginStatus, error) {
	accountID = defaultAccount(accountID)
	return base.LoginStatus{}, c.checkLoginWithBrowser(ctx, accountID)
}

func (c *Client) Search(ctx context.Context, accountID string, query base.SearchQuery) ([]base.Post, error) {
	if strings.TrimSpace(query.Keyword) == "" {
		return nil, fmt.Errorf("keyword is required")
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	query.SortBy = base.NormalizeSortBy(query.SortBy)
	query.TimeRange = base.NormalizeTimeRange(query.TimeRange)
	return c.searchWithBrowser(ctx, defaultAccount(accountID), query)
}

func (c *Client) Comment(ctx context.Context, accountID string, req base.CommentRequest) error {
	if strings.TrimSpace(req.PostID) == "" {
		return fmt.Errorf("post_id is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("comment content is empty")
	}
	return c.commentWithBrowser(ctx, defaultAccount(accountID), req)
}

func (c *Client) loginWithBrowser(ctx context.Context, accountID string) error {
	_ = ctx
	_ = accountID
	_ = c.selectors
	return base.NewNotImplementedError(c.Name(), "Login")
}

func (c *Client) checkLoginWithBrowser(ctx context.Context, accountID string) error {
	_ = ctx
	_ = accountID
	_ = c.selectors
	return base.NewNotImplementedError(c.Name(), "CheckLogin")
}

func (c *Client) searchWithBrowser(ctx context.Context, accountID string, query base.SearchQuery) ([]base.Post, error) {
	_ = ctx
	_ = accountID
	_ = query
	_ = c.selectors
	return nil, base.NewNotImplementedError(c.Name(), "Search")
}

func (c *Client) commentWithBrowser(ctx context.Context, accountID string, req base.CommentRequest) error {
	_ = ctx
	_ = accountID
	_ = req
	_ = c.selectors
	return base.NewNotImplementedError(c.Name(), "Comment")
}

func defaultAccount(accountID string) string {
	if strings.TrimSpace(accountID) == "" {
		return "default"
	}
	return strings.TrimSpace(accountID)
}
