package messages

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ankater/last-1000/internal/testsql"
)

func TestAddMessage(t *testing.T) {
	var gotQuery string
	var gotArgs []any

	db, err := testsql.Open(testsql.Config{
		Exec: func(ctx context.Context, query string, args []any) (driver.Result, error) {
			gotQuery = query
			gotArgs = args
			return driver.RowsAffected(1), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	if err := repo.AddMessage(context.Background(), "hello"); err != nil {
		t.Fatalf("AddMessage returned error: %v", err)
	}

	if gotQuery != `INSERT INTO messages (message) VALUES ($1)` {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "hello" {
		t.Fatalf("unexpected args: %#v", gotArgs)
	}
}

func TestAddMessagePropagatesExecError(t *testing.T) {
	db, err := testsql.Open(testsql.Config{
		Exec: func(ctx context.Context, query string, args []any) (driver.Result, error) {
			return nil, errors.New("insert failed")
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	err = repo.AddMessage(context.Background(), "hello")
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("expected insert error, got %v", err)
	}
}

func TestGetMessagesNormalizesInvalidPage(t *testing.T) {
	var gotArgs []any

	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			gotArgs = args
			return testsql.Rows([]string{"id", "message", "created_at"}), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	msgs, err := repo.GetMessages(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}
	if msgs != nil {
		t.Fatalf("expected nil slice for empty result, got %#v", msgs)
	}
	if len(gotArgs) != 2 || !isIntArg(gotArgs[0], pageSize) || !isIntArg(gotArgs[1], 0) {
		t.Fatalf("expected limit/offset [%d 0], got %#v", pageSize, gotArgs)
	}
}

func TestGetMessagesReturnsRowsInQueryOrder(t *testing.T) {
	now := time.Now().UTC().Round(0)
	var gotArgs []any

	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			gotArgs = args
			if !strings.Contains(query, "ORDER BY created_at DESC, id DESC") {
				t.Fatalf("expected descending order query, got %q", query)
			}
			return testsql.Rows(
				[]string{"id", "message", "created_at"},
				[]any{int64(2), "newest", now},
				[]any{int64(1), "older", now.Add(-time.Minute)},
			), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	msgs, err := repo.GetMessages(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetMessages returned error: %v", err)
	}

	if len(gotArgs) != 2 || !isIntArg(gotArgs[0], pageSize) || !isIntArg(gotArgs[1], pageSize) {
		t.Fatalf("expected limit/offset [%d %d], got %#v", pageSize, pageSize, gotArgs)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].ID != 2 || msgs[0].Message != "newest" {
		t.Fatalf("unexpected first message: %#v", msgs[0])
	}
	if msgs[1].ID != 1 || msgs[1].Message != "older" {
		t.Fatalf("unexpected second message: %#v", msgs[1])
	}
}

func TestGetMessagesPropagatesQueryError(t *testing.T) {
	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			return nil, errors.New("query failed")
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	_, err = repo.GetMessages(context.Background(), 1)
	if err == nil || err.Error() != "query failed" {
		t.Fatalf("expected query error, got %v", err)
	}
}

func TestGetMessagesPropagatesScanError(t *testing.T) {
	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			return testsql.Rows(
				[]string{"id", "message", "created_at"},
				[]any{"bad-id", "hello", time.Now().UTC()},
			), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	_, err = repo.GetMessages(context.Background(), 1)
	if err == nil {
		t.Fatal("expected scan error, got nil")
	}
}

func isIntArg(value any, want int) bool {
	switch got := value.(type) {
	case int:
		return got == want
	case int64:
		return got == int64(want)
	default:
		return false
	}
}
