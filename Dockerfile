# Build stage
FROM golang:1.26-alpine AS builder

# Version argument for build-time injection
ARG VERSION=docker

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server and CLI with version
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X github.com/mtzanidakis/budgeting/internal/version.Version=${VERSION}" \
    -o /app/server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X github.com/mtzanidakis/budgeting/internal/version.Version=${VERSION}" \
    -o /app/admin ./cmd/admin

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

# Copy binaries from builder
COPY --from=builder /app/server /app/server
COPY --from=builder /app/admin /app/admin

# Create data directory
RUN mkdir -p /data

# Run the server
CMD ["/app/server"]
