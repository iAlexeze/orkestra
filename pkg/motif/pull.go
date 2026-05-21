// pkg/motif/pull.go
//
// PullImport fetches a motif OCI artifact to the local cache without expanding it.
// Used as the pre-pull step before validate, generate, or run commands.
package motif

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/merger"
	"github.com/orkspace/orkestra/pkg/registry"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// PullImport fetches a motif artifact to the local cache.
// Non-OCI refs (file paths, git URLs) are silently skipped — they don't need pulling.
// Resolution mirrors LoadImport exactly, including bare-name → default registry expansion.
func PullImport(imp *orktypes.MotifImport) error {
	ref := strings.TrimSpace(imp.Motif)

	if isFilePath(ref) || isGitURL(ref) {
		return nil
	}

	oci := imp.OCI
	if strings.HasPrefix(ref, "oci://") {
		oci = true
		ref = strings.TrimPrefix(ref, "oci://")
	}

	// Bare name → resolve against default motif registry.
	if !oci && !looksLikeFullRef(ref) {
		resolved, err := registry.ResolveForKind(ref, registry.MotifKind)
		if err != nil {
			return fmt.Errorf("motif %q: resolving reference: %w", imp.Motif, err)
		}
		ref = resolved.Full
		oci = true
	}

	if !oci {
		return nil // full ref without oci: true → not an OCI import
	}

	cleanURL, version := resolveMotifRef(ref, imp.Version, oci)

	auth, err := imp.Auth.Resolve()
	if err != nil {
		return fmt.Errorf("motif %q: auth: %w", imp.Motif, err)
	}

	_, cleanup, err := merger.PullMotifToDir(cleanURL, version, true, auth)
	if err != nil {
		return fmt.Errorf("motif %q@%s: pull failed: %w", cleanURL, version, err)
	}
	cleanup()
	return nil
}
