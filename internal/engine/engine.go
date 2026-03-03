package engine

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"dy-ks-mcp/internal/config"
	"dy-ks-mcp/internal/platform"
	"dy-ks-mcp/internal/store"
)

type RunRequest struct {
	Platform  string `json:"platform"`
	AccountID string `json:"account_id"`
	Keyword   string `json:"keyword"`
	SortBy    string `json:"sort_by"`
	TimeRange string `json:"time_range"`
	Limit     int    `json:"limit"`
}

type RunResult struct {
	Platform       string   `json:"platform"`
	AccountID      string   `json:"account_id"`
	Keyword        string   `json:"keyword"`
	SortBy         string   `json:"sort_by"`
	TimeRange      string   `json:"time_range"`
	Searched       int      `json:"searched"`
	FilteredOut    int      `json:"filtered_out"`
	Attempted      int      `json:"attempted"`
	Commented      int      `json:"commented"`
	Duplicates     int      `json:"duplicates"`
	SkippedWindow  int      `json:"skipped_window"`
	SkippedBreaker int      `json:"skipped_breaker"`
	Failures       int      `json:"failures"`
	Errors         []string `json:"errors,omitempty"`
}

type Runner struct {
	cfg        config.EngineConfig
	store      store.Repository
	loc        *time.Location
	startMin   int
	endMin     int
	ratePeriod time.Duration
	openFor    time.Duration
	minPause   time.Duration
	maxPause   time.Duration
	blocked    []string

	mu  sync.Mutex
	rng *rand.Rand
}

func NewRunner(cfg config.EngineConfig, repo store.Repository) (*Runner, error) {
	ratePeriod, err := cfg.RateLimit.PerDuration()
	if err != nil {
		return nil, fmt.Errorf("parse rate limit period: %w", err)
	}
	openFor, err := cfg.CircuitBreaker.OpenDuration()
	if err != nil {
		return nil, fmt.Errorf("parse circuit breaker duration: %w", err)
	}
	startMin, endMin, err := cfg.TimeWindow.Parsed()
	if err != nil {
		return nil, fmt.Errorf("parse time window: %w", err)
	}
	loc, err := cfg.TimeWindow.Location()
	if err != nil {
		return nil, fmt.Errorf("load time zone: %w", err)
	}

	blocked := make([]string, 0, len(cfg.Filter.BlockedKeywords))
	for _, b := range cfg.Filter.BlockedKeywords {
		v := strings.ToLower(strings.TrimSpace(b))
		if v != "" {
			blocked = append(blocked, v)
		}
	}

	return &Runner{
		cfg:        cfg,
		store:      repo,
		loc:        loc,
		startMin:   startMin,
		endMin:     endMin,
		ratePeriod: ratePeriod,
		openFor:    openFor,
		minPause:   cfg.RandomPause.MinDuration(),
		maxPause:   cfg.RandomPause.MaxDuration(),
		blocked:    blocked,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (r *Runner) Run(ctx context.Context, client platform.Client, req RunRequest) (RunResult, error) {
	if req.Keyword == "" {
		return RunResult{}, fmt.Errorf("keyword is required")
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.AccountID == "" {
		req.AccountID = "default"
	}

	req.SortBy = platform.NormalizeSortBy(req.SortBy)
	req.TimeRange = platform.NormalizeTimeRange(req.TimeRange)

	result := RunResult{
		Platform:  client.Name(),
		AccountID: req.AccountID,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		TimeRange: req.TimeRange,
	}

	posts, err := client.Search(ctx, req.AccountID, platform.SearchQuery{
		Keyword:   req.Keyword,
		Limit:     req.Limit,
		SortBy:    req.SortBy,
		TimeRange: req.TimeRange,
	})
	if err != nil {
		return result, fmt.Errorf("search posts: %w", err)
	}

	result.Searched = len(posts)
	candidates := r.filterPosts(posts, &result)
	limiter := r.newLimiter()
	breaker := &breakerState{}

	for _, post := range candidates {
		duplicate, err := r.store.HasCommented(ctx, client.Name(), req.AccountID, post.ID)
		if err != nil {
			result.Failures++
			result.addError(fmt.Sprintf("dedupe check failed for %s: %v", post.ID, err))
			continue
		}
		if duplicate {
			result.Duplicates++
			continue
		}

		now := time.Now().In(r.loc)
		if !inWindow(now, r.startMin, r.endMin) {
			result.SkippedWindow++
			continue
		}
		if !breaker.Allow(now) {
			result.SkippedBreaker++
			continue
		}

		if err := limiter.Wait(ctx); err != nil {
			return result, fmt.Errorf("rate limit wait failed: %w", err)
		}
		if err := r.randomPause(ctx); err != nil {
			return result, err
		}

		comment := r.renderComment(req.Keyword, post)
		result.Attempted++
		err = client.Comment(ctx, req.AccountID, platform.CommentRequest{
			PostID:  post.ID,
			Content: comment,
		})
		if err != nil {
			result.Failures++
			result.addError(fmt.Sprintf("comment failed for %s: %v", post.ID, err))
			breaker.RecordFailure(now, r.cfg.CircuitBreaker.FailureThreshold, r.openFor)
			continue
		}

		err = r.store.SaveComment(ctx, store.CommentRecord{
			Platform:  client.Name(),
			AccountID: req.AccountID,
			PostID:    post.ID,
			Keyword:   req.Keyword,
			Comment:   comment,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			if errors.Is(err, store.ErrDuplicate) {
				result.Duplicates++
				continue
			}
			result.Failures++
			result.addError(fmt.Sprintf("save failed for %s: %v", post.ID, err))
			breaker.RecordFailure(now, r.cfg.CircuitBreaker.FailureThreshold, r.openFor)
			continue
		}

		breaker.RecordSuccess()
		result.Commented++
	}

	return result, nil
}

func (r *Runner) filterPosts(posts []platform.Post, result *RunResult) []platform.Post {
	out := make([]platform.Post, 0, len(posts))
	for _, post := range posts {
		if strings.TrimSpace(post.ID) == "" {
			result.FilteredOut++
			continue
		}
		title := strings.ToLower(post.Title)
		blocked := false
		for _, kw := range r.blocked {
			if strings.Contains(title, kw) {
				blocked = true
				break
			}
		}
		if blocked {
			result.FilteredOut++
			continue
		}
		out = append(out, post)
	}
	return out
}

func (r *Runner) newLimiter() *rate.Limiter {
	ratePerSec := float64(r.cfg.RateLimit.Actions) / r.ratePeriod.Seconds()
	if ratePerSec <= 0 {
		return rate.NewLimiter(rate.Inf, 1)
	}
	burst := r.cfg.RateLimit.Actions
	if burst < 1 {
		burst = 1
	}
	return rate.NewLimiter(rate.Limit(ratePerSec), burst)
}

func (r *Runner) randomPause(ctx context.Context) error {
	if r.maxPause <= 0 {
		return nil
	}
	pause := r.minPause
	if r.maxPause > r.minPause {
		delta := r.maxPause - r.minPause
		pause += time.Duration(r.randInt63n(int64(delta + 1)))
	}
	timer := time.NewTimer(pause)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *Runner) randInt63n(n int64) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Int63n(n)
}

func (r *Runner) renderComment(keyword string, post platform.Post) string {
	t := r.cfg.CommentTemplates[0]
	if len(r.cfg.CommentTemplates) > 1 {
		i := int(r.randInt63n(int64(len(r.cfg.CommentTemplates))))
		t = r.cfg.CommentTemplates[i]
	}
	t = strings.ReplaceAll(t, "{keyword}", keyword)
	t = strings.ReplaceAll(t, "{title}", post.Title)
	return t
}

func inWindow(now time.Time, startMin int, endMin int) bool {
	current := now.Hour()*60 + now.Minute()
	if startMin <= endMin {
		return current >= startMin && current <= endMin
	}
	return current >= startMin || current <= endMin
}

type breakerState struct {
	failures  int
	openUntil time.Time
}

func (b *breakerState) Allow(now time.Time) bool {
	return now.After(b.openUntil) || now.Equal(b.openUntil)
}

func (b *breakerState) RecordFailure(now time.Time, threshold int, openFor time.Duration) {
	b.failures++
	if b.failures >= threshold {
		b.openUntil = now.Add(openFor)
		b.failures = 0
	}
}

func (b *breakerState) RecordSuccess() {
	b.failures = 0
}

func (r *RunResult) addError(msg string) {
	if len(r.Errors) >= 20 {
		return
	}
	r.Errors = append(r.Errors, msg)
}
