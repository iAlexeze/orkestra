// pkg/kontroller/crd_health.go
package kontroller

import (
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/utils"
)

type CRDHealth struct {
	name             string
	healthy          atomic.Bool
	totalReconciles  atomic.Int64
	failedReconciles atomic.Int64
	consecutiveFails atomic.Int64
	lastError        atomic.Value // stores string
	lastReconcile    atomic.Value // stores time.Time
}

func NewCRDHealth(name string) *CRDHealth {
	h := &CRDHealth{name: name}
	h.healthy.Store(false) // starts false — set true after first successful reconcile
	return h
}

// CRD Handler
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

// BuildCRDInfoHandler
func BuildCRDInfoHandler(crd initialize.CRDEntry, health *CRDHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"name":      crd.Name,
			"gvk":       utils.SetGroupVersionKindObj(crd.GroupVersionKind),
			"workers":   crd.Workers,
			"resync":    crd.Resync.String(),
			"dependsOn": crd.DependsOn,
			"healthy":   health.IsHealthy(),
			"errorRate": health.ErrorRate(),
		})
	}
}

// Katalog Handler
func BuildKatalogHandler(
	katalog *katalog.Katalog,
	healthMap map[string]*CRDHealth,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := make([]map[string]interface{}, 0)

		for _, crd := range katalog.Enabled() {
			gvk := utils.SetGroupVersionKindObj(crd.GroupVersionKind)
			h := healthMap[gvk]

			crds = append(crds, map[string]interface{}{
				"name":      crd.Name,
				"gvk":       gvk,
				"workers":   crd.Workers,
				"resync":    crd.Resync.String(),
				"dependsOn": crd.DependsOn,
				"healthy":   h.IsHealthy(),
				"errorRate": h.ErrorRate(),
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

func (h *CRDHealth) RecordSuccess() {
	h.totalReconciles.Add(1)
	h.consecutiveFails.Store(0)
	h.lastReconcile.Store(time.Now())
	h.healthy.Store(true)
}

func (h *CRDHealth) RecordFailure(err error, degradeThreshold int) {
	h.totalReconciles.Add(1)
	h.failedReconciles.Add(1)
	h.consecutiveFails.Add(1)
	h.lastError.Store(err.Error())
	h.lastReconcile.Store(time.Now())

	// Degrade after N consecutive failures — configurable per CRD
	if h.consecutiveFails.Load() >= int64(degradeThreshold) {
		h.healthy.Store(false)
	}
}

func (h *CRDHealth) ErrorRate() float64 {
	total := h.totalReconciles.Load()
	if total == 0 {
		return 0
	}
	return float64(h.failedReconciles.Load()) / float64(total)
}

func (h *CRDHealth) IsHealthy() bool {
	return h.healthy.Load()
}

func (h *CRDHealth) Name() string {
	return h.name
}

func (h *CRDHealth) TotalReconciles() int64 {
	return h.totalReconciles.Load()
}

func (h *CRDHealth) FailedReconciles() int64 {
	return h.failedReconciles.Load()
}

func (h *CRDHealth) LastError() string {
	return h.lastError.Load().(string)
}

func (h *CRDHealth) LastReconcile() time.Time {
	return h.lastReconcile.Load().(time.Time)
}

func (h *CRDHealth) ConsecutiveFails() int64 {
	return h.consecutiveFails.Load()
}
