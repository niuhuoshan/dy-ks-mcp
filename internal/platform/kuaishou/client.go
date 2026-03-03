package kuaishou

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	base "dy-ks-mcp/internal/platform"
)

type Client struct {
	selectors base.SelectorSet
	mu        sync.RWMutex
	logins    map[string]bool
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
		logins:    map[string]bool{},
	}, nil
}

func (c *Client) Name() string {
	return "kuaishou"
}

func (c *Client) Login(ctx context.Context, accountID string) error {
	_ = ctx
	if accountID == "" {
		return fmt.Errorf("account_id is required")
	}
	c.mu.Lock()
	c.logins[accountID] = true
	c.mu.Unlock()
	return nil
}

func (c *Client) CheckLogin(ctx context.Context, accountID string) (base.LoginStatus, error) {
	_ = ctx
	if accountID == "" {
		accountID = "default"
	}
	loggedIn := c.isLoggedIn(accountID)
	message := "placeholder: ready for go-rod login workflow"
	if loggedIn {
		message = "logged in (placeholder state)"
	}
	return base.LoginStatus{
		Platform:  c.Name(),
		AccountID: accountID,
		LoggedIn:  loggedIn,
		Message:   message,
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (c *Client) Search(ctx context.Context, accountID string, query base.SearchQuery) ([]base.Post, error) {
	_ = ctx
	if !c.isLoggedIn(defaultAccount(accountID)) {
		return nil, fmt.Errorf("account %q is not logged in", defaultAccount(accountID))
	}
	if query.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}
	if query.Limit <= 0 {
		query.Limit = 10
	}
	query.SortBy = base.NormalizeSortBy(query.SortBy)
	query.TimeRange = base.NormalizeTimeRange(query.TimeRange)

	out := make([]base.Post, 0, query.Limit)
	keyword := sanitizeKeyword(query.Keyword)
	for i := 1; i <= query.Limit; i++ {
		id := fmt.Sprintf("ks-%s-%s-%s-%d", keyword, query.SortBy, query.TimeRange, i)
		out = append(out, base.Post{
			ID:       id,
			Title:    fmt.Sprintf("Kuaishou placeholder result %d for %s (sort=%s time=%s)", i, query.Keyword, query.SortBy, query.TimeRange),
			URL:      fmt.Sprintf("https://www.kuaishou.com/short-video/%s", id),
			AuthorID: fmt.Sprintf("ks-author-%d", i),
		})
	}
	return out, nil
}

func (c *Client) Comment(ctx context.Context, accountID string, req base.CommentRequest) error {
	_ = ctx
	if !c.isLoggedIn(defaultAccount(accountID)) {
		return fmt.Errorf("account %q is not logged in", defaultAccount(accountID))
	}
	if req.PostID == "" {
		return fmt.Errorf("post_id is required")
	}
	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("comment content is empty")
	}
	return nil
}

func (c *Client) isLoggedIn(accountID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logins[defaultAccount(accountID)]
}

func defaultAccount(accountID string) string {
	if accountID == "" {
		return "default"
	}
	return accountID
}

func sanitizeKeyword(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	if s == "" {
		return "keyword"
	}
	return s
}
