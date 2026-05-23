# pkg/registry — stability contract

This package is the single implementation of OCI registry logic for the Orkestra toolchain. Both the CLI (`ork registry push/pull/info`) and the Marketplace (`store/memory/oci_client.go`) depend on it. Changes here affect both consumers.

## What is stable

- `Resolve(ociURL string) (Ref, error)` — parses any `oci://` URL into a typed reference. The URL format is `oci://<registry>/<repository>:<tag>`.
- `NewClient() (*Client, error)` — creates an ORAS client. Picks up Docker credential helpers and `~/.docker/config.json` automatically.
- `Client.Info(ctx, ref) (*ArtifactInfo, error)` — fetches manifest annotations without downloading layers. Returns `ArtifactInfo.Meta` which holds name, version, description, author, tags, and E2E result.
- `Client.Pull(ctx, ref, force) (cacheDir string, error)` — pulls all layers to a local cache directory and returns its path. Files can be read from disk after this call.
- `ArtifactMeta` fields: `Name`, `Version`, `Description`, `Author`, `Tags`, `E2E` (`*E2EResult` with `Status string`).

## E2E trust model

`E2EResult.Status` is written by `ork registry push`, not by the publisher manually. The push command detects `e2e.yaml` in the pattern, runs the tests, and bakes the outcome (`passed`, `skipped`, or `failed`) into the OCI annotation before the push completes. The Marketplace reads this annotation and surfaces it as a verified signal — it is not self-reported.

## What is not stable yet

- The exact set of fields in `ArtifactMeta`. New fields may be added; no fields will be removed without a deprecation notice in this file.
- The cache directory layout returned by `Pull`. Treat it as opaque — read files from it but do not rely on its path structure.
- `ArtifactInfo` fields beyond `Meta` (e.g. digest, size). These are implementation details.

## Versioning rules

The Marketplace pins to a specific `github.com/orkspace/orkestra` version in its `go.mod`. Changes to any stable surface above require:

1. A minor or patch version bump in `orkestra`.
2. A corresponding `go get github.com/orkspace/orkestra@<new-version>` + `go mod tidy` in the Marketplace.
3. If `ArtifactMeta` gains new fields used by the Marketplace, update `store/memory/oci_client.go` to map them.

If you are changing this package and the Marketplace depends on what you are changing, update both in the same release cycle.
