package reconciler

import (
	"context"
	"os"

	orkexternal "github.com/orkspace/orkestra/pkg/external"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func expandEnv(s string) string {
	return os.ExpandEnv(s)
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
