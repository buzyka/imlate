# Build stage - Debian 12 (Bookworm)
FROM golang:1.27-bookworm AS builder

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    gcc \
    g++ \
    libc6-dev \
    libsqlite3-dev \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build the CLI tools shipped in the image (swag, migrate). Their versions come
# from tools/go.mod, which Dependabot keeps up to date.
COPY tools/go.mod tools/go.sum /tools/
RUN cd /tools && GOBIN=/tools/bin go install -tags 'mysql' \
    github.com/swaggo/swag/cmd/swag \
    github.com/golang-migrate/migrate/v4/cmd/migrate

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG APP_VERSION=2.0.x-dev
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X github.com/buzyka/imlate/internal/version.Version=${APP_VERSION}" \
    -o tracker cmd/app/main.go

# Runtime stage - Debian 12 slim
FROM debian:bookworm-slim

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    default-mysql-client \
    bash \
    && rm -rf /var/lib/apt/lists/*

# Install Go (minimal for tools)
COPY --from=builder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/go/bin:${PATH}"
ENV GOPATH="/go"

# CLI tools built in the builder stage from tools/go.mod
COPY --from=builder /tools/bin/ /go/bin/

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/tracker .

# Copy necessary files
COPY migrations ./migrations
COPY website ./website
COPY storage ./storage

# Create storage subdirectories for dynamic data
RUN mkdir -p storage/img/visitors storage/img/students storage/output

# Expose port
EXPOSE 8080

CMD ["./tracker"]
