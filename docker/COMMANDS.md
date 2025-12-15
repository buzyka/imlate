# Command Reference

All Docker commands use the same names as devenv.nix for consistency.

## Development Commands (Identical to devenv.nix)

| Command | Description |
|---------|-------------|
| `make install-mod` | Install Go modules (`go mod download`) |
| `make build-app` | Build the application |
| `make start-app` | Start the application |
| `make got` | Run tests (`go test ./...`) |
| `make gotc` | Run tests with coverage |
| `make gol` | Run golangci-lint |
| `make golf` | Run golangci-lint with auto-fix |

## Environment Management

| Command | Description |
|---------|-------------|
| `make start` | Start development environment |
| `make stop` | Stop development environment |
| `make restart` | Restart development environment |
| `make status` | Show container status |
| `make logs` | View all logs |
| `make logs-app` | View application logs |
| `make logs-mysql` | View MySQL logs |

## Database Management

| Command | Description |
|---------|-------------|
| `make migrate-up` | Apply database migrations |
| `make migrate-down` | Rollback database migrations |
| `make mysql-shell` | Access MySQL shell |

## Maintenance

| Command | Description |
|---------|-------------|
| `make rebuild` | Rebuild containers from scratch |
| `make clean` | Remove all containers and volumes |
| `make shell` | Open bash shell in app container |

## Direct Script Usage

You can also use the script directly:

```bash
./docker/docker-dev.sh <command>
```

All commands are available through both `make` and the script.
