# imlate

A Go-based tracking application with Docker support.

## Quick Start

### Using Docker (Recommended)

```bash
# Start the development environment
make start
# or
./docker/docker-dev.sh start

# Run tests
make got

# Run linter with fix
make golf

# View logs
make logs-app
```

See [docker/DOCKER.md](docker/DOCKER.md) for detailed Docker documentation.

### Using devenv (Nix)

```bash
devenv up
```

## Development Commands

### With Docker/Make (same commands as devenv.nix)

```bash
make start           # Start development environment
make stop            # Stop development environment
make install-mod     # Install Go modules
make build-app       # Build application
make got             # Run tests
make gotc            # Run tests with coverage
make gol             # Run linter
make golf            # Run linter with fix
make start-app       # Start application
make migrate-up      # Apply database migrations
make mysql-shell     # Access MySQL shell
make clean           # Clean up everything
```

### With devenv.nix

```bash
install-mod          # Install Go modules
build-app            # Build application
got                  # Run tests
gotc                 # Run tests with coverage
gol                  # Run linter
golf                 # Run linter with fix
start-app            # Start application
```

## Project Structure

```
.
├── cmd/
│   └── app/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration
│   ├── infrastructure/      # Infrastructure layer
│   └── isb/                 # Business logic
├── migrations/              # Database migrations
├── website/                 # Static web assets
├── docker/                  # Docker-related files
│   ├── Dockerfile           # Production build
│   ├── Dockerfile.dev       # Development build
│   ├── docker-dev.sh        # Docker management script
│   ├── mysql/               # MySQL initialization
│   └── DOCKER.md            # Docker documentation
├── docker-compose.yml       # Docker Compose configuration
└── Makefile                 # Make commands (same as devenv.nix)
```

## Documentation

- [Docker Setup](docker/DOCKER.md) - Complete Docker development guide
- [Migrations](migrations/) - Database migration files

## Requirements

- Go 1.22.1+
- Docker & Docker Compose (for Docker setup)
- Nix with devenv (for devenv setup)

## Environment Variables

See `docker/.env.docker.example` for Docker environment variables.
