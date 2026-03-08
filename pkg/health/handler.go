package health

import (
	"net/http"
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/logger"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
)

// healthHandler handles health checks -> /health
func (h *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(responseReq{
		writer:  w,
		message: h.client + " is " + string(utils.StatusHealthy),
		status:  http.StatusOK,
		details: utils.H{
			"service": h.client,
			"status":  utils.StatusOnline,
		},
	})
}

// readyHandler confirms whether a client is ready or not -> /ready
func (h *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		h.writeResponse(responseReq{
			writer:  w,
			message: h.client + " is " + string(utils.StatusNotReady),
			status:  http.StatusInternalServerError,
			details: utils.H{
				"service": h.client,
				"status":  utils.StatusNotReady,
			},
		})
		return
	}

	h.writeResponse(responseReq{
		writer:  w,
		message: h.client + " is " + string(utils.StatusReady),
		status:  http.StatusOK,
		details: utils.H{
			"service": h.client,
			"status":  utils.StatusRunning,
		},
	})
}

// logRouteMiddleware logs every handler request
func (h *HealthServer) logRouteMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		// log routes
		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			// Str("status", r.Response.Status).
			Str("userAgent", r.UserAgent()).
			Dur("duration", time.Since(start)).
			Msg("request processed")
	}
}
