[Docs](../README.md) / Testing

# Testing

How to run tests, write tests, generate coverage, and understand the CI test pipeline.

## Test Stack

| Library | Purpose |
|---------|---------|
| `testing` | Go standard test framework |
| `testify/assert` | Fluent assertions |
| `testify/mock` | Mock generation and expectations |
| `go-sqlmock` | SQL driver mock for repository tests |
| `zaptest` | Test-friendly zap logger |
| `httptest` | HTTP handler testing with `httptest.NewRecorder` |

## Running Tests

### Via Docker (Recommended)

Requires the Docker environment to be running for DB-dependent tests:

```bash
make start      # Start MySQL + app containers
make got        # Run all tests
make gotc       # Run tests with coverage report
```

### Locally

```bash
go test ./...                    # All tests
go test -v ./...                 # Verbose
go test -race ./...              # With race detector
```

### Single Package or Test

```bash
# Run a specific package
go test ./internal/isb/tracker/...

# Run a specific test by name
go test ./internal/isb/tracker/ -run TestTrackHandler_WithAdminID -v

# List all tests in a package
go test ./internal/isb/tracker/ -list .

# Via Docker
make shell
go test ./internal/isb/tracker/ -run TestTrackHandler_WithAdminID -v
```

## Coverage

### Generate Coverage Report

```bash
# Via Docker
make gotc

# Locally
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage-report.html
```

### View Coverage

Open `coverage-report.html` in a browser after running `make gotc` or the local commands above.

> Coverage artifacts (`coverage.out`, `coverage-report.html`) are gitignored and should not be committed.

## Writing Tests

### Test File Conventions

- Test files are co-located with the code they test: `foo.go` -> `foo_test.go`.
- Same package name (white-box testing).
- Import `testify/assert` and `testify/mock`.

### Handler Tests Pattern

HTTP handler tests use `gin.CreateTestContext` with `httptest.NewRecorder`:

```go
func TestMyHandler_Success(t *testing.T) {
    gin.SetMode(gin.TestMode)
    mockRepo := new(providertest.VisitorRepositoryMock)

    controller := &MyController{Repo: mockRepo}

    // Setup mock expectations
    mockRepo.On("FindById", int32(1)).Return(&entity.Visitor{Id: 1}, nil)

    // Create test request
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    body, _ := json.Marshal(map[string]interface{}{"id": 1})
    c.Request = httptest.NewRequest(http.MethodPost, "/path", bytes.NewReader(body))
    c.Request.Header.Set("Content-Type", "application/json")

    // Execute
    controller.MyHandler()(c)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
    mockRepo.AssertExpectations(t)
}
```

### Repository Tests Pattern

Repository tests use `go-sqlmock` to mock the database driver:

```go
func TestStore_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    assert.NoError(t, err)
    defer db.Close()

    repo := &repository.VisitorTrack{Connection: db}

    mock.ExpectExec("INSERT INTO track").
        WithArgs(/* expected args */).
        WillReturnResult(sqlmock.NewResult(1, 1))

    // ... setup query expectations for GetById ...

    result, err := repo.Store(track)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.NoError(t, mock.ExpectationsWereMet())
}
```

### Mock Repositories

Mock implementations are in `internal/domain/provider/providertest/`:

| Mock | Interface |
|------|-----------|
| `VisitorRepositoryMock` | `provider.VisitorRepository` |
| `VisitorTrackRepositoryMock` | `provider.VisitorTrackRepository` |

These are testify mocks. Use `mock.On(...)` to set expectations and `mock.AssertExpectations(t)` to verify.

### Test Helpers

Common helper functions used across test files:

```go
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func boolPtr(b bool) *bool    { return &b }
```

### Setting Up Auth Context in Tests

For handlers that require authenticated users:

```go
adminUser := &entity.User{ID: uuid.New(), Role: entity.UserRoleAdmin}
c.Set("id", adminUser)  // Simulates JWT middleware identity
```

## CI Test Flow

The GitHub Actions workflow (`.github/workflows/go-tests.yml`) runs on every push to `main`/`develop` and on all PRs:

1. Starts a MySQL 8 service container.
2. Sets up Go 1.26.7 with module caching.
3. Runs database migrations via `scripts/migrate-test-db.sh` against `tracker_test`.
4. Executes `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`.
5. Uploads coverage to Codecov.
6. Generates and uploads an HTML coverage report as a build artifact.

### Local Equivalence

To replicate CI locally:

```bash
make start                # Ensures MySQL is running with tracker_test DB
make got                  # Runs all tests
```

## Test Categories

| Category | Location | DB Required | Description |
|----------|----------|-------------|-------------|
| Handler tests | `internal/isb/*/` | No | Test HTTP handlers with mocked repos |
| Repository tests | `internal/infrastructure/repository/` | No | Test SQL with go-sqlmock |
| Use case tests | `internal/usecase/*/` | No | Test business logic with mocked deps |
| Auth tests | `internal/infrastructure/http/auth/` | No | Test JWT and terminal auth |
| Config tests | `internal/config/` | No | Test env parsing and validation |
| Integration tests | `internal/infrastructure/integration/isams/` | No | Test iSAMS client with httptest server |

> All tests currently use mocks and do not require a running database. However, some test scenarios (especially around migrations and full stack) benefit from the Docker MySQL setup.

Back to [Docs Index](../README.md)
