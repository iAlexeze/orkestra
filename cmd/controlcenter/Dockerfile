# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Full Go toolchain — only used during compilation, never in the final image.
FROM golang:alpine3.23 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build arguments injected by the release workflow via --build-arg
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w \
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

ENTRYPOINT ["/usr/local/bin/orkcc"]
CMD ["--help"]