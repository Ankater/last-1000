# CLAUDE.md

## Project Overview

This repository is a small Go HTTP API backed by PostgreSQL.

It currently provides:

- `GET /` health-style endpoint returning `hello`
- `POST /messages` to store a message
- `GET /messages?page=N` to fetch messages in reverse chronological order
- `cmd/admin` CLI command to create bearer tokens

The main application entrypoint is [main.go](./main.go).

## Architecture

The codebase is intentionally small and split by responsibility:

- [main.go](./main.go): HTTP server, routing, auth middleware, request/response handling
- [internal/db/open.go](./internal/db/open.go): PostgreSQL connection setup from environment
- [internal/messages/repository.go](./internal/messages/repository.go): message persistence and pagination
- [internal/tokens/repository.go](./internal/tokens/repository.go): token storage and lookup
- [internal/tokens/generator.go](./internal/tokens/generator.go): random token generation
- [cmd/admin/main.go](./cmd/admin/main.go): admin CLI for token creation

Database initialization lives in:

- [docker/postgres/init/001-add-tokens.sql](./docker/postgres/init/001-add-tokens.sql)
- [docker/postgres/init/002-add-messages.sql](./docker/postgres/init/002-add-messages.sql)

## Auth Model

- API auth is `Authorization: Bearer <token>`.
- Tokens are stored as SHA-256 hashes, not plain text.
- `TokenExists` currently checks only for hash existence.
- `expires_at` and `revoked_at` exist in the schema but are not enforced by the current middleware or repository query. If auth behavior changes, review this first.

## Message Behavior

- Messages are stored as raw text in PostgreSQL.
- Pagination uses a fixed page size of `100`.
- Results are ordered by `created_at DESC, id DESC`.
- Invalid or missing `page` values fall back to page `1`.

## Local Development

This project expects a `.env` file and a local PostgreSQL instance, usually via Docker Compose.

Start Postgres:

```bash
docker compose up -d
```

Run the API:

```bash
go run .
```

Create an API token:

```bash
go run ./cmd/admin create-token --name dev
```

The app listens on `:8080`.

## Environment

The application loads environment variables from `.env`.

`internal/db.Open()` supports either:

- `DATABASE_URL`
- or the individual Postgres variables:
  - `POSTGRES_HOST` default `localhost`
  - `POSTGRES_PORT` default `5432`
  - `POSTGRES_USER`
  - `POSTGRES_PASSWORD`
  - `POSTGRES_DB`
  - `POSTGRES_SSLMODE` default `disable`

`docker-compose.yml` also depends on:

- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_PORT`

## Testing

Run the full suite from the repo root:

```bash
GOCACHE=/tmp/last-1000-go-build-cache go test ./...
```

Useful variants:

```bash
go test ./internal/messages
go test ./internal/tokens
go test -run TestGetMessages ./...
go test -v ./...
```

Notes:

- Tests are written with the standard `testing` package.
- Repository and handler tests avoid a real database by using the local fake SQL driver in [internal/testsql/testsql.go](./internal/testsql/testsql.go).
- No external services are required to run the current test suite.

## Change Guidelines

When modifying this project:

- Keep HTTP handlers thin and push database behavior into repositories.
- Preserve the current JSON API shape unless intentionally changing the contract.
- Add or update tests alongside behavior changes.
- If auth logic changes, cover missing token, malformed header, DB failure, and valid-token cases.
- If pagination changes, cover default page handling, invalid page handling, ordering, and empty results.

## Current Gaps Worth Knowing

- There is no service layer; handlers talk directly to repositories.
- Auth does not yet honor token revocation or expiry despite schema support.
- There is no migration tool; schema is initialized from Docker entrypoint SQL files.