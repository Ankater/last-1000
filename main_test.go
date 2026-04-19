package main

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ankater/last-1000/internal/messages"
	"github.com/Ankater/last-1000/internal/testsql"
	"github.com/Ankater/last-1000/internal/tokens"
)

func TestRequireBearerAuth(t *testing.T) {
	t.Run("rejects missing header", func(t *testing.T) {
		s := &server{}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("rejects invalid prefix", func(t *testing.T) {
		s := &server{}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		req.Header.Set("Authorization", "Token abc")
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("rejects empty token", func(t *testing.T) {
		s := &server{}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		req.Header.Set("Authorization", "Bearer ")
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("rejects unknown token", func(t *testing.T) {
		s := &server{
			tokens: newTokenRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					return testsql.Rows([]string{"exists"}, []any{false}), nil
				},
			}),
		}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("returns internal error when token lookup fails", func(t *testing.T) {
		s := &server{
			tokens: newTokenRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					return nil, errors.New("db down")
				},
			}),
		}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
		if nextCalled {
			t.Fatal("expected next handler not to be called")
		}
	})

	t.Run("passes through for valid token", func(t *testing.T) {
		s := &server{
			tokens: newTokenRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					if len(args) != 1 {
						t.Fatalf("expected one query arg, got %d", len(args))
					}
					return testsql.Rows([]string{"exists"}, []any{true}), nil
				},
			}),
		}
		nextCalled := false

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()

		s.requireBearerAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusNoContent)
		})).ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
		if !nextCalled {
			t.Fatal("expected next handler to be called")
		}
	})
}

func TestAddMessage(t *testing.T) {
	t.Run("returns bad request for invalid json", func(t *testing.T) {
		s := &server{}

		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader("{"))
		rec := httptest.NewRecorder()

		s.addMessage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("returns bad request for blank message", func(t *testing.T) {
		s := &server{}

		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(`{"message":"   "}`))
		rec := httptest.NewRecorder()

		s.addMessage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("creates message", func(t *testing.T) {
		var gotArgs []any
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Exec: func(ctx context.Context, query string, args []any) (driver.Result, error) {
					gotArgs = args
					return driver.RowsAffected(1), nil
				},
			}),
		}

		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(`{"message":"hello"}`))
		rec := httptest.NewRecorder()

		s.addMessage(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}
		if len(gotArgs) != 1 || gotArgs[0] != "hello" {
			t.Fatalf("expected message arg %q, got %#v", "hello", gotArgs)
		}
	})

	t.Run("returns internal error on repository failure", func(t *testing.T) {
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Exec: func(ctx context.Context, query string, args []any) (driver.Result, error) {
					return nil, errors.New("insert failed")
				},
			}),
		}

		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(`{"message":"hello"}`))
		rec := httptest.NewRecorder()

		s.addMessage(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestGetMessages(t *testing.T) {
	type response struct {
		Messages []messages.Message `json:"messages"`
		Page     int                `json:"page"`
	}

	t.Run("uses default page and returns empty array", func(t *testing.T) {
		var gotArgs []any
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					gotArgs = args
					return testsql.Rows([]string{"id", "message", "created_at"}), nil
				},
			}),
		}

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		rec := httptest.NewRecorder()

		s.getMessages(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", rec.Header().Get("Content-Type"))
		}
		if len(gotArgs) != 2 || !isIntArg(gotArgs[0], 100) || !isIntArg(gotArgs[1], 0) {
			t.Fatalf("expected limit/offset [100 0], got %#v", gotArgs)
		}

		var body response
		if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if body.Page != 1 {
			t.Fatalf("expected page 1, got %d", body.Page)
		}
		if len(body.Messages) != 0 {
			t.Fatalf("expected empty messages, got %#v", body.Messages)
		}
		if body.Messages == nil {
			t.Fatal("expected messages to be encoded as an empty array")
		}
	})

	t.Run("falls back to page one for invalid query value", func(t *testing.T) {
		var gotArgs []any
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					gotArgs = args
					return testsql.Rows([]string{"id", "message", "created_at"}), nil
				},
			}),
		}

		req := httptest.NewRequest(http.MethodGet, "/messages?page=abc", nil)
		rec := httptest.NewRecorder()

		s.getMessages(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if len(gotArgs) != 2 || !isIntArg(gotArgs[1], 0) {
			t.Fatalf("expected offset 0 for invalid page, got %#v", gotArgs)
		}
	})

	t.Run("uses requested page and returns messages", func(t *testing.T) {
		now := time.Now().UTC().Round(0)
		var gotArgs []any
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					gotArgs = args
					return testsql.Rows(
						[]string{"id", "message", "created_at"},
						[]any{int64(2), "newest", now},
						[]any{int64(1), "older", now.Add(-time.Minute)},
					), nil
				},
			}),
		}

		req := httptest.NewRequest(http.MethodGet, "/messages?page=2", nil)
		rec := httptest.NewRecorder()

		s.getMessages(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if len(gotArgs) != 2 || !isIntArg(gotArgs[0], 100) || !isIntArg(gotArgs[1], 100) {
			t.Fatalf("expected limit/offset [100 100], got %#v", gotArgs)
		}

		var body response
		if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if body.Page != 2 {
			t.Fatalf("expected page 2, got %d", body.Page)
		}
		if len(body.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(body.Messages))
		}
		if body.Messages[0].Message != "newest" || body.Messages[1].Message != "older" {
			t.Fatalf("unexpected messages order: %#v", body.Messages)
		}
	})

	t.Run("returns internal error on repository failure", func(t *testing.T) {
		s := &server{
			messages: newMessageRepository(t, testsql.Config{
				Query: func(ctx context.Context, query string, args []any) (driver.Rows, error) {
					return nil, errors.New("query failed")
				},
			}),
		}

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		rec := httptest.NewRecorder()

		s.getMessages(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func newTokenRepository(t *testing.T, cfg testsql.Config) *tokens.Repository {
	t.Helper()

	db, err := testsql.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return &tokens.Repository{DB: db}
}

func newMessageRepository(t *testing.T, cfg testsql.Config) *messages.Repository {
	t.Helper()

	db, err := testsql.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	return &messages.Repository{DB: db}
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
