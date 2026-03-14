package kontroller

import (
	"net/http"
	"strings"

	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/client-go/tools/cache"
)

// CRD Handlers
//
// HealthHandler
func BuildCRDHealthHandler(name string, h *CRDHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		message := name + " healthy"

		if !h.IsHealthy() {
			status = http.StatusServiceUnavailable
			message = name + " degraded"
		}

		utils.WriteJSON(w, status, map[string]interface{}{
			"name":             name,
			"healthy":          h.IsHealthy(),
			"message":          message,
			"errorRate":        h.ErrorRate(),
			"consecutiveFails": h.consecutiveFails.Load(),
			"totalReconciles":  h.totalReconciles.Load(),
			"lastError":        h.lastError.Load(),
			"lastReconcile":    h.lastReconcile.Load(),
		})
	}
}

// InfoHandler
func BuildCRDInfoHandler(
	crd initialize.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
	health *CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		v := resolveCRDDisplayValues(crd, kfg, inf)
		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"name":          crd.Name,
			"gvk":           utils.SetGroupVersionKindObj(crd.GroupVersionKind),
			"resourceCount": v.resourceCount,
			"workers":       v.workers,
			"workersSource": v.workersSource,
			"queueDepth":    v.queueDepth,
			"resync":        v.resync,
			"resyncSource":  v.resyncSource,
			"dependsOn":     crd.DependsOn,
			"healthy":       health.IsHealthy(),
			"errorRate":     health.ErrorRate(),
		})
	}
}

// Katalog Handler
func BuildKatalogHandler(
	katalog *katalog.Katalog,
	kfg *konfig.Konfig,
	reg *ResourceKatalog,
	healthMap map[string]*CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := make([]map[string]interface{}, 0)

		for _, crd := range katalog.Enabled() {
			gvk := utils.SetGroupVersionKindObj(crd.GroupVersionKind)
			h := healthMap[gvk]

			// Get informer from registry
			entry, ok := reg.Get(gvk)
			var inf cache.SharedIndexInformer
			if ok {
				inf = entry.Informer
			}

			v := resolveCRDDisplayValues(crd, kfg, inf)

			crds = append(crds, map[string]interface{}{
				"name":          crd.Name,
				"gvk":           gvk,
				"resourceCount": v.resourceCount,
				"workers":       v.workers,
				"workersSource": v.workersSource,
				"queueDepth":    v.queueDepth,
				"resync":        v.resync,
				"resyncSource":  v.resyncSource,
				"dependsOn":     crd.DependsOn,
				"healthy":       h.IsHealthy(),
				"errorRate":     h.ErrorRate(),
				"endpoints": map[string]string{
					"health": "/katalog/" + strings.ToLower(crd.Name) + "/health",
					"info":   "/katalog/" + strings.ToLower(crd.Name),
				},
			})
		}

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"crds":  crds,
			"total": len(crds),
		})
	}
}

/* The resulting endpoints you get automatically for every enabled CRD in the Katalog:
GET /health              → controller-level (aggregate of all CRDs)
GET /ready               → readyz
GET /metrics             → Prometheus
GET /katalog             → all CRDs with health summary
GET /katalog/project     → Project config + live health state
GET /katalog/project/health       → 200 or 503
GET /katalog/managednamespace     → ManagedNamespace config + health
GET /katalog/managednamespace/health  → 200 or 503
*/

// Helper
type crdDisplayValues struct {
	resync        string // resync: ← came from explicitly set in Katalog
	resyncSource  string // resyncSource: ← came from Orkestra default
	workers       int    // workers: ← came from explicitly set in Katalog
	workersSource string // workersSource: "default"      ← came from Orkestra default
	queueDepth    int
	resourceCount int // number of custom resources managed by the CRD
}

func resolveCRDDisplayValues(crd initialize.CRDEntry, kfg *konfig.Konfig, inf cache.SharedIndexInformer) crdDisplayValues {
	queueDepth := crd.Queue.MaxQueueDepth
	if queueDepth == 0 {
		queueDepth = kfg.Katalog().DefaultMaxQueueDepth
	}

	resync := crd.Resync.String()
	resyncSource := "configured"
	if crd.Resync == 0 {
		resyncSource = "default"
		resync = kfg.Cluster().DefaultResync.String()
	}

	workers := crd.Workers
	workersSource := "configured"
	if crd.Workers == 0 {
		workers = kfg.Cluster().DefaultWorkers
		workersSource = "default"
	}

	// Read directly from the informer
	resourceCount := 0
	if inf != nil {
		resourceCount = len(inf.GetStore().List())
	}

	return crdDisplayValues{
		resync:        resync,
		resyncSource:  resyncSource,
		workers:       workers,
		workersSource: workersSource,
		queueDepth:    queueDepth,
		resourceCount: resourceCount,
	}
}
