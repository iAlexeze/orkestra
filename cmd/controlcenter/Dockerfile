# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:alpine3.23 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files from repository root
COPY go.mod go.sum ./

# Set Go proxy for faster downloads
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Download dependencies with retry
RUN for i in 1 2 3; do \
      go mod download && break || sleep 10; \
    done

# Copy the entire source (we need the control center code and embedded assets)
COPY . .

# Build arguments
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

# Build with limited parallelism
RUN CGO_ENABLED=0 go build \
    -p=2 \
    -trimpath \
    -gcflags="all=-l=4" \
    -ldflags="-s -w -extldflags '-static' \
      -X github.com/iAlexeze/orkestra/pkg/version.Version=${VERSION} \
      -X github.com/iAlexeze/orkestra/pkg/version.Commit=${COMMIT} \
      -X github.com/iAlexeze/orkestra/pkg/version.Date=${BUILD_DATE}" \
    -o /orkcc .

# ── Stage 2: Final image ───────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /orkcc /usr/local/bin/orkcc

USER 1001:1001

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/usr/local/bin/orkcc", "--help"]

ENTRYPOINT ["/usr/local/bin/orkcc"]
CMD ["--help"]