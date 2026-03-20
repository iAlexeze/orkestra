// pkg/merger/helper.go
package merger

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ialexeze/orkestra/pkg/logger"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ── Internal helpers ──────────────────────────────────────────────────────────

func (m *Merger) mustBeMerged() {
	if !m.merged {
		panic("merger: call Merge() before querying")
	}
}

// checkDuplicate returns an error if name is already in seen from a different source.
func checkDuplicate(seen map[string]string, name, source string) error {
	if existing, ok := seen[name]; ok && existing != source {
		return fmt.Errorf(
			"duplicate CRD %q: defined in %q and %q — names must be unique across all sources",
			name, existing, source,
		)
	}
	return nil
}

// removeCRD removes a CRD by name from the list (for inline override).
func removeCRD(crds []orktypes.CRDEntry, name string) []orktypes.CRDEntry {
	out := crds[:0]
	for _, crd := range crds {
		if crd.Name != name {
			logger.Debug().Msgf("overriding with inline declaration for %s", crd.Name)
			out = append(out, crd)
		}
	}
	return out
}

// resolveEnvVar replaces $VAR_NAME with its environment variable value.
func resolveEnvVar(s string) (string, error) {
	if !strings.HasPrefix(s, "$") {
		return s, nil
	}
	varName := strings.TrimPrefix(s, "$")
	val := os.Getenv(varName)
	if val == "" {
		return "", fmt.Errorf("env var %q is not set or empty", varName)
	}
	return val, nil
}

// writeTempFile writes data to a temp file and returns the path.
func writeTempFile(data []byte, pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	return f.Name(), nil
}

// unused but kept for completeness
var _ = bytes.NewBuffer
