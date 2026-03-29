[Docs](../README.md) / Contributing

# Contributing

Guidelines for developing features, fixing bugs, and submitting changes to imlate.

## Development Workflow

1. Create a branch from `develop` (or `main` for hotfixes).
2. Make changes following the style guidelines below.
3. Run tests: `make got` (or `go test ./...` locally).
4. Run linter: `make gol` (or `golangci-lint run`).
5. Format code: `gofmt -w` on changed files.
6. Commit with a descriptive message.
7. Open a pull request.

## Code Style

### Formatting and Imports

- Follow `gofmt` exactly. Run `gofmt -w <files>` on changed files.
- Use standard Go import grouping: stdlib, then external packages, then internal packages.
- Remove unused imports. Use blank imports (`_`) only for required side effects (e.g., database drivers).

### Naming

- Exported names: `PascalCase`.
- Unexported helpers: `camelCase`.
- JSON struct tags: `snake_case` (e.g., `visitor_id`, `signed_in`).
- Keep interfaces at package boundaries, typically in `internal/domain/provider/`.

### Error Handling

- Prefer early returns with `if err != nil`.
- Add context with `fmt.Errorf("descriptive message: %w", err)` where useful.
- Do not swallow errors silently.

### Architecture Rules

- Keep domain logic in `internal/usecase/` and `internal/domain/`, not in controllers.
- Handlers should be thin: parse request, call a dependency, map the response.
- Register services in `internal/infrastructure/gocontainer/`.
- Put schema changes in `migrations/` with explicit SQL (no ORM auto-migration).

### Logging

- Use the zap logger from `internal/infrastructure/logging`.
- Prefer structured logging with key-value pairs.

### API Documentation

- Keep Swagger annotations in sync when changing admin endpoints.
- Regenerate with `make swag` after modifying annotations.

## Database Migrations

Migrations use [golang-migrate](https://github.com/golang-migrate/migrate) with explicit SQL files.

### Creating a Migration

```bash
make migrate-create name=describe-the-change
```

This creates `migrations/NNNNNN_describe-the-change.up.sql` and `.down.sql` files.

### Writing Migrations

- Always provide both `up` and `down` migrations.
- Use explicit SQL (`ALTER TABLE`, `CREATE TABLE`), not ORM-generated DDL.
- Handle data transformations in the migration when column constraints change (e.g., update NULL values before adding `NOT NULL`).
- Test rollback: `make migrate-down` then `make migrate-up`.

### Applying Migrations

```bash
make migrate-up     # Apply pending migrations
make migrate-down   # Roll back the last migration
```

Migrations also run automatically on application startup via `db.Open()`.

## Commit Messages

Use clear, descriptive commit messages. Recommended format:

```
<type>: <short description>

<optional body with more context>
```

Common types:
- `feat` -- new feature
- `fix` -- bug fix
- `refactor` -- code restructuring without behavior change
- `docs` -- documentation only
- `test` -- adding or updating tests
- `chore` -- tooling, CI, dependencies

Examples:

```
feat: add paginated visit attendance report endpoint
fix: skip ERP API calls for division ID 0
refactor: remove CSV file logging from track repository
docs: add developer documentation
test: add tests for manual track handler with signed_in
```

## PR Guidelines

- Keep PRs focused on a single concern.
- Include tests for new functionality.
- Ensure `make got` passes before requesting review.
- Ensure `make gol` passes with no new warnings.
- Reference related issues in the PR description.

## Useful Commands Reference

| Command | Description |
|---------|-------------|
| `make start` | Start Docker environment |
| `make got` | Run all tests |
| `make gotc` | Run tests with coverage |
| `make gol` | Run linter |
| `make golf` | Run linter with auto-fix |
| `make swag` | Regenerate Swagger docs |
| `make shell` | Open bash in app container |
| `make migrate-create name=...` | Create new migration |
| `make migrate-up` | Apply migrations |
| `make migrate-down` | Roll back last migration |

Back to [Docs Index](../README.md)
