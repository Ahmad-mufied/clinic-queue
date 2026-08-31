# ==============================================================================
# SMART CLINIC QUEUE - MULTI-STAGE DOCKERFILE
# Target: Northflank / Container Cloud / VPS
# ==============================================================================

# Stage 1: Build binary with Go toolchain
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Install git and ca-certificates for downloading dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build standalone binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.21

WORKDIR /app

# Install CA certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root application user for container security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy compiled backend binary
COPY --from=builder /app/bin/api /app/api

# Copy Casbin RBAC configuration policies
COPY --from=builder /app/config /app/config

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

USER appuser

# Expose default HTTP API port
EXPOSE 8080

# Run API server
ENTRYPOINT ["/app/api"]
