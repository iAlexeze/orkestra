package api

import (
	"context"
	"net/http"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

func WithTokenName(ctx context.Context, name string) context.Context {
	return contextWithTokenName(ctx, name)
}

func ParsePath(path string) (kind, ns, name string, err error) {
	return parsePath(path)
}

func IsTargetRequest(raw map[string]interface{}) bool {
	return isTargetRequest(raw)
}

func ExportedSchemaHandler(kat *katalog.Katalog) http.Handler {
	return schemaHandler(kat)
}

func ExportedResourcesHandler(kube kubeclient.KubeClient, kat *katalog.Katalog, notes orktypes.NoteRegistry) http.Handler {
	return resourcesHandler(kube, kat, notes)
}

func ExportedApplyHandler(kube kubeclient.KubeClient, kat *katalog.Katalog, notes orktypes.NoteRegistry) http.Handler {
	return applyHandler(kube, kat, notes)
}

func ExportedCheckServePermission(w http.ResponseWriter, r *http.Request, crd *orktypes.CRDEntry, class orktypes.ServeEndpointClass, op, ns, alias string) bool {
	return checkServePermission(w, r, crd, class, op, ns, alias)
}

func ExportedWriteKubeError(w http.ResponseWriter, err error) {
	writeKubeError(w, err)
}
