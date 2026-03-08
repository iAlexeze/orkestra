# ---- Build Stage ----
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first (for better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the controller
# - CGO_ENABLED=0 for static binary
# - ldflags to strip debug info and reduce size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o bin/multi-crd-controller ./cmd

# ---- Final Stage ----
FROM alpine:3.19

# Install ca-certificates for TLS and curl for health checks
RUN apk add --no-cache ca-certificates curl jq

# Create non-root user for security
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/multi-crd-controller /app/multi-crd-controller

# Copy .env.example as reference (optional)
COPY --from=builder /app/.env.example /app/.env.example

# Create directory for potential volume mounts
RUN mkdir -p /app/config && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Run the controller
ENTRYPOINT ["/app/multi-crd-controller"]