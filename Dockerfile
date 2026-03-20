# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Full Go toolchain — only used during compilation, never in the final image.
FROM golang:1.25-alpine AS builder

# Install git — needed for go modules that reference git repos
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy dependency files first — Docker cache layer reuse.
# Only re-downloads modules when go.mod or go.sum changes.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build arguments injected by the release workflow via --build-arg
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Build the binary
# CGO_ENABLED=0  — fully static binary, no libc dependency
# -trimpath      — removes local file paths from stack traces
# -ldflags "-s -w" — strip debug info and symbol table (smaller binary)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/iAlexeze/orkestra/pkg/version.Version=${VERSION} \
      -X github.com/iAlexeze/orkestra/pkg/version.Commit=${COMMIT} \
      -X github.com/iAlexeze/orkestra/pkg/version.Date=${BUILD_DATE}" \
    -o /build/ork \
    ./cmd/orkestra/

# Verify the binary is statically linked
RUN file /build/ork && ldd /build/ork 2>&1 || true

# ── Stage 2: Final image ───────────────────────────────────────────────────────
# distroless/static — no shell, no package manager, no libc.
# Attack surface is as small as a container image can be.
# Contains only: the binary, CA certificates, and timezone data.
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CA certs and timezone data from builder
# (distroless/static includes these but being explicit documents the intent)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy only the compiled binary — nothing else from the build stage
COPY --from=builder /build/ork /usr/local/bin/ork

# distroless/nonroot runs as uid 65532 by default
# Declaring it here makes the intent explicit and satisfies security scanners
USER 65532:65532

# Health check — Kubernetes probes handle this, but useful for docker run
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/ork", "version"]

ENTRYPOINT ["/usr/local/bin/ork"]
CMD ["--help"]