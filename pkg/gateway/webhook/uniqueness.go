// pkg/gateway/webhook/uniqueness.go
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/orkspace/orkestra/pkg/runtime/kordinator"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

const runtimeUniquenessTimeout = 3 * time.Second

// runtimeUniquenessChecker implements orktypes.UniquenessChecker at
// admission time by querying the runtime's own GET /katalog/{crd}/cr?field=
// endpoint. The runtime already holds every instance of the CRD in its
// informer cache — this is a fast in-memory lookup on the runtime side, not
// a live List() against the API server. That's deliberately different from
// the reconciler's own checker (pkg/runtime/reconciler/uniqueness.go), which
// uses a live call specifically because a stale cache could let two
// concurrent duplicates both pass — reconcile time is where that guarantee
// actually needs to hold.
//
// Here, a stale informer cache is an acceptable trade: reconcile time
// remains the authoritative, live-checked enforcement point regardless of
// what admission sees. Worst case on a race or a runtime read that's a
// moment behind: a duplicate slips past admission and is caught on the very
// next reconcile — exactly the behavior every katalog already had before
// this existed. This is a fast, best-effort early rejection layered on top
// of a guarantee that doesn't depend on it.
type runtimeUniquenessChecker struct {
	ctx      context.Context
	client   *http.Client
	endpoint string // e.g. http://orkestra-runtime.orkestra-system.svc:8080
	crdName  string // the katalog key (spec.crds.<key>), matches {crd} in /katalog/{crd}/cr
}

// newRuntimeUniquenessChecker builds a checker targeting the given runtime
// endpoint for one CRD. Constructing it does no I/O — the HTTP call only
// happens inside IsUnique, itself only reached when a validation.rules entry
// actually uses operator: unique.
func newRuntimeUniquenessChecker(ctx context.Context, endpoint, crdName string) orktypes.UniquenessChecker {
	return &runtimeUniquenessChecker{
		ctx:      ctx,
		client:   &http.Client{Timeout: runtimeUniquenessTimeout},
		endpoint: endpoint,
		crdName:  crdName,
	}
}

// IsUnique lists every existing instance of the CRD via the runtime's cache
// and reports whether none of them (other than the CR under evaluation
// itself) has field == value.
func (c *runtimeUniquenessChecker) IsUnique(field, value, selfNamespace, selfName string) (bool, error) {
	reqURL := fmt.Sprintf("%s/katalog/%s/cr?field=%s", c.endpoint, c.crdName, url.QueryEscape(field))
	req, err := http.NewRequestWithContext(c.ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("building uniqueness request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("querying runtime for uniqueness: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("runtime returned %d checking uniqueness", resp.StatusCode)
	}

	var list kordinator.CRListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return false, fmt.Errorf("decoding uniqueness response: %w", err)
	}

	for _, item := range list.Items {
		if item.Namespace == selfNamespace && item.Name == selfName {
			continue // never a duplicate of its own stored value
		}
		if item.Value == value {
			return false, nil
		}
	}
	return true, nil
}

// runtimeEndpoint builds the in-cluster URL for this Orkestra instance's own
// runtime — the same service the gateway is deployed alongside, not an
// operator-declared cross: endpoint. Port comes from konfig's ORK_PORT
// (default "8080") — the runtime and gateway both read the same env var
// convention for their own /katalog server, so the gateway's own configured
// value is the runtime's too.
func (ws *WebhookServer) runtimeEndpoint() string {
	svc := ws.katalog.RuntimeServiceName()
	ns := ws.konfig.Cluster().Namespace()
	port := ws.konfig.Health().Port()
	return fmt.Sprintf("http://%s.%s.svc:%s", svc, ns, port)
}

// crdNameForKind returns the katalog key (spec.crds.<key>) for a CRD whose
// apiTypes.kind matches kind — the reverse of the usual CRDEntry lookup,
// needed because admission only carries the Kind, but /katalog/{crd}/cr
// addresses CRDs by their katalog key. Empty when no enabled CRD matches.
func (ws *WebhookServer) crdNameForKind(kind string) string {
	for name, entry := range ws.katalog.EnabledCRDs() {
		if entry.APITypes.Kind == kind {
			return name
		}
	}
	return ""
}
