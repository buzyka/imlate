# Contributing to imlate

This guide covers everything you need to set up imlate locally for development and contribute code.

## Requirements

- **Go** 1.26+ (included in the Docker dev setup)
- **Docker** 20.10+ with Docker Compose v2
- **Make** (any version)

## Local Development Setup

```bash
# Clone the repository
git clone git@github.com:buzyka/imlate.git
cd imlate

# Copy the example environment file
cp docker/.env.docker.example .env
# Open .env and set AUTH_TOKEN_SECRET to a random string of at least 32 characters

# Start the development environment (MySQL + Go app with hot source mount)
make start

# Verify the app is running
curl http://localhost:8080/ping
# {"message":"pong"}

# Open the Swagger UI
open http://localhost:8080/swagger/index.html
```

## Configuration

All configuration is via environment variables loaded from `.env`. Key variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `AUTH_TOKEN_SECRET` | Yes | JWT signing secret, min 32 characters |
| `DATABASE_HOST` | Yes | MySQL host (set to `mysql` in Docker Compose) |
| `DATABASE_USERNAME` | Yes | MySQL user |
| `DATABASE_PASSWORD` | Yes | MySQL password |
| `DATABASE_NAME` | Yes | MySQL database name |
| `ERP_INTEGRATION_ENABLED` | No | Enable iSAMS sync (default `false`) |
| `ENVIRONMENT` | No | `development`, `staging`, or `production` |

See the full [Configuration Reference](docs/getting-started/configuration.md) for all available variables.

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
- [Contributing Guidelines](docs/contributing/README.md) -- Workflow, code style, migrations, PR guidelines
- [Infrastructure](docs/infrastructure/README.md) -- Docker, CI, database management
- [Testing](docs/testing/README.md) -- Test stack, patterns, coverage
