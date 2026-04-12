[Docs](../README.md) / [Architecture](README.md) / Services

# Package Map

Detailed breakdown of every package under `internal/` and its responsibilities.

## Configuration

### `internal/config`

Environment-based configuration loaded via `caarlos0/env`. The `Config` struct holds all settings with `env` struct tags and sensible defaults. Key methods:

- `NewFromEnv()` -- parses env vars, resolves DB URL from individual vars, loads timezones.
- `Validate()` -- enforces `AUTH_TOKEN_SECRET` minimum length.
- `IsERPIntegrated()` -- returns `true` only when ERP is enabled and all iSAMS credentials are set.

## Domain Layer

### `internal/domain/entity`

Core data structures with no external dependencies:

| Type | Description |
|------|-------------|
| `User` | Admin or terminal user with UUID, bcrypt password, role, soft-delete fields |
| `UserRole` | String enum: `admin`, `terminal` |
| `Visitor` | Person record with optional iSAMS linkage, keys, year group, divisions |
| `VisitTrack` | Single visit event with visitor ID, optional key, sign-in flag, optional admin UUID |
| `VisitDetails` | Visitor + key lookup result |
| `Schedule` | Registration period schedule from ERP |

### `internal/domain/provider`

Repository interfaces that define the contract between use cases and data access:

| Interface | Key Methods |
|-----------|-------------|
| `VisitorRepository` | `FindById`, `FindByKey`, `Store`, `Update`, `Delete`, `GetAll`, `AddKey`, `RemoveKey` |
| `VisitorTrackRepository` | `Store`, `GetById`, `CountEventsByVisitorIdSince`, `GetVisitReport` |
| `UserRepository` | `FindByID`, `FindByUsername`, `GetAll`, `Store`, `Update`, `Delete`, `UpdatePassword` |

Mock implementations live in `internal/domain/provider/providertest/` for use in unit tests.

### `internal/domain/erp`

Abstract ERP integration interfaces:

| Type | Description |
|------|-------------|
| `Client` | Methods for registration periods, student data, absence codes, attendance updates |
| `Factory` | Creates authenticated `Client` instances per request context |

## Use Case Layer

### `internal/usecase/adminapi`

Admin panel business logic:

- `AdminAPI` struct holds `VisitorRepo`, `UserRepo`, and provides orchestration methods.
- `GetReportsVisits()` -- paginated visit attendance report with date range, filters, sign-status logic.
- User CRUD, visitor CRUD, terminal registration logic.

### `internal/usecase/theme`

Theme and tracking-page customization logic:

- `Service` -- loads and persists `storage/theme/theme.json`, applies defaults, validates settings.
- Builds public reader page data including favicon/logo/animation URLs and welcome/goodbye durations.
- Handles custom asset assignment, reset-to-default, and theme response mapping for admin API.

### `internal/usecase/tracking`

Student attendance tracking against ERP:

- `StudentTracker` -- when a student is tracked, fetches current registration periods from iSAMS and writes attendance/registration status back.
- `TrackUntrackedStudentsAsAbsence()` -- cron-driven: marks students who have not registered as absent in iSAMS.

### `internal/usecase/synchroniser`

ERP data synchronization:

- `StudentSync` -- pages through iSAMS students, upserts into the local `visitors` table, syncs photos.
- Registration code dictionary sync -- pulls absence/present codes from iSAMS and caches locally.

## HTTP Delivery Layer

### `internal/http/controller/adminapi`

Gin handlers for `/admin-api/*` routes. Each handler is a method on `AdminAPIController` returning `gin.HandlerFunc`:

- User management (CRUD, password update)
- Visitor management (CRUD, keys, image upload)
- Theme management (current theme, asset upload/reset, animation timing update)
- `VisitsReportsHandler` -- attendance report endpoint
- `ManualTrackHandler` -- manual sign-in/out by admin
- `RegisterTerminalHandler` -- terminal device pairing
- Auth wrappers (`LoginRequest`, `RefreshRequest`, response types)

### `internal/isb/tracker`

Gin handlers for internal tracker endpoints called by physical terminal devices:

- `TrackHandler` (`POST /track`) -- records a visit by visitor ID
- `FindAndTrackHandler` (`POST /find-and-track`) -- looks up visitor by key and records a visit
- `ChangeTimeHandler` (`POST /change-time`) -- dev utility to override current time
- `Request` struct includes optional `admin_id` (UUID of the terminal's admin user) for audit

### `internal/isb/search`

- `SearchHandler` (`GET /search/:id`) -- looks up a visitor by key for terminal display

### `internal/isb/visitor`

Visitor-related delivery helpers and types.

## Infrastructure Layer

### `internal/infrastructure/db`

Database connection and migration:

- `Open(engine, dsn, logger)` -- opens a `*sql.DB` connection for `mysql` or `sqlite3` and runs `MigrateUp` automatically.
- Supports both MySQL (production) and SQLite (testing/local).

### `internal/infrastructure/repository`

SQL implementations of domain provider interfaces:

| Struct | Implements | Description |
|--------|-----------|-------------|
| `Visitor` | `VisitorRepository` | Full visitor CRUD with keys, soft delete, iSAMS fields |
| `VisitorTrack` | `VisitorTrackRepository` | Track storage, event counting, visit report query |
| `UserMySQL` | `UserRepository` | User CRUD with bcrypt password handling |

### `internal/infrastructure/http/auth`

Two authentication mechanisms:

- `AuthManager` -- JWT-based admin authentication using `gin-jwt/v3`. HS256 with configurable secret, 30-minute token timeout, role-based authorization (admin only).
- `TerminalAuthMiddleware` -- shared-secret authentication for terminal devices. Requires `Terminal-Name` header (matches a terminal user's username) and `Authorization: Bearer <token>` (validated via bcrypt against stored password).

### `internal/infrastructure/integration/isams`

iSAMS REST API client:

- OAuth2 client credentials flow for token acquisition.
- Methods for students, registration periods, absence/present codes, student photos, attendance updates.
- `ClientFactory` creates per-request authenticated clients.

### `internal/infrastructure/gocontainer`

DI wiring. The `Build(*config.Config)` function registers all singletons into the golobby global container. Called once from `main()`.

### `internal/infrastructure/cron`

Scheduled jobs using `gocron/v2`. Only activated when `Config.IsERPIntegrated()` returns `true`. Jobs include student sync, photo sync, registration code sync, and mark-absent.

### `internal/infrastructure/logging`

Zap logger initialization with structured logging support.

### `internal/infrastructure/util`

Utility functions: application time management (overridable clock for testing), file existence checks, root path resolution.

## Other Top-Level Packages

### `internal/version`

Build-injected version string. Returns `2.0.x-dev` in development, actual release version in production builds.

Back to [Architecture](README.md) | [Docs Index](../README.md)
