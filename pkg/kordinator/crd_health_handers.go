// pkg/kordinator/crd_health_handlers.go
package kordinator

import (
	"fmt"
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
// CRD Health Response
// ─────────────────────────────────────────────────────────────────────────────

type CRDHealthResponse struct {
	Name                     string                      `json:"name"`
	State                    string                      `json:"state"` // "not started", "pending", "started", "healthy", "degraded"
	Healthy                  bool                        `json:"healthy"`
	Started                  bool                        `json:"started"`
	Pending                  bool                        `json:"pending"`
	StartedAt                string                      `json:"startedAt"`
	Uptime                   string                      `json:"uptime"`
	QueueDepth               int                         `json:"queueDepth"`
	ErrorRate                float64                     `json:"errorRate"`
	ConsecutiveFails         int64                       `json:"consecutiveFails"`
	TotalReconciles          int64                       `json:"totalReconciles"`
	ResourceCount            int                         `json:"resourceCount"`
	LastError                string                      `json:"lastError"`
	LastReconcile            string                      `json:"lastReconcile"`
	HasUnhealthyDependencies bool                        `json:"hasUnhealthyDependencies"`
	Dependencies             map[string]DependencyStatus `json:"dependencies,omitempty"`
	Missing                  bool                        `json:"missing,omitempty"`
}

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
//
// ─────────────────────────────────────────────────────────────────────────────
func BuildCRDHealthHandler(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
	h *CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isStarted := h.Started()
		isPending := h.Pending()
		isHealthy := h.IsHealthy()

		var httpStatus int
		var state string

		switch {
		case !isStarted && !isPending:
			httpStatus = http.StatusServiceUnavailable
			state = "not started"
		case isPending:
			httpStatus = http.StatusOK
			state = "pending"
		case isStarted && !isHealthy:
			httpStatus = http.StatusServiceUnavailable
			state = "degraded"
		case isHealthy:
			httpStatus = http.StatusOK
			state = "healthy"
		default:
			httpStatus = http.StatusOK
			state = "pending"
		}

		v := resolveCRDDisplayValues(crd, kfg, inf)

		response := CRDHealthResponse{
			Name:                     crd.Name,
			State:                    state,
			Healthy:                  isHealthy,
			Started:                  isStarted,
			Pending:                  isPending,
			StartedAt:                h.StartedAt(),
			Uptime:                   h.Uptime(),
			QueueDepth:               h.QueueDepth(crd.GVK().String()),
			ErrorRate:                h.ErrorRate(),
			ConsecutiveFails:         h.ConsecutiveFails(),
			TotalReconciles:          h.TotalReconciles(),
			ResourceCount:            v.resourceCount,
			LastError:                h.LastError(),
			LastReconcile:            h.LastReconcile(),
			Dependencies:             h.GetDependencyStatuses(),
			HasUnhealthyDependencies: h.HasUnhealthyDependencies(),
			Missing:                  h.IsMissing(),
		}

		utils.WriteJSON(w, httpStatus, response)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CRD Info Response
// ─────────────────────────────────────────────────────────────────────────────
//
// Future use
type WorkerStats struct {
	Workers           int32             `json:"workers"`
	WorkersActive     int32             `json:"workersActive"`
	WorkersIdle       int32             `json:"workersIdle"`
	WorkersProcessing int32             `json:"workersProcessing"`
	WorkerDetails     map[string]string `json:"workerDetails,omitempty"`
}

type CRDInfoResponse struct {
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	Mode                string                   `json:"mode"`
	GVK                 string                   `json:"gvk"`
	GVR                 string                   `json:"gvr"`
	Namespaced          *bool                    `json:"namespaced"`
	Namespace           string                   `json:"namespace"`
	DependsOn           []string                 `json:"dependsOn,omitempty"`
	Workers             int                      `json:"workers"`
	WorkersActive       int32                    `json:"workersActive"`
	WorkersIdle         int32                    `json:"workersIdle"`
	WorkersProcessing   int32                    `json:"workersProcessing"`
	WorkerDetails       map[string]string        `json:"workerDetails,omitempty"`
	WorkersSource       string                   `json:"workersSource"`
	Resync              string                   `json:"resync"`
	ResyncSource        string                   `json:"resyncSource"`
	QueueDepth          int                      `json:"queueDepth"`
	MaxQueueDepth       int                      `json:"maxQueueDepth"`
	MaxQueueDepthSource string                   `json:"maxQueueDepthSource"`
	ResourceCount       int                      `json:"resourceCount"`
	TotalReconciles     int64                    `json:"totalReconciles"`
	Reconciler          ReconcilerInfo           `json:"reconciler"`
	Healthy             bool                     `json:"healthy"`
	Started             bool                     `json:"started"`
	Pending             bool                     `json:"pending"`
	ErrorRate           float64                  `json:"errorRate"`
	Conversion          *ConversionStatsResponse `json:"conversion,omitempty"`
	Admission           *AdmissionStatsResponse  `json:"admission,omitempty"`
	Protection          *ProtectionStatsResponse `json:"protection,omitempty"`
	RBAC                RBACInfo                 `json:"rbac,omitempty"`
}

type ReconcilerInfo struct {
	Type        string                 `json:"type"`
	Finalizers  FinalizersInfo         `json:"finalizers"`
	Hooks       HooksInfo              `json:"hooks"`
	Constructor ConstructorInfo        `json:"constructor"`
	Templates   map[string]interface{} `json:"templates,omitempty"`
}

type FinalizersInfo struct {
	Source string   `json:"source"`
	Values []string `json:"values"`
}

type HooksInfo struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"`
	Location   string `json:"location,omitempty"`
	Function   string `json:"function,omitempty"`
}

type ConstructorInfo struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source,omitempty"`
	Location   string `json:"location,omitempty"`
	Function   string `json:"function,omitempty"`
}

type ConversionStatsResponse struct {
	Enabled      bool  `json:"enabled"`
	Total        int64 `json:"total"`
	Success      int64 `json:"success"`
	Failures     int64 `json:"failures"`
	AvgLatencyMs int64 `json:"avgLatencyMs"`
	P95LatencyMs int64 `json:"p95LatencyMs"`
}

type AdmissionStatsResponse struct {
	WebhooksEnabled   bool    `json:"webhooksEnabled"`
	ValidationTotal   int64   `json:"validationTotal"`
	ValidationAllowed int64   `json:"validationAllowed"`
	ValidationDenied  int64   `json:"validationDenied"`
	ValidationWarned  int64   `json:"validationWarned"`
	ValAvgLatencyMs   float64 `json:"valAvgLatencyMs"`
	ValP95LatencyMs   float64 `json:"valP95LatencyMs"`
	ValMaxLatencyMs   float64 `json:"valMaxLatencyMs"`
	MutationTotal     int64   `json:"mutationTotal"`
	MutationApplied   int64   `json:"mutationApplied"`
	MutationSkipped   int64   `json:"mutationSkipped"`
	MutAvgLatencyMs   float64 `json:"mutAvgLatencyMs"`
	MutP95LatencyMs   float64 `json:"mutP95LatencyMs"`
	MutMaxLatencyMs   float64 `json:"mutMaxLatencyMs"`
}

// ProtectionStatsResponse exposes deletion protection status for the CRD detail view.
// All counts are cumulative since operator startup.
type ProtectionStatsResponse struct {
	Enabled bool  `json:"enabled"`
	Total   int64 `json:"total"`   // total DELETE admission reviews received
	Blocked int64 `json:"blocked"` // DELETE requests denied
	Allowed int64 `json:"allowed"` // DELETE requests allowed through
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
//
// ─────────────────────────────────────────────────────────────────────────────
func BuildCRDInfoHandler(
	crd orktypes.CRDEntry,
	kfg *konfig.Konfig,
	inf cache.SharedIndexInformer,
	h *CRDHealth,
	stats *health.ConversionStats,
	admStats *health.AdmissionStats,
	protStats *health.ProtectionStats,
	isProtected bool,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := resolveCRDDisplayValues(crd, kfg, inf)

		// Generate RBAC info for this CRD
		rbacInfo := generateRBACInfo(crd, v)

		response := CRDInfoResponse{
			Name:                crd.Name,
			Description:         crd.Description,
			Mode:                crd.Mode.String(),
			GVK:                 crd.GVK().String(),
			GVR:                 crd.GroupVersionResource.String(),
			Namespaced:          crd.Namespaced,
			Namespace:           crd.Namespace,
			DependsOn:           crd.DependsOn.Names(),
			Workers:             v.workers,
			WorkersActive:       h.GetActiveWorkers(),
			WorkersIdle:         h.GetIdleWorkers(),
			WorkersProcessing:   h.GetProcessingWorkers(),
			WorkerDetails:       h.GetWorkerStates(),
			WorkersSource:       v.workersSource,
			Resync:              v.resync,
			ResyncSource:        v.resyncSource,
			QueueDepth:          h.QueueDepth(crd.GVK().String()),
			MaxQueueDepth:       v.maxQueueDepth,
			MaxQueueDepthSource: v.maxQueueDepthSource,
			ResourceCount:       v.resourceCount,
			TotalReconciles:     h.TotalReconciles(),
			Reconciler:          reconcilerInfoStruct(crd),
			Healthy:             h.IsHealthy(),
			Started:             h.Started(),
			Pending:             h.Pending(),
			ErrorRate:           h.ErrorRate(),
			RBAC:                rbacInfo,
		}

		if stats != nil {
			snapshot := stats.GetStats()
			response.Conversion = &ConversionStatsResponse{
				Enabled:      kfg.ConversionEnabled(),
				Total:        snapshot.TotalRequests,
				Success:      snapshot.SuccessRequests,
				Failures:     snapshot.FailedRequests,
				AvgLatencyMs: snapshot.AvgLatency.Milliseconds(),
				P95LatencyMs: snapshot.P95Latency.Milliseconds(),
			}
		}

		if admStats != nil {
			snap := admStats.GetStats(crd.Webhooks.WebhookValidationEnabled() || crd.Webhooks.WebhookMutationEnabled())
			response.Admission = &AdmissionStatsResponse{
				WebhooksEnabled:   kfg.AdmissionEnabled(),
				ValidationTotal:   snap.ValidationTotal,
				ValidationAllowed: snap.ValidationAllowed,
				ValidationDenied:  snap.ValidationDenied,
				ValidationWarned:  snap.ValidationWarned,
				ValAvgLatencyMs:   snap.ValAvgLatencyMs,
				ValP95LatencyMs:   snap.ValP95LatencyMs,
				ValMaxLatencyMs:   snap.ValMaxLatencyMs,
				MutationTotal:     snap.MutationTotal,
				MutationApplied:   snap.MutationApplied,
				MutationSkipped:   snap.MutationSkipped,
				MutAvgLatencyMs:   snap.MutAvgLatencyMs,
				MutP95LatencyMs:   snap.MutP95LatencyMs,
				MutMaxLatencyMs:   snap.MutMaxLatencyMs,
			}
		}

		if protStats != nil {
			snap := protStats.GetStats()
			response.Protection = &ProtectionStatsResponse{
				Enabled: isProtected,
				Total:   snap.TotalRequests,
				Blocked: snap.Blocked,
				Allowed: snap.Allowed,
			}
		} else {
			response.Protection = &ProtectionStatsResponse{Enabled: isProtected}
		}

		utils.WriteJSON(w, http.StatusOK, response)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Katalog Response
// ─────────────────────────────────────────────────────────────────────────────

type KatalogResponse struct {
	CRDs               []CRDSummaryResponse `json:"crds"`
	Total              int                  `json:"total"`
	TotalEnabled       int                  `json:"totalEnabled"`
	OrkReady           bool                 `json:"OrkReady"`
	DeletionProtection bool                 `json:"deletionProtection"`
	Healthy            bool                 `json:"healthy"`
	Status             int                  `json:"status"`
	DegradedReason     string               `json:"degradedReason,omitempty"`
	StatusCounts       StatusCounts         `json:"statusCounts"`
	Name               string               `json:"name,omitempty"`
	Version            string               `json:"version,omitempty"`
	Author             string               `json:"author,omitempty"`
	Description        string               `json:"description,omitempty"`
	License            string               `json:"license,omitempty"`
}

type CRDSummaryResponse struct {
	Name                     string            `json:"name"`
	Description              string            `json:"description"`
	Mode                     string            `json:"mode"`
	GVK                      string            `json:"gvk"`
	GVR                      string            `json:"gvr"`
	Namespaced               *bool             `json:"namespaced"`
	Namespace                string            `json:"namespace"`
	DependsOn                []string          `json:"dependsOn,omitempty"`
	HasUnhealthyDependencies bool              `json:"hasUnhealthyDependencies"`
	Workers                  int               `json:"workers"`
	WorkersSource            string            `json:"workersSource"`
	WorkersActive            int32             `json:"workersActive"`
	Resync                   string            `json:"resync"`
	ResyncSource             string            `json:"resyncSource"`
	QueueDepth               int               `json:"queueDepth"`
	MaxQueueDepth            int               `json:"maxQueueDepth"`
	MaxQueueDepthSource      string            `json:"maxQueueDepthSource"`
	ResourceCount            int               `json:"resourceCount"`
	Reconciler               ReconcilerSummary `json:"reconciler"`
	Healthy                  bool              `json:"healthy"`
	State                    string            `json:"state"`
	Started                  bool              `json:"started"`
	Pending                  bool              `json:"pending"`
	StartedAt                string            `json:"startedAt"`
	Uptime                   string            `json:"uptime"`
	ErrorRate                float64           `json:"errorRate"`
	Endpoints                EndpointInfo      `json:"endpoints"`
	RBACCount                int               `json:"rbacCount,omitempty"`
	DeletionProtection       bool              `json:"deletionProtection"`
}

type ReconcilerSummary struct {
	Type           string `json:"type"`
	HasTemplates   bool   `json:"hasTemplates,omitempty"`
	HasHooks       bool   `json:"hasHooks,omitempty"`
	HasConstructor bool   `json:"hasConstructor,omitempty"`
}

type EndpointInfo struct {
	Health string `json:"health"`
	Info   string `json:"info"`
}

type StatusCounts struct {
	Healthy  int `json:"healthy"`
	Degraded int `json:"degraded"`
	Started  int `json:"started"`
	Pending  int `json:"pending"`
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
	o *OrkestraHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := make([]CRDSummaryResponse, 0)
		statusCounts := StatusCounts{}
		protectedCRDs := kat.ProtectedCRDNames()

		for _, crd := range kat.Enabled() {
			gvk := crd.GVK().String()
			h := healthMap[gvk]

			entry, ok := reg.Get(gvk)
			var inf cache.SharedIndexInformer
			if ok {
				inf = entry.Informer
			}

			v := resolveCRDDisplayValues(crd, kfg, inf)

			isStarted := h.Started()
			isPending := h.Pending()
			isHealthy := h.IsHealthy()

			var state string

			switch {
			case !isStarted && !isPending:
				state = "not started"
				statusCounts.Pending++
			case isPending:
				state = "pending"
				statusCounts.Pending++
			case isStarted && !isHealthy:
				state = "degraded"
				statusCounts.Degraded++
			case isHealthy:
				state = "healthy"
				statusCounts.Healthy++
			default:
				state = "pending"
				statusCounts.Pending++
			}

			crds = append(crds, CRDSummaryResponse{
				Name:                     crd.Name,
				State:                    state,
				HasUnhealthyDependencies: h.HasUnhealthyDependencies(),
				Description:              crd.Description,
				Mode:                     crd.Mode.String(),
				GVK:                      gvk,
				GVR:                      crd.GroupVersionResource.String(),
				Namespaced:               crd.Namespaced,
				Namespace:                crd.Namespace,
				DependsOn:                crd.DependsOn.Names(),
				Workers:                  v.workers,
				WorkersSource:            v.workersSource,
				WorkersActive:            h.GetActiveWorkers(),
				Resync:                   v.resync,
				ResyncSource:             v.resyncSource,
				QueueDepth:               h.QueueDepth(gvk),
				MaxQueueDepth:            v.maxQueueDepth,
				MaxQueueDepthSource:      v.maxQueueDepthSource,
				RBACCount:                generateRBACInfo(crd, v).TotalRules,
				ResourceCount:            v.resourceCount,
				DeletionProtection:       isCRDProtected(protectedCRDs, crd.APITypes.Plural, crd.APITypes.Group),
				Reconciler: ReconcilerSummary{
					Type:           "generic",
					HasTemplates:   crd.ReconcilerConfig.OnCreate != nil,
					HasHooks:       crd.ReconcilerConfig.Hooks != nil || crd.ReconcilerConfig.HookFactory != nil,
					HasConstructor: crd.ReconcilerConfig.Constructor != nil,
				},
				Healthy:   isHealthy,
				Started:   isStarted,
				Pending:   isPending,
				StartedAt: h.StartedAt(),
				Uptime:    h.Uptime(),
				ErrorRate: h.ErrorRate(),
				Endpoints: EndpointInfo{
					Health: "/katalog/" + strings.ToLower(crd.Name) + "/health",
					Info:   "/katalog/" + strings.ToLower(crd.Name),
				},
			})
		}

		healthy := statusCounts.Degraded == 0
		status := http.StatusOK
		degradedReason := ""

		if !healthy {
			status = http.StatusServiceUnavailable
			var parts []string
			if statusCounts.Degraded > 0 {
				parts = append(parts, fmt.Sprintf("%d degraded", statusCounts.Degraded))
			}
			if statusCounts.Pending > 0 {
				parts = append(parts, fmt.Sprintf("%d pending", statusCounts.Pending))
			}
			if statusCounts.Started > 0 {
				parts = append(parts, fmt.Sprintf("%d started", statusCounts.Started))
			}
			degradedReason = strings.Join(parts, ", ")
		}

		deletionProtection := false
		if kat.IsDeletionProtectionEnabled() && kat.DeletionProtectionGVRs() != nil {
			deletionProtection = true
		}

		utils.WriteJSON(w, status, KatalogResponse{
			CRDs:               crds,
			Total:              len(kat.All()),
			TotalEnabled:       len(kat.Enabled()),
			OrkReady:           o.IsOrkReady(),
			DeletionProtection: deletionProtection,
			Healthy:            o.IsKatalogReady(),
			Status:             status,
			DegradedReason:     degradedReason,
			StatusCounts:       statusCounts,
			Name:               kat.Meta().Name,
			Version:            kat.Meta().Version,
			Author:             kat.Meta().Author,
			License:            kat.Meta().License,
			Description:        kat.Meta().Description,
		})
	}
}

// isCRDProtected returns true when the CRD identified by (plural, group) is
// present in the protected names set built from the Katalog at startup.
// Returns false when the set is nil (deletion protection not enabled) or when
// the CRD is not managed by this operator.
func isCRDProtected(protected map[string]struct{}, plural, group string) bool {
	if protected == nil {
		return false
	}
	_, ok := protected[plural+"."+group]
	return ok
}

// Helper function to convert to struct-based reconciler info
func reconcilerInfoStruct(crd orktypes.CRDEntry) ReconcilerInfo {
	rc := crd.ReconcilerConfig

	reconcilerType := "generic"
	if !crd.DefaultReconcile() {
		reconcilerType = "custom"
	}

	finalizersInfo := FinalizersInfo{
		Source: "default",
		Values: []string{},
	}
	if len(rc.Finalizers) > 0 {
		finalizersInfo.Source = "configured"
		finalizersInfo.Values = rc.Finalizers
	}

	hooksInfo := HooksInfo{Configured: false}
	if rc.Hooks != nil {
		hooksInfo = HooksInfo{
			Configured: true,
			Source:     "yaml",
			Location:   rc.Hooks.Location,
			Function:   rc.Hooks.Function,
		}
	} else if rc.HookFactory != nil {
		hooksInfo = HooksInfo{
			Configured: true,
			Source:     "go",
		}
	}

	constructorInfo := ConstructorInfo{Configured: false}
	if rc.ConstructorDecl != nil {
		constructorInfo = ConstructorInfo{
			Configured: true,
			Source:     "yaml",
			Location:   rc.ConstructorDecl.Location,
			Function:   rc.ConstructorDecl.Function,
		}
	} else if rc.Constructor != nil {
		constructorInfo = ConstructorInfo{
			Configured: true,
			Source:     "go",
		}
	}

	result := ReconcilerInfo{
		Type:        reconcilerType,
		Finalizers:  finalizersInfo,
		Hooks:       hooksInfo,
		Constructor: constructorInfo,
	}

	if rc.OnCreate != nil || rc.OnReconcile != nil || rc.OnDelete != nil {
		result.Templates = make(map[string]interface{})
		if rc.OnCreate != nil {
			result.Templates["onCreate"] = templateSummary(rc.OnCreate)
		}
		if rc.OnReconcile != nil {
			result.Templates["onReconcile"] = templateSummary(rc.OnReconcile)
		}
		if rc.OnDelete != nil {
			result.Templates["onDelete"] = templateSummary(rc.OnDelete)
		}
	}

	return result
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
