package health

import (
	"encoding/json"
	"net/http"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/utils"
)

type responseReq struct {
	writer  http.ResponseWriter
	message string
	status  int
	details utils.H
}

func (h *HealthServer) writeResponse(req responseReq) {
	req.writer.Header().Set(utils.ContentType, utils.JSONContentType)
	req.writer.WriteHeader(req.status)

	// Build response with nested structs
	resp := struct {
		Data struct {
			Client string `json:"client"`
			Status int    `json:"status"`
		} `json:"data"`
		Meta struct {
			Message string  `json:"message"`
			Details utils.H `json:"details,omitempty"`
		} `json:"meta"`
	}{
		Data: struct {
			Client string `json:"client"`
			Status int    `json:"status"`
		}{
			Client: h.client,
			Status: req.status,
		},
		Meta: struct {
			Message string  `json:"message"`
			Details utils.H `json:"details,omitempty"`
		}{
			Message: req.message,
			Details: req.details,
		},
	}

	// Encode response as JSON
	if err := json.NewEncoder(req.writer).Encode(resp); err != nil {
		http.Error(req.writer, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
