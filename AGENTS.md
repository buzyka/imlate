# AGENTS.md
Guide for agentic coding tools working in this repository.

## Project Snapshot
- Module: `github.com/buzyka/imlate`
- Go `1.25.7`
- Entry point: `cmd/app/main.go`
- Main stacks: Gin, JWT auth, Swagger, MySQL/SQLite, golang-migrate, zap, testify
- Core layout: `cmd/`, `internal/{config,domain,usecase,http,infrastructure,isb}`, `migrations/`, `website/`

## Developer Documentation
For detailed documentation beyond this agent guide, see [`docs/`](docs/README.md):
- [Architecture & package map](docs/architecture/README.md) -- layered design, DI, request flow.
- [API reference](docs/api/README.md) -- all endpoints, auth schemes, Swagger annotations.
- [Configuration](docs/getting-started/configuration.md) -- full env var reference.
- [Testing patterns](docs/testing/README.md) -- test stack, mocks, coverage.
- [Infrastructure](docs/infrastructure/README.md) -- Docker, CI, migrations.

## Command Sources Of Truth
- Prefer: `Makefile`, `docker/docker-dev.sh`, `.github/workflows/go-tests.yml`, `README.md`.
- `devenv.nix` is partly stale; prefer Make/Docker/CI docs.

## Common Commands
- Setup/build/run:
  - `make install-mod`
  - `make build-app`
  - `make dist`
  - `make start-app`
- Environment helpers:
  - `make start`, `make stop`, `make restart`, `make restart-app`, `make status`
  - `make logs-app`, `make logs-mysql`, `make shell`
- Local equivalents:
  - `go mod download`
  - `go build -o tracker cmd/app/main.go`
  - `go run cmd/app/main.go`
- Distribution build:
  - `make dist` builds `dist/app` inside the app container for `linux/amd64` by default.
  - Override target platform with `make dist GOOS=linux GOARCH=arm64`.
  - The command recreates `dist/`, sets executable permissions on `dist/app`, and copies `migrations/` plus `website/`.

## Lint Commands
- `make gol` -> `golangci-lint run`
- `make golf` -> `golangci-lint run --fix`
- Formatting:
  - `gofmt -w <files>` for changed files
  - `go fmt ./...` for broader formatting

## Test Commands
- Preferred wrappers:
  - `make got` -> `go test ./...` (Docker)
  - `make gotc` -> tests + coverage report (Docker)
- Local equivalents:
  - `go test ./...`
  - `go test -v ./...`
  - `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
- Single test patterns:
  - `go test <package> -run <TestName> -v`
  - `go test <package> -list .`
  - Docker: `./docker/docker-dev.sh exec go test <package> -run <TestName> -v`

## Database Commands
- Create a migration: `make migrate-create name=<migration_name>`.
- Apply migrations: `make migrate-up`.
- Roll back migrations: `make migrate-down`.
- Open MySQL shell: `make mysql-shell`.
- DB-dependent tests usually require MySQL + migrations.
- Safest local flow for DB-backed tests: `make start` then `make got`.
- Keep MySQL URLs with `?parseTime=true`.

## Common Pitfalls
- Mixed DB engine naming exists (`mysql`, `sqlite`, `sqlite3`); follow nearby code.
- Auth config is validated on startup: `AUTH_TOKEN_SECRET` must be at least 32 characters.
- `ERP_INTEGRATION_ENABLED=false` disables ERP-dependent behavior even if other ERP vars are set.
- Do not commit local artifacts: `.env`, `tracker`, `dist/`, `coverage.out`, `coverage-report.out`.

## Style Guidelines
- Formatting and imports:
  - Follow `gofmt` exactly; use Go-standard import grouping.
  - Remove unused imports; use blank imports only for required side effects.
- Naming and types:
  - Exported names: PascalCase; unexported helpers: camelCase.
  - Keep interfaces at package boundaries (commonly in `internal/domain/provider`).
  - Keep JSON tags consistent with existing snake_case fields (for example `visitor_id`).
- Errors and flow:
  - Prefer early returns with `if err != nil`.
  - Add context with `fmt.Errorf` where useful.
  - Keep handlers thin: parse request, call dependency, map response.
- Architecture:
  - Keep domain logic in `internal/usecase`/`internal/domain`, not controllers.
  - Register services in `internal/infrastructure/gocontainer`.
  - Put schema changes in `migrations/` with explicit SQL.
- Logging/tests/config:
  - Use zap logger from `internal/infrastructure/logging` and structured logs.
  - Testing stack: `testing`, `testify/assert`, `sqlmock`, `zaptest`; prefer table-driven tests.
  - Keep env-backed config in `internal/config`; avoid scattered `os.Getenv` reads.
- Auth/API:
  - Admin JWT middleware is wired through `internal/infrastructure/http/auth`.
  - Protected admin routes are under `/admin-api`.
  - Keep Swagger docs in sync when admin endpoints change.

## Practical Editing Guidance
- Preserve behavior unless the task explicitly changes it.
- Prefer targeted fixes over broad refactors.
- Check whether a change affects Docker startup, migrations, or DB assumptions.
- If you touch DB-backed tests, note whether local MySQL setup is required.
- Do not commit `.env`, coverage outputs, built binaries, or temp files.
- Ignore generated coverage artifacts already present unless the task is specifically about coverage output.

## Recommended Verification Flow
- Normal Go changes:
  - `gofmt -w` on touched files
  - run affected package tests
  - run broader tests: `go test ./...` or `make got`
  - run lint if needed: `golangci-lint run` or `make gol`
- DB-related changes:
  - `make start`
  - ensure migrations apply
  - run affected tests and finish with `make got`
