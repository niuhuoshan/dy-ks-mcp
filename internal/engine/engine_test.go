package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"dy-ks-mcp/internal/config"
	"dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/store"
)

type notImplementedSearchClient struct{}

func (c notImplementedSearchClient) Name() string {
	return "douyin"
}

func (c notImplementedSearchClient) Login(ctx context.Context, accountID string) error {
	return nil
}

func (c notImplementedSearchClient) CheckLogin(ctx context.Context, accountID string) (platform.LoginStatus, error) {
	return platform.LoginStatus{}, nil
}

func (c notImplementedSearchClient) Search(ctx context.Context, accountID string, query platform.SearchQuery) ([]platform.Post, error) {
	return nil, platform.NewNotImplementedError(c.Name(), "Search")
}

func (c notImplementedSearchClient) Comment(ctx context.Context, accountID string, req platform.CommentRequest) error {
	return nil
}

type noopStore struct{}

func (s noopStore) Init(ctx context.Context) error {
	return nil
}

func (s noopStore) HasCommented(ctx context.Context, platform string, accountID string, postID string) (bool, error) {
	return false, nil
}

func (s noopStore) SaveComment(ctx context.Context, record store.CommentRecord) error {
	return nil
}

func (s noopStore) Close() error {
	return nil
}

func TestRunHandlesNotImplementedSearchAsResultError(t *testing.T) {
	runner, err := NewRunner(config.EngineConfig{
		RateLimit: config.RateLimitConfig{
			Actions: 1,
			Per:     "1s",
		},
		RandomPause: config.RandomPauseConfig{
			MinMS: 0,
			MaxMS: 0,
		},
		TimeWindow: config.TimeWindowConfig{
			Timezone: "UTC",
			Start:    "00:00",
			End:      "23:59",
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			FailureThreshold: 1,
			OpenFor:          "1s",
		},
		CommentTemplates: []string{"ok"},
	}, noopStore{})
	if err != nil {
		t.Fatalf("NewRunner error = %v", err)
	}

	result, runErr := runner.Run(context.Background(), notImplementedSearchClient{}, RunRequest{
		Platform:  "douyin",
		AccountID: "acct",
		Keyword:   "automation",
		Limit:     5,
	})
	if runErr != nil {
		t.Fatalf("Run error = %v", runErr)
	}
	if result.Failures != 1 {
		t.Fatalf("Failures = %d, want 1", result.Failures)
	}
	if result.Searched != 0 {
		t.Fatalf("Searched = %d, want 0", result.Searched)
	}
	if result.Attempted != 0 {
		t.Fatalf("Attempted = %d, want 0", result.Attempted)
	}
	if result.Commented != 0 {
		t.Fatalf("Commented = %d, want 0", result.Commented)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors length = %d, want 1", len(result.Errors))
	}
	if !strings.Contains(result.Errors[0], "douyin.Search: not implemented") {
		t.Fatalf("Errors[0] = %q, want method/platform context", result.Errors[0])
	}
}

func TestInWindowAcrossMidnight(t *testing.T) {
	now := time.Date(2026, 3, 3, 1, 30, 0, 0, time.UTC)
	if !inWindow(now, 23*60, 2*60) {
		t.Fatal("expected true for wrapped time window")
	}
}
