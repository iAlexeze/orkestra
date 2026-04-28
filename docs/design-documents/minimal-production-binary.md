# Design Document: Minimal Production Binary for Orkestra (Build‑Tag Approach)

## Problem

The production Docker image uses the same binary that developers use – a full `ork` CLI containing `init`, `generate`, `validate`, `template`, `diff`, `upgrade`, `controlcenter`, etc. The runtime only needs the `run` command. The extra commands bloat the binary and increase attack surface.

## Solution

Use **Go build tags** to compile a slim binary (`ork-runtime`) that contains **only** `run` and its dependencies, while the development binary (`ork`) remains full‑featured.

### Key Points

- The Dockerfile stays **unchanged** – it copies a binary from the build stage.
- The CI pipeline decides which binary to build:
  - **Development / release artifacts** → build full binary (no tags).
  - **Production image** → build slim binary with `-tags runtime`.
- The Docker image can still be tagged as `ork` – it’s just a different binary.

## Implementation

### 1. Add build constraints to non‑runtime commands

In each CLI file that is **not** required for `ork run` (e.g., `init.go`, `generate_*.go`, `validate.go`, `template.go`, `diff.go`, `upgrade.go`, `controlcenter.go`), add at the top:

```go
//go:build !runtime

package cli
```

For the `run.go` command and `version.go` (optional), either omit the constraint or use:

```go
//go:build !runtime || runtime   // always compiled
```

### 2. Modify the CI build step for production

In your workflow, when building for the Docker image, use the `runtime` tag:

```yaml
- name: Build Linux binaries (for runtime image)
  run: |
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
      -tags runtime \
      -trimpath \
      -ldflags="..." \
      -o ork-amd64 ./cmd/orkestra

    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
      -tags runtime \
      -trimpath \
      -ldflags="..." \
      -o ork-arm64 ./cmd/orkestra
```

The resulting binary will only include `run` (and `version`) – no `init`, `generate`, etc.

### 3. Keep the development build unchanged

For local development, release tarballs, or any other use where the full CLI is needed, build **without** the `runtime` tag.

### 4. Dockerfile remains the same

```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
ARG TARGETARCH
COPY ork-${TARGETARCH} /usr/local/bin/ork
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/ork"]
```

The binary copied is the slim one (built with `-tags runtime`).

## Benefits

- **Smaller production image** – binary size reduces from ~60 MB to ~20 MB.
- **Lower attack surface** – no `init`, `generate`, `controlcenter` code in runtime.
- **No code duplication** – one codebase, two builds.
- **Zero changes to user experience** – users still run `ork` (the slim binary) in production.

## Drawbacks

- Must remember to use `-tags runtime` in the CI for Docker builds.
- Developers need to understand the tag, but they don’t have to use it locally.

## Implementation Order

1. Add `//go:build !runtime` to all non‑runtime CLI files.
2. Update the GitHub workflow’s binary build step to include `-tags runtime` for the Docker job.
3. Test that `ork run` works correctly in the slim binary (it should).
4. Optionally add a `make runtime` target for convenience.

## Conclusion

This approach leverages Go’s built‑in conditional compilation to create a minimal production binary without changing the existing codebase structure. It is simple, effective, and keeps the Dockerfile untouched – exactly as your current setup demands.