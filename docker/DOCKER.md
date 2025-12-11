# Docker Environment for imlate Project

This project includes a complete Docker development environment that mirrors the functionality from `devenv.nix`.

## Quick Start

### Prerequisites
- Docker Desktop installed and running
- Docker Compose v2.0+

### Start Development Environment

```bash
# Make the script executable (first time only)
chmod +x docker-dev.sh

# Start the environment
./docker-dev.sh start
```

The application will be available at:
- **Application**: http://localhost:8080
- **MySQL**: localhost:3307 (user: trackme, password: trackme, database: tracker)

## Available Commands

### Environment Management

```bash
./docker-dev.sh start          # Start all services
./docker-dev.sh stop           # Stop all services
./docker-dev.sh restart        # Restart all services
./docker-dev.sh status         # Show container status
./docker-dev.sh logs           # View all logs
./docker-dev.sh logs app       # View app logs only
./docker-dev.sh logs mysql     # View MySQL logs only
```

### Development Commands

```bash
./docker-dev.sh install-mod    # Install Go modules (go mod download)
./docker-dev.sh build-app      # Build the application
./docker-dev.sh run-app        # Run the application
```

### Testing & Linting

```bash
./docker-dev.sh got            # Run tests (go test ./...)
./docker-dev.sh gotc           # Run tests with coverage
./docker-dev.sh gol            # Run golangci-lint
./docker-dev.sh golf           # Run golangci-lint with auto-fix
```

### Database Management

```bash
./docker-dev.sh migrate-up                    # Apply all migrations
./docker-dev.sh migrate-down                  # Rollback migrations
./docker-dev.sh migrate-create <name>         # Create new migration
./docker-dev.sh mysql-shell                   # Access MySQL shell
```

### Utilities

```bash
./docker-dev.sh exec bash                     # Open bash shell in app container
./docker-dev.sh exec <any-command>            # Execute any command in app container
./docker-dev.sh rebuild                       # Rebuild containers from scratch
./docker-dev.sh clean                         # Remove all containers and volumes
./docker-dev.sh help                          # Show help message
```

## Project Structure

```
.
├── Dockerfile                 # Production Dockerfile
├── Dockerfile.dev            # Development Dockerfile with hot-reload support
├── docker-compose.yml        # Docker Compose configuration
├── docker-dev.sh             # Development management script
└── docker/
    └── mysql/
        └── init.sql          # MySQL initialization script
```

## Environment Variables

The following environment variables are configured in `docker-compose.yml`:

```yaml
ENVIRONMENT: development
DATABASE_HOST: mysql
DATABASE_PORT: 3306
DATABASE_USERNAME: trackme
DATABASE_PASSWORD: trackme
DATABASE_NAME: tracker
```

You can override these by creating a `.env` file in the project root.

## Development Workflow

### 1. Start the Environment
```bash
./docker-dev.sh start
```

### 2. Run Tests
```bash
./docker-dev.sh got
```

### 3. Run Linter
```bash
./docker-dev.sh golf
```

### 4. Apply Migrations
```bash
./docker-dev.sh migrate-up
```

### 5. View Logs
```bash
./docker-dev.sh logs app
```

### 6. Access MySQL
```bash
./docker-dev.sh mysql-shell
```

## Installed Tools

The development container includes:
- Go 1.22.1
- golangci-lint (latest)
- migrate (golang-migrate/migrate with MySQL support)
- swag (Swagger documentation generator)
- air (hot-reload for Go - optional)
- MySQL client

## Volumes

- **mysql_data**: Persistent MySQL data
- **go_modules**: Cached Go modules for faster builds
- **./**: Application source code (mounted for live editing)

## Troubleshooting

### Containers won't start
```bash
./docker-dev.sh clean
./docker-dev.sh start
```

### Database connection issues
```bash
./docker-dev.sh logs mysql
./docker-dev.sh mysql-shell
```

### Port conflicts
If port 8080 or 3307 is already in use, edit `docker-compose.yml`:
```yaml
ports:
  - "8081:8080"  # Change host port
```

### Rebuild from scratch
```bash
./docker-dev.sh clean
./docker-dev.sh rebuild
./docker-dev.sh start
```

## Comparison with devenv.nix

| devenv.nix Command | Docker Equivalent | Description |
|--------------------|-------------------|-------------|
| `devenv up` | `./docker-dev.sh start` | Start environment |
| `install-mod` | `./docker-dev.sh install-mod` | Install Go modules |
| `build-app` | `./docker-dev.sh build-app` | Build application |
| `gol` | `./docker-dev.sh gol` | Run linter |
| `golf` | `./docker-dev.sh golf` | Run linter with fix |
| `got` | `./docker-dev.sh got` | Run tests |
| `gotc` | `./docker-dev.sh gotc` | Tests with coverage |
| `start-app` | `./docker-dev.sh run-app` | Run application |
| MySQL service | `mysql` container | Database service |

## Production Build

To build for production:

```bash
docker build -t imlate-app:latest .
docker run -p 8080:8080 \
  -e DATABASE_HOST=your-db-host \
  -e DATABASE_PORT=3306 \
  -e DATABASE_USERNAME=trackme \
  -e DATABASE_PASSWORD=trackme \
  -e DATABASE_NAME=tracker \
  imlate-app:latest
```

## Notes

- The development environment automatically applies migrations on startup
- Source code changes are reflected immediately (live reload)
- MySQL data persists between container restarts
- Go modules are cached for faster rebuilds
