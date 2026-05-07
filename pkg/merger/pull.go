// pkg/merger/pull.go
//
// Exported pull helpers — shared with pkg/motif and any other package
// that needs to fetch a registry artifact without loading a full Katalog.
package merger

import (
	"fmt"
	"os"

	"github.com/orkspace/orkestra/pkg/utils"
)

// PullToDir fetches a registry pattern artifact (OCI or Git) to a fresh temp directory.
// It follows the same URL@version shorthand and OCI/Git dispatch as RegistrySource.
//
// Unlike loadRegistrySource, this function does NOT validate artifact structure —
// the caller decides which files to read after the pull succeeds.
//
// Patterns may include an optional motif.yaml when they expose a reusable Motif.
// Use PullMotifToDir for standalone Motif repos (no pattern files required).
//
// Returns (dir, cleanup, err). Always call cleanup() when done.
func PullToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
	m := &Merger{}
	return m.pullPattern(url, version, oci, auth)
}

// PullMotifToDir fetches only motif.yaml from a registry to a temp directory.
// Use this for standalone Motif repos (not full patterns).
//
// For OCI: the entire OCI artifact is pulled — motif.yaml must be at the root.
// For Git (GitHub/GitLab): only motif.yaml is fetched via raw URL.
// For generic Git: the repo is cloned and motif.yaml is copied.
//
// Returns (dir, cleanup, err). Always call cleanup() when done.
func PullMotifToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
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
