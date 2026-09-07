# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

Tooling is managed with [mise](https://mise.jdx.dev/). Key tools: Go, buf, protoc-gen-go, protoc-gen-connect-go, cedar-policy-cli, pocketbase, golangci-lint, goimports.

```bash
# Build
mise run go:build          # outputs to build/cli

# Run (starts PocketBase + ConnectRPC server)
mise run go:run             # serves at localhost:8090

# Tests
go test ./...               # all tests
go test ./internal/authz/   # just authz tests
go test ./internal/store/   # store tests (memory + PocketBase)
go test -run TestCedarAuthorization/author_can_get_own_draft ./internal/authz/  # single case

# Protobuf
mise run proto:generate     # regenerate gen/ from proto/
mise run proto:check        # lint protos

# Cedar policies
mise run cedar:check        # format-check + parse + validate + test
mise run cedar:fix          # auto-format policies

# Go formatting & linting
mise run go:fix             # fmt + goimports + tidy
mise run go:lint            # golangci-lint (requires proto:generate first)
```

## Architecture

A ConnectRPC library service backed by PocketBase (SQLite) with Cedar authorization.

**Request flow:** ConnectRPC request → `authz.Interceptor` (extracts Bearer token → PocketBase user ID → context) → `library.Server` handler → handler calls `authz.Authorize` (Cedar policy eval) → `store.Store` for persistence.

**Authorization:** Two-phase per request. The interceptor handles authentication (token → user ID). Each handler then constructs Cedar entities with the relevant attributes and calls `authz.Authorize` for the policy decision. Cedar policies live in `policies/` under namespace `Library` — all entity types are `Library::User`, `Library::Book`, `Library::Review`, and actions are `Library::Action::"..."`.

**Store abstraction:** `internal/store/store.go` defines the `Store` interface. Two implementations:
- `pbstore` — production, wraps PocketBase `core.App`
- `memory` — in-memory, used in tests alongside pbstore

Both implementations are tested via a shared test suite in `store_test.go` that runs every test against both backends.

**Protobuf/ConnectRPC:** Proto definitions in `proto/`, generated code in `gen/` (gitignored). `buf generate` regenerates. The generated `libraryv1connect` package provides the service handler interface that `library.Server` implements.

**Migrations:** PocketBase migrations in `migrations/`. Imported via blank import in `main.go`. Auto-applied when running with `go run`.
