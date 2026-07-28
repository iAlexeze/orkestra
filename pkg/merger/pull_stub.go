// pkg/merger/pull_stub.go
//
// Stand-in for pull.go in the runtime and gateway builds. See the comment
// at the top of pull.go for why. PullToDir/PullMotifToDir are called by
// pkg/registry/motif (motif import resolution) — another authoring-time-only
// path the runtime and gateway never exercise, since they only ever read
// the katalog.yaml key from an already-merged ConfigMap.

//go:build runtime || gateway

package merger

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/utils"
)

func PullToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
	return "", nil, fmt.Errorf("pulling registry pattern %q@%s is not supported in this build — "+
		"registry/motif imports are authoring-time only; run 'ork generate bundle' to "+
		"pre-merge before deploying the runtime or gateway", url, version)
}

func PullMotifToDir(url, version string, oci bool, auth *utils.FileAuth) (dir string, cleanup func(), err error) {
	return "", nil, fmt.Errorf("pulling motif %q@%s is not supported in this build — "+
		"registry/motif imports are authoring-time only; run 'ork generate bundle' to "+
		"pre-merge before deploying the runtime or gateway", url, version)
}
