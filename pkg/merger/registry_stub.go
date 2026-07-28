// pkg/merger/registry_stub.go
//
// Stand-in for registry.go in the runtime and gateway builds. See the
// comment at the top of registry.go for why.

//go:build runtime || gateway

package merger

import (
	"fmt"

	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// loadRegistrySource is unavailable in this build — Registry Katalog
// imports are an authoring-time feature only. Pre-merge with the ork CLI
// (ork generate bundle) before deploying.
func (m *Merger) loadRegistrySource(src orktypes.RegistrySource) (map[string]orktypes.CRDEntry, error) {
	url, _ := src.ResolvedURL()
	return nil, fmt.Errorf("imports.registry (%q) is not supported in this build — "+
		"registry imports are authoring-time only; run 'ork generate bundle' to "+
		"pre-merge before deploying the runtime or gateway", url)
}
