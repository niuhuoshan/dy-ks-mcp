package store

import (
	"context"
	"errors"
	"time"
)

var ErrDuplicate = errors.New("duplicate comment")

type CommentRecord struct {
	Platform  string
	AccountID string
	PostID    string
	Keyword   string
	Comment   string
	CreatedAt time.Time
}

type Repository interface {
	Init(ctx context.Context) error
	HasCommented(ctx context.Context, platform string, accountID string, postID string) (bool, error)
	SaveComment(ctx context.Context, record CommentRecord) error
	Close() error
}
