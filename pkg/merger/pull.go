// pkg/merger/pull.go
//
// Exported pull helpers — shared with pkg/motif and any other package
// that needs to fetch a registry artifact without loading a full Katalog.
//
// Both PullToDir and PullMotifToDir follow a cache-first strategy for OCI
// artifacts: ~/.orkestra/registry/<host>/<repo>/<version>/ is checked for a
// sentinel file (katalog.yaml or motif.yaml) before any network call is made.
// This avoids redundant pulls and removes the need for Docker credential
// forwarding inside the process — callers should use `ork pull`
// to populate the cache, then rely on these helpers to read from it.
package merger

import (
	"fmt"
	"os"

	"github.com/orkspace/orkestra/pkg/registry"
	"github.com/orkspace/orkestra/pkg/utils"
)

// noop cleanup used when serving from cache — nothing to remove.
var noopCleanup = func() {}

// PullToDir returns the directory for a registry pattern artifact (OCI or Git).
// For OCI artifacts it checks the local cache (~/.orkestra/registry/) first and
// returns the cached directory without a network call when available.
//
// Returns (dir, cleanup, err). Always call cleanup() when done —
// for cached hits cleanup is a no-op; for fresh pulls it removes the temp dir.
func PullToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
	if oci {
		if cached, ok := registry.CachedDir(url, version); ok {
			return cached, noopCleanup, nil
		}
	}
	m := &Merger{}
	return m.pullPattern(url, version, oci, auth)
}

// PullMotifToDir returns the directory for a motif artifact (OCI or Git).
// For OCI artifacts it checks the local cache (~/.orkestra/registry/) first and
// returns the cached directory without a network call when available.
//
// For OCI: the entire OCI artifact is pulled — motif.yaml must be at the root.
// For Git (GitHub/GitLab): only motif.yaml is fetched via raw URL.
// For generic Git: the repo is cloned and motif.yaml is copied.
//
// Returns (dir, cleanup, err). Always call cleanup() when done.
func PullMotifToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
	if oci {
		if cached, ok := registry.CachedDir(url, version); ok {
			return cached, noopCleanup, nil
		}
	}

	tmpDir, err := os.MkdirTemp("", "orkestra-motif-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { os.RemoveAll(tmpDir) }

	m := &Merger{}
	if oci {
		err = m.pullOCIPattern(url, version, tmpDir, auth)
	} else {
		err = m.pullMotifFromGit(url, version, tmpDir, auth)
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpDir, cleanup, nil
}
