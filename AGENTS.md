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
- [Architecture & package map](docs/architecture/README.md) — layered design, DI, request flow.
- [API reference](docs/api/README.md) — all endpoints, auth schemes, Swagger annotations.
- [Configuration](docs/getting-started/configuration.md) — full env var reference.
- [Testing patterns](docs/testing/README.md) — test stack, mocks, coverage.
- [Infrastructure](docs/infrastructure/README.md) — Docker, CI, migrations.

---

## Default Development Environment: Docker

**Docker is the default and recommended development environment.** All project tooling (Go, linters, test runner, swagger generator, migrate CLI, hot-reload) is pre-installed in the dev container. You do not need a local Go installation.

### First-time Setup
```bash
cp docker/.env.docker.example .env
# Edit .env: set AUTH_TOKEN_SECRET to a random string of at least 32 characters
make start
curl http://localhost:8080/ping   # → {"message":"pong"}
```

### Docker-First Rule
- **Always prefer `make <target>`** over running Go/lint/test commands directly on the host.
- `make got`, `make gol`, `make swag` etc. run inside the container with correct tool versions.
- Direct `go test ./...` on host is fine for quick iteration when Go is installed locally, but
  Docker is the authoritative environment used in CI.
- Open a shell in the container with `make shell`.

---

## Complete Make Command Reference
All targets below delegate to `docker/docker-dev.sh` unless noted.

### Environment Lifecycle
| Command | Description |
|---------|-------------|
| `make start` | Start Docker environment (MySQL + app) |
| `make stop` | Stop Docker environment |
| `make restart` | Restart everything |
| `make restart-app` | Restart only the app container |
| `make status` | Show container status |
| `make logs` | Tail all service logs |
| `make logs-app` | Tail application logs |
| `make logs-mysql` | Tail MySQL logs |
| `make shell` | Open bash in the app container |
| `make clean` | Remove containers and volumes |
| `make rebuild` | Rebuild images and restart |

### Build & Run
| Command | Description |
|---------|-------------|
| `make install-mod` | `go mod download` inside container |
| `make build-app` | Build the application binary |
| `make start-app` | Run the app (inside container) |
| `make dist` | Build distributable binary for `linux/amd64` |
| `make dist GOOS=linux GOARCH=arm64` | Cross-compile for target platform |

### Testing
| Command | Description |
|---------|-------------|
| `make got` | `go test ./...` (Docker — preferred) |
| `make gotc` | Tests with coverage report (Docker — preferred) |

Local equivalents (when Go is installed on host):
```bash
go test ./...
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go test ./internal/usecase/adminapi/... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out
```

### Linting
| Command | Description |
|---------|-------------|
| `make gol` | `golangci-lint run ./...` (Docker — preferred) |
| `make golf` | `golangci-lint run --fix ./...` |

### Swagger
| Command | Description |
|---------|-------------|
| `make swag` | Regenerate Swagger docs from annotations |

**Always run `make swag` after adding or modifying any HTTP endpoint**, including adding parameters, changing response types, or marking an endpoint as deprecated.
- Annotations are in the handler godoc immediately above the function.
- Generated files: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.
- Swagger UI: `http://localhost:8080/swagger/index.html`.

### Database
| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-create name=<name>` | Scaffold a new migration pair |
| `make mysql-shell` | Open MySQL CLI |

### UI
| Command | Description |
|---------|-------------|
| `make update-ui version=<tag>` | Download and install admin UI release |

---

## Recommended Verification Flow

For any code change, in order:

1. **Format** — `gofmt -w` on touched files (or `go fmt ./...`)
2. **Test affected packages** — `go test ./path/to/package/...`
3. **Test all** — `make got` (in Docker) or `go test ./...`
4. **Lint** — `make gol` or `golangci-lint run ./internal/...`
5. **Swagger** — `make swag` if any HTTP handler was changed

For DB-related changes, use Docker to ensure migrations are applied before tests:
```bash
make start       # ensure MySQL is running
make migrate-up  # apply migrations
make got         # run all tests
```

---

## Admin API Development Pattern

All new admin API endpoints follow the same 3-layer pattern. **Implement in this order:**

### 1. Domain layer — `internal/domain/provider/`
- Add or extend filter structs, result row structs, and repository interface methods.
- Use slices (`[]string`, `[]int`) for multi-value filters instead of single pointers.
- Keep field names snake_case in JSON tags.

### 2. Repository layer — `internal/infrastructure/repository/`
- Implement the interface method (SQL, scanning, query building).
- Use helper methods to build modular query parts (`buildWhereClause`, `buildHavingClause`, `buildOrderClause`).
- Add `email` and `image` to SELECT when visitor profile fields are needed.
- Always test with `go-sqlmock`; add `email`/`image` to mock column lists when extending the report query.

### 3. Usecase layer — `internal/usecase/adminapi/`
- Business logic, validation, and response mapping live here.
- Validate request inputs; return `fmt.Errorf("%w: ...", ErrInvalidRequestFormat)` for 400-class errors.
- Use option functions (`With*`) pattern for the GET-style endpoint; use a request struct for POST-style endpoints.
- For dynamic field selection (user picks a subset of fields), return `[]map[string]interface{}` and keep the allowed field set in an exported `map[string]bool` so it can be extended.
- For allowed sort fields, define an unexported `map[string]bool`; validate in the usecase, map to SQL expressions in the repository.

### 4. Controller layer — `internal/http/controller/adminapi/`
- Keep handlers thin: parse request → call usecase → map error → return JSON.
- Add Swagger-visible type aliases to `swagger_dto.go`:
  ```go
  type MyRequest  = usecase.MyRequest
  type MyResponse = usecase.MyResponse
  ```
- Swagger annotations format for a POST endpoint with a JSON body:
  ```go
  // @Param  body  body  MyRequest  true  "Description"
  // @Success 200 {object} MyResponse
  ```
- Mark a deprecated endpoint with `// @Deprecated  true` in the Swagger block.
- Error mapping pattern:
  ```go
  if errors.Is(err, usecase.ErrInvalidRequestFormat) {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
  }
  c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
  ```

### 5. Route registration — `cmd/app/main.go`
```go
adminGroup.GET("/resources", adminController.ResourcesHandler())
adminGroup.POST("/resources", adminController.ResourcesPostHandler())
```

### 6. Swagger regeneration
```bash
make swag
```

### 7. Tests — write tests for every new function
See testing section below.

---

## Testing Standards

### Coverage Requirement
**All new code must have 100% test coverage.** Check coverage after writing tests:
```bash
go test ./path/to/package/... -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | grep "your_file"
```

### Test File Conventions
- Tests live alongside source files in the same package.
- Repository tests: `internal/infrastructure/repository/*_test.go`
- Usecase tests: `internal/usecase/adminapi/*_test.go`
- Controller tests: `internal/http/controller/adminapi/*_test.go`
- Mock implementations: `internal/domain/provider/providertest/`

### Controller Tests
```go
func setupMyTest() (*providertest.MyRepoMock, *AdminAPIController) {
    mockRepo := new(providertest.MyRepoMock)
    api := &usecase.AdminAPI{MyRepo: mockRepo}
    controller := &AdminAPIController{AdminAPI: api}
    gin.SetMode(gin.TestMode)
    return mockRepo, controller
}

func TestMyHandler_Success(t *testing.T) {
    mockRepo, controller := setupMyTest()
    mockRepo.On("Method", mock.Anything, ...).Return(result, nil)

    body, _ := json.Marshal(request)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest(http.MethodPost, "/path", bytes.NewBuffer(body))
    c.Request.Header.Set("Content-Type", "application/json")

    controller.MyHandler()(c)

    assert.Equal(t, http.StatusOK, w.Code)
    mockRepo.AssertExpectations(t)
}
```

### Usecase Tests
- Use `providertest.*Mock` structs with `mock.On("Method", expectedArgs...).Return(...)`.
- Use `mock.MatchedBy(func(f FilterType) bool { ... })` when you need to assert on specific filter fields without matching the whole struct.
- Use `mock.Anything` for time arguments that are computed internally.
- Always call `mockRepo.AssertExpectations(t)` at the end.

### Repository Tests (sqlmock)
- `reportDataColumns` must include all columns in the SELECT order — keep it in sync with `buildReportBaseQuery`.
- When adding columns to a query, also update all `AddRow(...)` calls in the test file.
- HAVING conditions have no bound parameters; args in `WithArgs` only cover WHERE placeholders.
- Test `buildHavingClause`, `buildOrderClause`, `buildVisitorFilters` directly as unit tests (no DB needed).

### Mock Generation
When adding a new method to a repository interface, also add the corresponding method to the mock in `internal/domain/provider/providertest/`. Pattern:
```go
func (m *MyRepoMock) NewMethod(arg Type) (ReturnType, error) {
    args := m.Called(arg)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(ReturnType), args.Error(1)
}
```

---

## Style Guidelines

### Formatting & Imports
- Follow `gofmt` exactly; use Go-standard import grouping (stdlib → external → internal).
- Remove unused imports; use blank imports only for required side effects.

### Naming & Types
- Exported names: PascalCase; unexported helpers: camelCase.
- Interfaces at package boundaries: `internal/domain/provider`.
- JSON tags: consistent snake_case (e.g., `visitor_id`, `year_group`).
- Multi-value filters: use `[]string` / `[]int` slices, not `*string` / `*int`.
- Allowlists: use `map[string]bool` for O(1) membership tests.

### Errors & Flow
- Prefer early returns with `if err != nil`.
- Wrap errors with context: `fmt.Errorf("failed to ...: %w", err)`.
- Domain validation errors: `fmt.Errorf("%w: specific reason", ErrInvalidRequestFormat)`.
- Keep handlers thin: parse request, call dependency, map response.

### Architecture Rules
- Domain logic in `internal/usecase` / `internal/domain`, never in controllers.
- Controllers only: parse request → call usecase → map response.
- Register new services in `internal/infrastructure/gocontainer`.
- Schema changes go in `migrations/` as explicit SQL.
- Swagger annotations kept in sync whenever endpoints change.

### Logging & Config
- Use zap logger from `internal/infrastructure/logging`.
- All env-backed config in `internal/config`; avoid scattered `os.Getenv` calls.

---

## Common Pitfalls
- Mixed DB engine naming (`mysql`, `sqlite`, `sqlite3`): follow nearby code.
- `AUTH_TOKEN_SECRET` must be at least 32 characters — validation runs at startup.
- `ERP_INTEGRATION_ENABLED=false` disables ERP behavior even when other ERP vars are set.
- DB-backed tests require MySQL + applied migrations; run `make start` before `make got`.
- Keep MySQL DSN with `?parseTime=true`.
- `make swag` runs the swaggo CLI directly on the host (not in Docker); `swag` must be in PATH, or use the dev container.
- Do not commit: `.env`, `tracker`, `dist/`, `coverage.out`, `coverage-report.html`, `app/`.

## Command Sources of Truth
- Prefer: `Makefile`, `docker/docker-dev.sh`, `.github/workflows/go-tests.yml`, `CONTRIBUTING.md`.
- `devenv.nix` is partly stale; prefer Make/Docker/CI docs.
