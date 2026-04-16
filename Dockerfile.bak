# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:alpine3.23 AS builder

# Install git and build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy dependency files first — Docker cache layer reuse
COPY go.mod go.sum ./

# Set Go proxy for faster downloads in CI
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Download dependencies with retry
RUN for i in 1 2 3; do \
      go mod download && break || sleep 10; \
    done

# Copy the rest of the source
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Build with limited parallelism to avoid OOM
RUN CGO_ENABLED=0 GOOS=linux go build \
    -p=2 \
    -trimpath \
    -gcflags="all=-l=4" \
    -ldflags="-s -w -extldflags '-static' \
      -X github.com/iAlexeze/orkestra/pkg/version.Version=${VERSION} \
      -X github.com/iAlexeze/orkestra/pkg/version.Commit=${COMMIT} \
      -X github.com/iAlexeze/orkestra/pkg/version.Date=${BUILD_DATE}" \
    -o /build/ork \
    ./cmd/orkestra/

# Verify the binary
RUN file /build/ork && ldd /build/ork 2>&1 || echo "Statically linked"

# ── Stage 2: Final image ───────────────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /build/ork /usr/local/bin/ork

USER 65532:65532

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/ork", "version"]

ENTRYPOINT ["/usr/local/bin/ork"]
CMD ["--help"]