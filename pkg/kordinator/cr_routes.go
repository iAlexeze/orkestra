// pkg/kordinator/cr_routes.go
//
// Route registration for CR endpoints.
//
// Call RegisterCRRoutes alongside the existing per-CRD handler registrations
// in pkg/kordinator/server.go (or wherever BuildCRDInfoHandler is called).
//
// The existing registration looks like:
//
//   mux.Handle("/katalog/"+name+"/health", BuildCRDHealthHandler(...))
//   mux.Handle("/katalog/"+name,           BuildCRDInfoHandler(...))
//
// Add after those two lines:
//
//   RegisterCRRoutes(mux, crd, inf, kube, rc)
//
// This registers:
//   /katalog/{crd}/cr                      → CR list
//   /katalog/{crd}/cr/{name}               → CR detail (cluster-scoped)
//   /katalog/{crd}/cr/{namespace}/{name}   → CR detail (namespaced)
//   /katalog/{crd}/cr/{...}/events         → CR events

package kordinator

import (
	"net/http"
	"strings"

	"github.com/ialexeze/orkestra/pkg/kubeclient"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// RegisterCRRoutes adds the CR list, detail, and events handlers to mux.
// Call this alongside BuildCRDHealthHandler and BuildCRDInfoHandler for each CRD.
func RegisterCRRoutes(
	mux *http.ServeMux,
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
	kube *kubeclient.Kubeclient,
	rc orktypes.OperatorBoxConfig,
) {
	name := strings.ToLower(crd.Name)
	crBase := "/katalog/" + name + "/cr"

	// List all CRs — exact match on /cr
	mux.Handle(crBase, BuildCRListHandler(crd, inf))

	// Detail and events — prefix match on /cr/ to capture name segments
	// The detail handler parses the remaining path segments itself.
	mux.Handle(crBase+"/", crDetailRouter(crd, inf, kube, rc))
}

// crDetailRouter is an http.Handler that dispatches between:
//   - /katalog/{crd}/cr/{name}                → detail (cluster-scoped)
//   - /katalog/{crd}/cr/{namespace}/{name}     → detail (namespaced)
//   - /katalog/{crd}/cr/{name}/events          → events
//   - /katalog/{crd}/cr/{namespace}/{name}/events → events
//
// The mux strips the /katalog/{crd}/cr/ prefix via the registered pattern,
// leaving just the path suffix for this handler to parse.
func crDetailRouter(
	crd orktypes.CRDEntry,
	inf cache.SharedIndexInformer,
	kube *kubeclient.Kubeclient,
	rc orktypes.OperatorBoxConfig,
) http.Handler {
	detailHandler := BuildCRDetailHandler(crd, inf, kube, rc)
	eventsHandler := BuildCREventsHandler(crd, kube)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rewrite the URL path so each handler sees only its relevant suffix.
		// Original: /katalog/{crd}/cr/{namespace}/{name}/events
		// After strip: /{namespace}/{name}/events
		prefix := "/katalog/" + strings.ToLower(crd.Name) + "/cr/"
		suffix := strings.TrimPrefix(r.URL.Path, prefix)

		if strings.HasSuffix(suffix, "/events") {
			// Strip /events and forward
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/" + strings.TrimSuffix(suffix, "/events")
			eventsHandler.ServeHTTP(w, r2)
			return
		}

		// Detail endpoint
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + suffix
		detailHandler.ServeHTTP(w, r2)
	})
}
