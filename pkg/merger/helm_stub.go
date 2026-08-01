// pkg/merger/helm_stub.go
//
// Stand-in for helm.go in the runtime and gateway builds, which exclude the
// Helm SDK entirely (see the comment at the top of helm.go for why). Keeps
// file.go's call site compiling in all three build configurations; a
// production Katalog with an imports.helm: entry gets a clear error instead
// of a missing-symbol build failure.

//go:build runtime || gateway

package merger

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// loadHelmSource is unavailable in this build — Helm-chart Katalog imports
// are an authoring-time feature only. Pre-merge with the ork CLI
// (ork generate bundle) before deploying.
func (m *Merger) loadHelmSource(src orktypes.HelmSource) (map[string]orktypes.CRDEntry, error) {
	return nil, fmt.Errorf("imports.helm (chart %q) is not supported in this build — "+
		"Helm-chart imports are authoring-time only; run 'ork generate bundle' to "+
		"pre-merge before deploying the runtime or gateway", src.Chart)
}
