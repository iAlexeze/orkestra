// pkg/kontroller/crd_health_handlers.go
package kontroller

import (
	"net/http"
	"strings"

	"github.com/ialexeze/orkestra/pkg/health"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/konfig"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/client-go/tools/cache"
)

// ─────────────────────────────────────────────────────────────────────────────
// CRD Health Handler
// Returns the live health status of a single CRD reconciler.
// This endpoint is used by:
//   - /katalog/<crd>/health
//   - dashboards
//   - readiness/liveness checks
//   - operator self‑diagnostics
//
// The handler exposes:
//   - health state (healthy/degraded)
//   - startup state
//   - uptime
//   - reconcile counters
//   - last error
//   - last reconcile timestamp
// ─────────────────────────────────────────────────────────────────────────────

func BuildCRDHealthHandler(crd orktypes.CRDEntry, h *CRDHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		message := crd.Name + " healthy"

		// Mark degraded if health flag is false
		if !h.IsHealthy() {
			status = http.StatusServiceUnavailable
			message = crd.Name + " degraded"
		}

		utils.WriteJSON(w, status, map[string]interface{}{
			"name":             crd.Name,
			"healthy":          h.IsHealthy(),
			"started":          h.Started(),
			"startedAt":        h.StartedAt(),
			"status":           status,
			"uptime":           h.Uptime(),
			"queueDepth":       h.QueueDepth(crd.GVK().String()),
			"message":          message,
			"errorRate":        h.ErrorRate(),
			"consecutiveFails": h.consecutiveFails.Load(),
			"totalReconciles":  h.totalReconciles.Load(),
			"lastError":        h.lastError.Load(),
			"lastReconcile":    h.LastReconcile(),
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD Info Handler
// Returns static + dynamic metadata about a CRD:
//   - GVK/GVR
//   - mode (dynamic/typed)
//   - workers, resync, queue depth (with source: default or configured)
//   - reconciler configuration (hooks, finalizers, constructor)
//   - resource count from informer
//   - health summary
//
// This endpoint powers:
//   - /katalog/<crd>
//   - dashboards
//   - operator introspection
// ─────────────────────────────────────────────────────────────────────────────

func BuildCRDInfoHandler(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
	h *CRDHealth,
	stats *health.ConversionStats,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := resolveCRDDisplayValues(crd, kfg, inf)

		response := map[string]interface{}{
			"name":                crd.Name,
			"description":         crd.Description,
			"mode":                crd.Mode,
			"gvk":                 crd.GVK().String(),
			"gvr":                 crd.GroupVersionResource.String(),
			"critical":            crd.Critical,
			"namespaced":          crd.Namespaced,
			"namespace":           crd.Namespace,
			"dependsOn":           crd.DependsOn,
			"workers":             v.workers,
			"workersActive":       h.WorkersActive(),
			"workersSource":       v.workersSource,
			"resync":              v.resync,
			"resyncSource":        v.resyncSource,
			"queueDepth":          h.QueueDepth(crd.GVK().String()),
			"maxQueueDepth":       v.maxQueueDepth,
			"maxQueueDepthSource": v.maxQueueDepthSource,
			"resourceCount":       v.resourceCount,
			"reconciler":          reconcilerInfo(crd),
			"healthy":             h.IsHealthy(),
			"started":             h.Started(),
			"errorRate":           h.ErrorRate(),
		}

		// Add conversion stats if available
		if stats != nil {
			snapshot := stats.GetStats()
			response["conversion"] = map[string]interface{}{
				"enabled":      true,
				"total":        snapshot.TotalRequests,
				"success":      snapshot.SuccessRequests,
				"failures":     snapshot.FailedRequests,
				"avgLatencyMs": snapshot.AvgLatency.Milliseconds(),
				"p95LatencyMs": snapshot.P95Latency.Milliseconds(),
			}
		}

		utils.WriteJSON(w, http.StatusOK, response)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Katalog Handler
// Returns a full list of CRDs in the running operator, including:
//   - metadata (GVK, mode, namespace, dependencies)
//   - resolved operational values (workers, resync, queue depth)
//   - reconciler configuration summary
//   - health summary (uptime, error rate, startedAt)
//   - per‑CRD endpoints (/health, /info)
//
// This is the top‑level endpoint for dashboards and operator UIs.
// ─────────────────────────────────────────────────────────────────────────────

func BuildKatalogHandler(
	kat *katalog.Katalog,
	kfg *konfig.Konfig,
	reg *ResourceKatalog,
	healthMap map[string]*CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := make([]map[string]interface{}, 0)

		for _, crd := range kat.Enabled() {
			gvk := crd.GVK().String()
			h := healthMap[gvk]

			entry, ok := reg.Get(gvk)
			var inf cache.SharedIndexInformer
			if ok {
				inf = entry.Informer
			}

			v := resolveCRDDisplayValues(crd, kfg, inf)

			crds = append(crds, map[string]interface{}{
				"name":                crd.Name,
				"description":         crd.Description,
				"mode":                crd.Mode,
				"gvk":                 gvk,
				"gvr":                 crd.GroupVersionResource.String(),
				"critical":            crd.Critical,
				"namespaced":          crd.Namespaced,
				"namespace":           crd.Namespace,
				"dependsOn":           crd.DependsOn,
				"workers":             v.workers,
				"workersSource":       v.workersSource,
				"workersActive":       h.WorkersActive(),
				"resync":              v.resync,
				"resyncSource":        v.resyncSource,
				"queueDepth":          h.QueueDepth(crd.GVK().String()),
				"maxQueueDepth":       v.maxQueueDepth,
				"maxQueueDepthSource": v.maxQueueDepthSource,
				"resourceCount":       v.resourceCount,
				"reconciler":          reconcilerInfo(crd),
				"healthy":             h.IsHealthy(),
				"started":             h.Started(),
				"startedAt":           h.StartedAt(),
				"uptime":              h.Uptime(),
				"errorRate":           h.ErrorRate(),
				"endpoints": map[string]string{
					"health": "/katalog/" + strings.ToLower(crd.Name) + "/health",
					"info":   "/katalog/" + strings.ToLower(crd.Name),
				},
			})
		}

		// Calculate overall health of the katalog
		healthy := true
		degradedReason := ""
		degradedCRDs := []string{}

		status := http.StatusOK
		for _, crd := range crds {
			if !crd["healthy"].(bool) {
				degradedCRDs = append(degradedCRDs, crd["name"].(string))
				break
			}
		}

		if len(degradedCRDs) > 0 {
			healthy = false
			status = http.StatusServiceUnavailable
			degradedReason = strings.Join(degradedCRDs, ", ")
			degradedReason = "degraded: " + degradedReason
		}

		utils.WriteJSON(w, http.StatusOK, KatalogResponse{
			CRDs:           crds,
			Total:          len(kat.All()),
			TotalEnabled:   len(kat.Enabled()),
			Healthy:        healthy,
			Status:         status,
			DegradedReason: degradedReason,

			// Metadata
			Name:        kat.Meta().Name,
			Version:     kat.Meta().Version,
			Author:      kat.Meta().Author,
			License:     kat.Meta().License,
			Description: kat.Meta().Description,
		})
	}
}

type KatalogResponse struct {
	CRDs           []map[string]interface{} `json:"crds"`
	TotalEnabled   int                      `json:"totalEnabled"`
	Total          int                      `json:"total"`
	Healthy        bool                     `json:"healthy"`
	Status         int                      `json:"status"`
	DegradedReason string                   `json:"degradedReason,omitempty"`
	Name           string                   `json:"name,omitempty"`
	Version        string                   `json:"version,omitempty"`
	Author         string                   `json:"author,omitempty"`
	Description    string                   `json:"description,omitempty"`
	License        string                   `json:"license,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// reconcilerInfo
// Produces a structured summary of how a CRD is reconciled.
// This avoids exposing internal Go function pointers and instead reports:
//
//   - reconciler type: generic (default) or custom
//   - finalizers: configured or default
//   - hooks: yaml, generated, go, or none
//   - constructor: yaml, go, or none
//   - declarative templates: counts only (not full templates)
//
// This keeps the API stable and safe for dashboards.
// ─────────────────────────────────────────────────────────────────────────────

func reconcilerInfo(crd orktypes.CRDEntry) map[string]interface{} {
	rc := crd.ReconcilerConfig

	// Determine reconciler type
	reconcilerType := "generic"
	if !crd.DefaultReconcile() {
		reconcilerType = "custom"
	}

	// Finalizers
	var finalizersInfo map[string]interface{}
	if len(rc.Finalizers) > 0 {
		finalizersInfo = map[string]interface{}{
			"source": "configured",
			"values": rc.Finalizers,
		}
	} else {
		finalizersInfo = map[string]interface{}{
			"source": "default",
			"values": []string{},
		}
	}

	// Hooks
	var hooksInfo map[string]interface{}
	if rc.Hooks != nil {
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "yaml",
			"location":   rc.Hooks.Location,
			"function":   rc.Hooks.Function,
		}
	} else if rc.HookFactory != nil && (rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil) {
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "generated",
		}
	} else if rc.HookFactory != nil {
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "go",
		}
	} else {
		hooksInfo = map[string]interface{}{
			"configured": false,
		}
	}

	// Constructor
	var constructorInfo map[string]interface{}
	if rc.ConstructorDecl != nil {
		constructorInfo = map[string]interface{}{
			"configured": true,
			"source":     "yaml",
			"location":   rc.ConstructorDecl.Location,
			"function":   rc.ConstructorDecl.Function,
		}
	} else if rc.Constructor != nil {
		constructorInfo = map[string]interface{}{
			"configured": true,
			"source":     "go",
		}
	} else {
		constructorInfo = map[string]interface{}{
			"configured": false,
		}
	}

	result := map[string]interface{}{
		"type":        reconcilerType,
		"finalizers":  finalizersInfo,
		"hooks":       hooksInfo,
		"constructor": constructorInfo,
	}

	// Declarative templates (dynamic mode)
	if rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil {
		templates := map[string]interface{}{}
		if rc.OnCreate != nil {
			onCreate := templateSummary(rc.OnCreate)
			if hasAutoReconcile(rc.OnCreate) {
				templates["onReconcile"] = map[string]interface{}{
					"source": "auto",
					"from":   "onCreate[reconcile:true]",
				}
			}
			templates["onCreate"] = onCreate
		}
		if rc.OnReconcile != nil {
			templates["onReconcile"] = templateSummary(rc.OnReconcile)
		}
		if rc.OnDelete != nil {
			templates["onDelete"] = templateSummary(rc.OnDelete)
		}
		result["templates"] = templates
	}

	return result
}

// ─────────────────────────────────────────────────────────────────────────────
// templateSummary
// Produces a compact summary of declarative templates for a CRD.
// Only counts are returned — not full templates — to keep API responses small.
// ─────────────────────────────────────────────────────────────────────────────

func templateSummary(t *orktypes.HookTemplates) map[string]interface{} {
	if t == nil {
		return map[string]interface{}{}
	}

	summary := map[string]interface{}{}

	if len(t.Deployments) > 0 {
		summary["deployments"] = len(t.Deployments)
	}
	if len(t.Services) > 0 {
		summary["services"] = len(t.Services)
	}
	if len(t.Pods) > 0 {
		summary["pods"] = len(t.Pods)
	}
	if len(t.Jobs) > 0 {
		summary["jobs"] = len(t.Jobs)
	}
	if len(t.CronJobs) > 0 {
		summary["cronJobs"] = len(t.CronJobs)
	}
	if len(t.ConfigMaps) > 0 {
		summary["configMaps"] = len(t.ConfigMaps)
	}
	if len(t.ServiceAccounts) > 0 {
		summary["serviceAccounts"] = len(t.ServiceAccounts)
	}

	return summary
}

// ─────────────────────────────────────────────────────────────────────────────
// resolveCRDDisplayValues
// Resolves operational values for a CRD, including:
//   - workers (configured or default)
//   - resync period (configured or default)
//   - queue depth (configured or default)
//   - resource count from informer
//
// This ensures the API always shows the *effective* values the operator uses.
// ─────────────────────────────────────────────────────────────────────────────

type crdDisplayValues struct {
	resync              string
	resyncSource        string
	workers             int
	workersSource       string
	maxQueueDepth       int
	maxQueueDepthSource string
	resourceCount       int
}

func resolveCRDDisplayValues(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
) crdDisplayValues {

	// Queue depth
	maxQueueDepth := crd.Queue.MaxQueueDepth
	maxQueueDepthSource := "configured"
	if maxQueueDepth == 0 {
		maxQueueDepth = kfg.Katalog().DefaultMaxQueueDepth
		maxQueueDepthSource = "default"
	}

	// Resync
	resync := crd.Resync.String()
	resyncSource := "configured"
	if crd.Resync == 0 {
		resyncSource = "default"
		resync = kfg.Cluster().DefaultResync.String()
	}

	// Workers
	workers := crd.Workers
	workersSource := "configured"
	if crd.Workers == 0 {
		workers = kfg.Cluster().DefaultWorkers
		workersSource = "default"
	}

	// Resource count from informer
	resourceCount := 0
	if inf != nil {
		resourceCount = len(inf.GetStore().List())
	}

	return crdDisplayValues{
		resync:              resync,
		resyncSource:        resyncSource,
		workers:             workers,
		workersSource:       workersSource,
		maxQueueDepth:       maxQueueDepth,
		maxQueueDepthSource: maxQueueDepthSource,
		resourceCount:       resourceCount,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// hasAutoReconcile
// Returns true if any declarative template has `reconcile: true`,
// meaning onCreate implicitly triggers onReconcile.
// ─────────────────────────────────────────────────────────────────────────────

func hasAutoReconcile(t *orktypes.HookTemplates) bool {
	if t == nil {
		return false
	}
	for _, d := range t.Deployments {
		if d.Reconcile {
			return true
		}
	}
	for _, s := range t.Services {
		if s.Reconcile {
			return true
		}
	}
	return false
}
