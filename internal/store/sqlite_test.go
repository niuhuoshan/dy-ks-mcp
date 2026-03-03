package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreSaveAndDedupe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comments.db")

	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore error = %v", err)
	}
	defer func() {
		_ = s.Close()
	}()

	ctx := context.Background()
	if err := s.Init(ctx); err != nil {
		t.Fatalf("Init error = %v", err)
	}

	has, err := s.HasCommented(ctx, "douyin", "acct", "post-1")
	if err != nil {
		t.Fatalf("HasCommented initial error = %v", err)
	}
	if has {
		t.Fatal("HasCommented initial = true, want false")
	}

	err = s.SaveComment(ctx, CommentRecord{
		Platform:  "douyin",
		AccountID: "acct",
		PostID:    "post-1",
		Keyword:   "automation",
		Comment:   "hello",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("SaveComment first error = %v", err)
	}

	has, err = s.HasCommented(ctx, "douyin", "acct", "post-1")
	if err != nil {
		t.Fatalf("HasCommented after insert error = %v", err)
	}
	if !has {
		t.Fatal("HasCommented after insert = false, want true")
	}

	err = s.SaveComment(ctx, CommentRecord{
		Platform:  "douyin",
		AccountID: "acct",
		PostID:    "post-1",
		Keyword:   "automation",
		Comment:   "hello again",
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("SaveComment duplicate error = %v, want ErrDuplicate", err)
	}
}
