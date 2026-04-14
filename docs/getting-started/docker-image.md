[Docs](../README.md) / [Getting Started](README.md) / Docker Image

# Running imlate with Docker

Pre-built production images are published to GitHub Container Registry on every release. No Go toolchain or repository clone required.

## Prerequisites

- Docker 20.10+ (or Podman)
- A running **MySQL 8** instance accessible from the container
- An `AUTH_TOKEN_SECRET` value of at least 32 characters

## Pull the image

```bash
docker pull ghcr.io/buzyka/imlate:latest
```

Available tags:

| Tag | Description |
|-----|-------------|
| `latest` | Most recent published release |
| `1.2.3` | Specific release version |
| `my-feature-dev` | Dev build from a branch (see [Dev images](#dev-images)) |

## Environment variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `AUTH_TOKEN_SECRET` | JWT signing secret — **minimum 32 characters** | `a-long-random-secret-at-least-32-chars` |
| `DATABASE_HOST` | MySQL host | `db` |
| `DATABASE_USERNAME` | MySQL user | `imlate` |
| `DATABASE_PASSWORD` | MySQL password | `imlate` |
| `DATABASE_NAME` | MySQL database name | `tracker` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `8080` | HTTP listen port inside the container |
| `DATABASE_PORT` | `3306` | MySQL port |
| `ENVIRONMENT` | `production` | Runtime environment (`development`, `staging`, `production`) |
| `APP_LOCAL_TIMEZONE` | `UTC` | IANA timezone for the application (e.g. `Europe/London`) |
| `DEBUG` | `false` | Enable verbose debug logging |

For iSAMS ERP integration variables see the [Configuration Reference](configuration.md).

## Minimal `docker run` example

```bash
docker run -d \
  --name imlate \
  -p 8080:8080 \
  -e AUTH_TOKEN_SECRET="replace-with-at-least-32-character-secret" \
  -e DATABASE_HOST="<mysql-host>" \
  -e DATABASE_USERNAME="<mysql-user>" \
  -e DATABASE_PASSWORD="<mysql-password>" \
  -e DATABASE_NAME="tracker" \
  ghcr.io/buzyka/imlate:latest
```

Verify it is running:

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

## Docker Compose example

```yaml
services:
  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: tracker
      MYSQL_USER: imlate
      MYSQL_PASSWORD: imlate
    volumes:
      - db_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "imlate", "-pimlate"]
      interval: 5s
      timeout: 5s
      retries: 10

  app:
    image: ghcr.io/buzyka/imlate:latest
    ports:
      - "8080:8080"
    environment:
      AUTH_TOKEN_SECRET: "replace-with-at-least-32-character-secret"
      DATABASE_HOST: db
      DATABASE_PORT: 3306
      DATABASE_USERNAME: imlate
      DATABASE_PASSWORD: imlate
      DATABASE_NAME: tracker
    depends_on:
      db:
        condition: service_healthy

volumes:
  db_data:
```

Save as `docker-compose.yml` and run:

```bash
docker compose up -d
```

## Ports

The container listens on port `8080` by default. Override with the `APP_PORT` environment variable and adjust your port mapping accordingly:

```bash
-e APP_PORT=9090 -p 9090:9090
```

## Health check

The application exposes a health endpoint at `GET /ping`:

```bash
curl http://localhost:8080/ping
# {"message":"pong"}
```

Use it in Docker or your orchestrator:

```yaml
healthcheck:
  test: ["CMD", "wget", "-qO-", "http://localhost:8080/ping"]
  interval: 10s
  timeout: 5s
  retries: 5
```

## First-run database initialisation

On every start the application automatically applies any pending database migrations. If the database is empty, the full schema is created and two default users are seeded:

| Username | Role | Used for |
|----------|------|----------|
| `admin` | admin | Admin panel (`/admin-api`) |
| `terminal` | terminal | Tracking devices (`/track`) |

**Change the default passwords immediately after first login.**

Migrations are idempotent — restarting the container against an already-initialised database is always safe.

## Persistent storage

The container writes uploaded images and theme files to `storage/` inside the container. Mount a volume to keep them across restarts:

```yaml
volumes:
  - storage_data:/app/storage
```

## Dev images

Images built from feature branches are tagged `<name>-dev` and can be triggered manually from the GitHub Actions tab. They are intended for testing only and are never tagged `latest`.

---

Back to [Getting Started](README.md) · [Configuration Reference](configuration.md)
