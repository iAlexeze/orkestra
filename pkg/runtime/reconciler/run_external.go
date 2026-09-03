package reconciler

import (
	"context"

	orkexternal "github.com/orkspace/orkestra/pkg/external"
	orktmpl "github.com/orkspace/orkestra/pkg/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/orkspace/orkestra/pkg/utils"
	"k8s.io/client-go/kubernetes"
)

func expandEnv(s string) (string, error) {
	return utils.ResolveEnvVar(s)
}

func runExternal(
	ctx context.Context,
	gvk string,
	resolver *orktmpl.Resolver,
	calls []orktypes.ExternalCallSpec,
	cs kubernetes.Interface,
) (*orktmpl.Resolver, error) {
	return orkexternal.Run(ctx, gvk, resolver, calls, cs)
}
