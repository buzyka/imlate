[Docs](../README.md) / Getting Started

# Getting Started

This guide covers setting up the imlate development environment, running the application, and verifying it works.

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.26+ | Required for local builds; Docker setup includes Go |
| Docker | 20.10+ | With Docker Compose v2 |
| Make | any | Thin wrapper around `docker/docker-dev.sh` |
| MySQL client | optional | For direct DB access via `make mysql-shell` |

## Clone the Repository

```bash
git clone git@github.com:buzyka/imlate.git
cd imlate
```

## Setup Options

### Option 1: Docker (Recommended)

This starts MySQL 8 and the Go application in containers with hot-reload.

1. Copy the environment file and set `AUTH_TOKEN_SECRET` (minimum 32 characters):

```bash
cp docker/.env.docker.example .env
# Edit .env and set AUTH_TOKEN_SECRET to a secure value
```

2. Start the environment:

```bash
make start
```

This runs `docker compose up`, waits for MySQL health check, applies migrations, and starts the app with `go run`.

3. Verify:

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

4. View logs:

```bash
make logs-app    # Application logs
make logs-mysql  # MySQL logs
```

### Option 2: Local Go (without Docker)

Requires a running MySQL 8 instance with a `tracker` database.

1. Set environment variables (or create a `.env` file in the project root):

```bash
export DATABASE_HOST=127.0.0.1
export DATABASE_PORT=3306
export DATABASE_USERNAME=trackme
export DATABASE_PASSWORD=trackme
export DATABASE_NAME=tracker
export AUTH_TOKEN_SECRET="your-secret-at-least-32-characters-long"
```

2. Install dependencies and run:

```bash
go mod download
go run cmd/app/main.go
```

Migrations are applied automatically on startup via `db.Open`.

## Verify the Setup

Once the application is running on port 8080:

| URL | Description |
|-----|-------------|
| `http://localhost:8080/ping` | Health check -- returns `{"message":"pong"}` |
| `http://localhost:8080/swagger/index.html` | Interactive Swagger UI for admin API |
| `http://localhost:8080/admin` | Admin SPA (requires built frontend assets) |
| `http://localhost:8080/` | Reader terminal UI |

## Default Users

Migration `000006` seeds two users:

| Username | Role | Notes |
|----------|------|-------|
| `admin` | admin | Default admin account for `/admin-api` |
| `terminal` | terminal | Default terminal device account for `/track` endpoints |

> Passwords are set as bcrypt hashes in the migration. Change them in production.

## Useful Commands

```bash
make start           # Start Docker environment
make stop            # Stop Docker environment
make restart         # Restart everything
make restart-app     # Restart only the app container
make shell           # Open bash in the app container
make mysql-shell     # Open MySQL CLI
make migrate-up      # Apply pending migrations
make migrate-down    # Roll back last migration
```

See the full command reference in the [Makefile](../../Makefile).

## Next Steps

- [Configuration Reference](configuration.md) -- All environment variables explained.
- [Architecture Overview](../architecture/README.md) -- How the codebase is structured.
- [API Reference](../api/README.md) -- All HTTP endpoints.

Back to [Docs Index](../README.md)
