package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS comments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	platform TEXT NOT NULL,
	account_id TEXT NOT NULL,
	post_id TEXT NOT NULL,
	keyword TEXT NOT NULL,
	comment TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(platform, account_id, post_id)
);
CREATE INDEX IF NOT EXISTS idx_comments_platform_account ON comments(platform, account_id);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) HasCommented(ctx context.Context, platform string, accountID string, postID string) (bool, error) {
	const q = `SELECT 1 FROM comments WHERE platform = ? AND account_id = ? AND post_id = ? LIMIT 1`
	var hit int
	err := s.db.QueryRowContext(ctx, q, platform, accountID, postID).Scan(&hit)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check duplicate: %w", err)
	}
	return true, nil
}

func (s *SQLiteStore) SaveComment(ctx context.Context, record CommentRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	const q = `
INSERT INTO comments (platform, account_id, post_id, keyword, comment, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`
	_, err := s.db.ExecContext(
		ctx,
		q,
		record.Platform,
		record.AccountID,
		record.PostID,
		record.Keyword,
		record.Comment,
		record.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrDuplicate
		}
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
