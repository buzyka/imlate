#!/usr/bin/env bash

# Docker Development Environment Management Script
# This script provides the commands for the Docker-based development environment

set -e

PROJECT_NAME="imlate"
COMPOSE_FILE="docker-compose.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if Docker is running
check_docker() {
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Start the development environment
start() {
    print_info "Starting development environment..."
    check_docker
    docker-compose -f $COMPOSE_FILE up -d
    print_success "Development environment started"
    print_info "Application: http://localhost:8080"
    print_info "MySQL: localhost:3307"
}

# Stop the development environment
stop() {
    print_info "Stopping development environment..."
    docker-compose -f $COMPOSE_FILE down
    print_success "Development environment stopped"
}

# Restart the development environment
restart() {
    print_info "Restarting development environment..."
    stop
    start
}

# View logs
logs() {
    if [ -z "$1" ]; then
        docker-compose -f $COMPOSE_FILE logs -f
    else
        docker-compose -f $COMPOSE_FILE logs -f "$1"
    fi
}

# Execute a command in the app container
exec_app() {
    docker-compose -f $COMPOSE_FILE exec app "$@"
}

# Install Go modules
install_mod() {
    print_info "Installing Go modules..."
    exec_app go mod download
    print_success "Go modules installed"
}

# Build the application
build_app() {
    print_info "Building application..."
    local ldflags=""
    if [ -n "${APP_VERSION:-}" ]; then
        ldflags="-X github.com/buzyka/imlate/internal/version.Version=${APP_VERSION}"
    fi
    exec_app go build -ldflags "${ldflags}" -o tracker cmd/app/main.go
    print_success "Application built successfully"
}

# Build distribution bundle
dist() {
    local target_goos="${1:-linux}"
    local target_goarch="${2:-amd64}"
    local go_bin="/usr/local/go/bin/go"

    local ldflags=""
    if [ -n "${APP_VERSION:-}" ]; then
        ldflags="-X github.com/buzyka/imlate/internal/version.Version=${APP_VERSION}"
    fi

    print_info "Building distribution for ${target_goos}/${target_goarch}..."
    check_docker
    exec_app bash -lc "mkdir -p /app/dist && rm -rf /app/dist/* && GOOS=${target_goos} GOARCH=${target_goarch} ${go_bin} build -ldflags '${ldflags}' -o /app/dist/app cmd/app/main.go && chmod 755 /app/dist/app && cp -R /app/migrations /app/dist/migrations && cp -R /app/website /app/dist/website"
    print_success "Distribution bundle created in dist/"
}

# Run golangci-lint
gol() {
    print_info "Running golangci-lint..."
    exec_app golangci-lint run
}

# Run golangci-lint with fix
golf() {
    print_info "Running golangci-lint with fix..."
    exec_app golangci-lint run --fix
    print_success "Linting completed with fixes applied"
}

# Run tests
got() {
    print_info "Running tests..."
    exec_app go test ./...
}

# Run tests with coverage
gotc() {
    print_info "Running tests with coverage..."
    exec_app bash -c "go test ./... -coverprofile=coverage-report.out && go tool cover -html=coverage-report.out -o coverage-report.html"
    print_success "Coverage report generated: coverage-report.html"
}

# Run database migrations
migrate_up() {
    print_info "Running database migrations..."
    exec_app migrate -database "mysql://trackme:trackme@tcp(mysql:3306)/tracker" -path /app/migrations up
    print_success "Migrations applied successfully"
}

# Rollback database migrations
migrate_down() {
    print_info "Rolling back database migrations..."
    exec_app migrate -database "mysql://trackme:trackme@tcp(mysql:3306)/tracker" -path /app/migrations down
    print_success "Migrations rolled back successfully"
}

# Create a new migration
migrate_create() {
    if [ -z "$1" ]; then
        print_error "Migration name is required"
        echo "Usage: $0 migrate-create <migration_name>"
        exit 1
    fi
    print_info "Creating migration: $1"
    exec_app migrate create -ext sql -dir /app/migrations -seq "$1"
    print_success "Migration files created"
}

# Access MySQL shell
mysql_shell() {
    print_info "Connecting to MySQL shell..."
    docker-compose -f $COMPOSE_FILE exec mysql mysql -u trackme -ptrackme tracker
}

# Show status of containers
status() {
    docker-compose -f $COMPOSE_FILE ps
}

# Clean up everything (including volumes)
clean() {
    print_info "Cleaning up all containers and volumes..."
    docker-compose -f $COMPOSE_FILE down -v
    print_success "Cleanup completed"
}

# Rebuild containers
rebuild() {
    print_info "Rebuilding containers..."
    docker-compose -f $COMPOSE_FILE build --no-cache
    print_success "Containers rebuilt"
}

# Restart app container only
restart_app() {
    print_info "Restarting app container..."
    docker-compose -f $COMPOSE_FILE restart app
    print_success "App container restarted"
}

# Run the application
run_app() {
    print_info "Running application..."
    exec_app go run cmd/app/main.go
}

# Update admin UI from GitHub release
update_ui() {
    local ver="$1"
    if [ -z "$ver" ]; then
        print_error "Version is required. Usage: make update-ui version=0.1.1"
        exit 1
    fi

    local url="https://github.com/buzyka/imlate-front/releases/download/v${ver}/imlate-ui-v${ver}.zip"
    local zip_file="/tmp/imlate-ui-v${ver}.zip"
    local admin_dir="website/admin"
    local backup_dir="website/admin.bak"

    print_info "Downloading UI v${ver} from ${url}..."
    local http_code
    http_code=$(curl -sL -w "%{http_code}" -o "$zip_file" "$url")
    if [ "$http_code" != "200" ]; then
        rm -f "$zip_file"
        print_error "Failed to download: HTTP ${http_code}. Check that version v${ver} exists at ${url}"
        exit 1
    fi

    if [ -d "$admin_dir" ]; then
        rm -rf "$backup_dir"
        mv "$admin_dir" "$backup_dir"
    fi

    if ! unzip -q "$zip_file" -d "website"; then
        print_error "Failed to unzip archive. Restoring previous version..."
        rm -rf "$admin_dir"
        if [ -d "$backup_dir" ]; then
            mv "$backup_dir" "$admin_dir"
        fi
        rm -f "$zip_file"
        exit 1
    fi

    rm -rf "$backup_dir"
    rm -f "$zip_file"
    print_success "UI updated to v${ver}"
}

# Show help
show_help() {
    cat << EOF
Docker Development Environment Management

Usage: $0 <command> [options]

Commands:
  start              Start the development environment
  stop               Stop the development environment
  restart            Restart the development environment
  status             Show status of containers
  logs [service]     View logs (optionally for specific service: app, mysql)
  
  # Build & Run
  install-mod        Install Go modules
  build-app          Build the application
  dist [GOOS GOARCH] Build distribution bundle
  run-app            Run the application
  
  # Testing & Linting
  got                Run tests
  gotc               Run tests with coverage
  gol                Run golangci-lint
  golf               Run golangci-lint with auto-fix
  
  # Database
  migrate-up         Run database migrations
  migrate-down       Rollback database migrations
  migrate-create     Create new migration file
  mysql-shell        Access MySQL shell
  
  # Maintenance
  rebuild            Rebuild containers from scratch
  clean              Stop and remove all containers and volumes
  
  # UI
  update-ui <ver>    Download and install admin UI from GitHub release

  # Utilities
  exec <command>     Execute a command in the app container
  help               Show this help message

Examples:
  $0 start           # Start the development environment
  $0 got             # Run tests
  $0 dist            # Build dist/app for linux/amd64
  $0 logs app        # View application logs
  $0 exec bash       # Open a bash shell in the app container

EOF
}

# Main command router
case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs "$2"
        ;;
    install-mod)
        install_mod
        ;;
    build-app)
        build_app
        ;;
    dist)
        dist "$2" "$3"
        ;;
    run-app)
        run_app
        ;;
    gol)
        gol
        ;;
    golf)
        golf
        ;;
    got)
        got
        ;;
    gotc)
        gotc
        ;;
    migrate-up)
        migrate_up
        ;;
    migrate-down)
        migrate_down
        ;;
    migrate-create)
        migrate_create "$2"
        ;;
    mysql-shell)
        mysql_shell
        ;;
    rebuild)
        rebuild
        ;;
    restart-app)
        restart_app
        ;;
    update-ui)
        update_ui "$2"
        ;;
    clean)
        clean
        ;;
    exec)
        shift
        exec_app "$@"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
