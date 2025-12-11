# Build stage - Debian 12 (Bookworm)
FROM golang:1.25-bookworm AS builder

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

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o tracker cmd/app/main.go

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

# Install Go tools with pinned versions
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.1 && \
    rm -rf /go/pkg /go/src

WORKDIR /app

# Copy built binary from builder
COPY --from=builder /app/tracker .

# Copy necessary files
COPY migrations ./migrations
COPY website ./website
COPY storage ./storage

# Create output directory
RUN mkdir -p output

# Expose port
EXPOSE 8080

CMD ["./tracker"]
