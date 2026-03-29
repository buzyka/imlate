# imlate

A visitor and student tracking system built with Go. Tracks sign-in/sign-out events via physical terminal devices or manual admin actions, provides an admin panel for visitor and user management, generates attendance reports, and optionally synchronizes student data with iSAMS (school management ERP).

## Features

- **Visitor tracking** -- sign-in/out via physical key readers or manual admin entry.
- **Admin panel** -- web-based SPA for managing visitors, users, keys, and viewing reports.
- **Attendance reports** -- paginated, filterable reports aggregated per visitor per day.
- **Terminal authentication** -- dedicated auth scheme for physical tracking devices.
- **iSAMS integration** (optional) -- syncs students, photos, registration codes, and writes back attendance/absence data.
- **REST API** -- fully documented with Swagger/OpenAPI.

## Quick Start

```bash
# Clone the repository
git clone git@github.com:buzyka/imlate.git
cd imlate

# Copy environment config and set AUTH_TOKEN_SECRET (min 32 chars)
cp docker/.env.docker.example .env

# Start the development environment (MySQL + Go app)
make start

# Verify
curl http://localhost:8080/ping
# {"message":"pong"}

# View Swagger UI
open http://localhost:8080/swagger/index.html
```

See the [Getting Started guide](docs/getting-started/README.md) for full setup instructions including local (non-Docker) setup.

## Development Commands

| Command | Description |
|---------|-------------|
| `make start` | Start Docker environment (MySQL + app) |
| `make stop` | Stop Docker environment |
| `make restart` | Restart everything |
| `make restart-app` | Restart only the app container |
| `make logs-app` | Tail application logs |
| `make got` | Run all tests |
| `make gotc` | Run tests with coverage |
| `make gol` | Run linter |
| `make golf` | Run linter with auto-fix |
| `make swag` | Regenerate Swagger docs |
| `make shell` | Open bash in app container |
| `make mysql-shell` | Open MySQL CLI |
| `make migrate-up` | Apply database migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-create name=...` | Create new migration |
| `make dist` | Build distributable binary |

## Project Structure

```
.
├── cmd/app/main.go              # Application entry point
├── internal/
│   ├── config/                  # Environment-based configuration
│   ├── domain/
│   │   ├── entity/              # Core data structures
│   │   ├── provider/            # Repository interfaces
│   │   └── erp/                 # ERP client interface
│   ├── usecase/                 # Business logic
│   │   ├── adminapi/            # Admin operations, reports
│   │   ├── tracking/            # Student attendance tracking
│   │   └── synchroniser/        # ERP data sync
│   ├── isb/                     # HTTP handlers (Gin)
│   │   ├── adminapi/            # Admin API handlers
│   │   ├── tracker/             # Tracker device handlers
│   │   └── search/              # Visitor search handler
│   └── infrastructure/
│       ├── db/                  # Database connection + auto-migration
│       ├── repository/          # SQL implementations
│       ├── http/auth/           # JWT + terminal authentication
│       ├── integration/isams/   # iSAMS REST client
│       ├── gocontainer/         # Dependency injection
│       ├── cron/                # Scheduled jobs
│       └── logging/             # Zap logger
├── migrations/                  # SQL migrations (golang-migrate)
├── website/                     # Static UI (reader, admin SPA)
├── docker/                      # Dockerfiles, scripts, MySQL init
├── docs/                        # Developer documentation + Swagger
├── docker-compose.yml           # Development environment
└── Makefile                     # Command shortcuts
```

## Documentation

- [Getting Started](docs/getting-started/README.md) -- Setup, prerequisites, first run
- [Configuration](docs/getting-started/configuration.md) -- All environment variables
- [Architecture](docs/architecture/README.md) -- System design and package structure
- [API Reference](docs/api/README.md) -- Endpoints, auth, Swagger
- [Contributing](docs/contributing/README.md) -- Workflow, style, migrations
- [Infrastructure](docs/infrastructure/README.md) -- Docker, CI, database
- [Testing](docs/testing/README.md) -- Test stack, patterns, coverage

## Requirements

- **Go** 1.25+ (included in Docker setup)
- **Docker** 20.10+ with Docker Compose v2
- **Make** (any version)

## Environment

All configuration is via environment variables. Copy the example file and set `AUTH_TOKEN_SECRET`:

```bash
cp docker/.env.docker.example .env
```

Key variables:
- `AUTH_TOKEN_SECRET` -- JWT signing secret (required, min 32 characters)
- `ERP_INTEGRATION_ENABLED` -- enable iSAMS integration (default `false`)
- `DATABASE_*` -- database connection settings

See the full [Configuration Reference](docs/getting-started/configuration.md).

## License

Private repository.
