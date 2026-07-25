package reconciler

import (
	"context"
	"os"

	orkexternal "github.com/orkspace/orkestra/pkg/external"
	orktmpl "github.com/orkspace/orkestra/pkg/resources/template"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func runExternal(
	ctx context.Context,
	gvk string,
	resolver *orktmpl.Resolver,
	calls []orktypes.ExternalCallSpec,
) (*orktmpl.Resolver, error) {
	return orkexternal.Run(ctx, gvk, resolver, calls)
}

func expandEnv(s string) string {
	return os.ExpandEnv(s)
}
