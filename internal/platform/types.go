package platform

import (
	"context"
	"strings"
	"time"
)

const (
	SortComprehensive = "comprehensive"
	SortLatest        = "latest"
)

const (
	TimeAll   = "all"
	TimeDay   = "day"
	TimeWeek  = "week"
	TimeMonth = "month"
	TimeYear  = "year"
)

type SearchQuery struct {
	Keyword   string
	Limit     int
	SortBy    string
	TimeRange string
}

type Post struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	AuthorID string `json:"author_id"`
}

type CommentRequest struct {
	PostID  string
	Content string
}

type LoginStatus struct {
	Platform  string    `json:"platform"`
	AccountID string    `json:"account_id"`
	LoggedIn  bool      `json:"logged_in"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}

type Client interface {
	Name() string
	Login(ctx context.Context, accountID string) error
	CheckLogin(ctx context.Context, accountID string) (LoginStatus, error)
	Search(ctx context.Context, accountID string, query SearchQuery) ([]Post, error)
	Comment(ctx context.Context, accountID string, req CommentRequest) error
}

func NormalizeSortBy(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", SortComprehensive:
		return SortComprehensive
	case SortLatest:
		return SortLatest
	default:
		return SortComprehensive
	}
}

func NormalizeTimeRange(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", TimeAll:
		return TimeAll
	case TimeDay:
		return TimeDay
	case TimeWeek:
		return TimeWeek
	case TimeMonth:
		return TimeMonth
	case TimeYear:
		return TimeYear
	default:
		return TimeAll
	}
}
