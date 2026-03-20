// pkg/kontroller/handlers.go
package kontroller

import (
	"net/http"
	"strings"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/konfig"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/client-go/tools/cache"
)

// ── Health Handler ────────────────────────────────────────────────────────────

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
			"started":          h.Started(),
			"startedAt":        h.StartedAt(),
			"status":           status,
			"uptime":           h.Uptime(),
			"message":          message,
			"errorRate":        h.ErrorRate(),
			"consecutiveFails": h.consecutiveFails.Load(),
			"totalReconciles":  h.totalReconciles.Load(),
			"lastError":        h.lastError.Load(),
			"lastReconcile":    h.lastReconcile.Load(),
		})
	}
}

// ── Info Handler ──────────────────────────────────────────────────────────────

func BuildCRDInfoHandler(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
	health *CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := resolveCRDDisplayValues(crd, kfg, inf)

		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"name":             crd.Name,
			"description":      crd.Description,
			"mode":             crd.Mode,
			"gvk":              utils.SetGroupVersionKindObj(crd.GroupVersionKind),
			"gvr":              crd.GroupVersionResource.String(),
			"critical":         crd.Critical,
			"namespaced":       crd.Namespaced,
			"namespace":        crd.Namespace,
			"dependsOn":        crd.DependsOn,
			"workers":          v.workers,
			"workersSource":    v.workersSource,
			"resync":           v.resync,
			"resyncSource":     v.resyncSource,
			"queueDepth":       v.queueDepth,
			"queueDepthSource": v.queueDepthSource,
			"resourceCount":    v.resourceCount,
			"reconciler":       reconcilerInfo(crd),
			"healthy":          health.IsHealthy(),
			"started":          health.Started(),
			"errorRate":        health.ErrorRate(),
		})
	}
}

// ── Katalog Handler ───────────────────────────────────────────────────────────

func BuildKatalogHandler(
	kat *katalog.Katalog,
	kfg *konfig.Konfig,
	reg *ResourceKatalog,
	healthMap map[string]*CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := make([]map[string]interface{}, 0)

		for _, crd := range kat.Enabled() {
			gvk := utils.SetGroupVersionKindObj(crd.GroupVersionKind)
			h := healthMap[gvk]

			entry, ok := reg.Get(gvk)
			var inf cache.SharedIndexInformer
			if ok {
				inf = entry.Informer
			}

			v := resolveCRDDisplayValues(crd, kfg, inf)

			crds = append(crds, map[string]interface{}{
				"name":             crd.Name,
				"description":      crd.Description,
				"mode":             crd.Mode,
				"gvk":              gvk,
				"gvr":              crd.GroupVersionResource.String(),
				"critical":         crd.Critical,
				"namespaced":       crd.Namespaced,
				"namespace":        crd.Namespace,
				"dependsOn":        crd.DependsOn,
				"workers":          v.workers,
				"workersSource":    v.workersSource,
				"resync":           v.resync,
				"resyncSource":     v.resyncSource,
				"queueDepth":       v.queueDepth,
				"queueDepthSource": v.queueDepthSource,
				"resourceCount":    v.resourceCount,
				"reconciler":       reconcilerInfo(crd),
				"healthy":          h.IsHealthy(),
				"started":          h.Started(),
				"startedAt":        h.StartedAt(),
				"uptime":           h.Uptime(),
				"errorRate":        h.ErrorRate(),
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

// ── reconcilerInfo ────────────────────────────────────────────────────────────
// Builds the reconciler section of the API response.
// Exposes what matters operationally — mode, type, hooks presence, finalizers.
// Never exposes Go function references — those are internal.

func reconcilerInfo(crd orktypes.CRDEntry) map[string]interface{} {
	rc := crd.ReconcilerConfig

	// Reconciler type — how is this CRD being reconciled
	reconcilerType := "generic" // default: true, GenericReconciler
	if !rc.Default {
		reconcilerType = "custom" // default: false, custom Constructor
	}

	// Finalizers — show resolved list or indicate using Katalog default
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

	// Hooks — are they configured and how?
	var hooksInfo map[string]interface{}
	if rc.Hooks != nil {
		// Explicit Go hook declared in Katalog via reconciler.hooks
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "yaml",
			"location":   rc.Hooks.Location,
			"function":   rc.Hooks.Function,
		}
	} else if rc.HookFactory != nil && (rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil) {
		// Hook factory was auto-generated from declarative templates by ork generate
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "generated",
		}
	} else if rc.HookFactory != nil {
		// Hook factory set directly in Go mode (BuildKatalogFromGo)
		hooksInfo = map[string]interface{}{
			"configured": true,
			"source":     "go",
		}
	} else {
		hooksInfo = map[string]interface{}{
			"configured": false,
		}
	}

	// Constructor — custom reconciler location if declared
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
		"type":        reconcilerType, // "generic" or "custom"
		"finalizers":  finalizersInfo,
		"hooks":       hooksInfo,
		"constructor": constructorInfo,
	}

	// Declarative templates — only show if configured (dynamic mode)
	if rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil {
		templates := map[string]interface{}{}
		if rc.OnCreate != nil {
			onCreate := templateSummary(rc.OnCreate)
			// Check if any onCreate entries have reconcile: true — show onReconcile implicitly
			if hasAutoReconcile(rc.OnCreate) {
				templates["onReconcile"] = map[string]interface{}{"source": "auto", "from": "onCreate[reconcile:true]"}
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

// templateSummary returns a summary of declared resource templates.
// Shows counts rather than full declarations — keeps the API response lean.
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

// ── Display value resolution ──────────────────────────────────────────────────

type crdDisplayValues struct {
	resync           string
	resyncSource     string
	workers          int
	workersSource    string
	queueDepth       int
	queueDepthSource string
	resourceCount    int
}

func resolveCRDDisplayValues(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
) crdDisplayValues {

	// Queue Depth
	queueDepth := crd.Queue.MaxQueueDepth
	queueDepthSource := "configured"

	if queueDepth == 0 {
		queueDepth = kfg.Katalog().DefaultMaxQueueDepth
		queueDepthSource = "default"
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

	// Resource count
	resourceCount := 0
	if inf != nil {
		resourceCount = len(inf.GetStore().List())
	}

	return crdDisplayValues{
		resync:           resync,
		resyncSource:     resyncSource,
		workers:          workers,
		workersSource:    workersSource,
		queueDepth:       queueDepth,
		queueDepthSource: queueDepthSource,
		resourceCount:    resourceCount,
	}
}

// Helper
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
