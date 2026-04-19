package tokens

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/Ankater/last-1000/internal/testsql"
)

func TestHashToken(t *testing.T) {
	sum := sha256.Sum256([]byte("secret"))
	want := hex.EncodeToString(sum[:])

	if got := hashToken("secret"); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCreateTokenStoresHash(t *testing.T) {
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
	if err := repo.CreateToken(context.Background(), "plain-token", "admin"); err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	if gotQuery != `INSERT INTO tokens (token_hash, name) VALUES ($1, $2)` {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if len(gotArgs) != 2 {
		t.Fatalf("expected two args, got %#v", gotArgs)
	}
	if gotArgs[0] != hashToken("plain-token") {
		t.Fatalf("expected hashed token, got %#v", gotArgs[0])
	}
	if gotArgs[1] != "admin" {
		t.Fatalf("expected token name %q, got %#v", "admin", gotArgs[1])
	}
}

func TestCreateTokenPropagatesExecError(t *testing.T) {
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
	err = repo.CreateToken(context.Background(), "plain-token", "admin")
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("expected insert error, got %v", err)
	}
}

func TestTokenExists(t *testing.T) {
	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			if len(args) != 1 {
				t.Fatalf("expected one arg, got %#v", args)
			}
			if args[0] != hashToken("plain-token") {
				t.Fatalf("expected hashed token arg, got %#v", args[0])
			}
			return testsql.Rows([]string{"exists"}, []any{true}), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	exists, err := repo.TokenExists(context.Background(), "plain-token")
	if err != nil {
		t.Fatalf("TokenExists returned error: %v", err)
	}
	if !exists {
		t.Fatal("expected token to exist")
	}
}

func TestTokenExistsReturnsFalse(t *testing.T) {
	db, err := testsql.Open(testsql.Config{
		Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
			return testsql.Rows([]string{"exists"}, []any{false}), nil
		},
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repo := Repository{DB: db}
	exists, err := repo.TokenExists(context.Background(), "plain-token")
	if err != nil {
		t.Fatalf("TokenExists returned error: %v", err)
	}
	if exists {
		t.Fatal("expected token not to exist")
	}
}

func TestTokenExistsPropagatesQueryError(t *testing.T) {
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
	_, err = repo.TokenExists(context.Background(), "plain-token")
	if err == nil || err.Error() != "query failed" {
		t.Fatalf("expected query error, got %v", err)
	}
}
