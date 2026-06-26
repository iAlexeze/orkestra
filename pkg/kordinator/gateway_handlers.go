// pkg/kordinator/gateway_handlers.go
//
// HTTP handlers served by the gateway process at /katalog and /katalog/{crd}.
//
// The gateway /katalog response carries the same per-CRD shape as the runtime
// /katalog response but only populates the fields the gateway owns:
// admission, conversion, deletion-protection, and namespace-protection stats.
// Reconciler health fields (workers, queue depth, error rate, etc.) are omitted.
//
// Control center merge strategy
// ──────────────────────────────
// The runtime /katalog response includes a "gatewayEndpoint" field.
// The control center reads that URL, fetches the gateway /katalog, then merges
// per-CRD by GVR string ("group/version/resource") — the canonical key used by
// both processes.  Neither process pushes to the other; each is independently
// queryable.
package kordinator

import (
	"net/http"

	"github.com/orkspace/orkestra/pkg/health"
	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/version"
)

// GatewayStatsProvider is the subset of webhook.WebhookServer that the gateway
// /katalog handlers need.  Defined here so pkg/kordinator does not import
// pkg/webhook (which would create a cycle).
type GatewayStatsProvider interface {
	AdmissionStatsFor(gvrKey string) *health.AdmissionStats
	ConversionStatsFor(gvrKey string) *health.ConversionStats
	ProtectionStatsFor(gvrKey string) *health.DeletionProtectionStats
	NamespaceStatsFor(gvrKey string) *health.NamespaceProtectionStats
	InfraProtectionStats() *health.DeletionProtectionStats
	HousekeeperStats() *health.WebhookStats
}

// ─────────────────────────────────────────────────────────────────────────────
// Response types
// ─────────────────────────────────────────────────────────────────────────────

// GatewayKatalogResponse is served at GET /katalog by the gateway process.
// It mirrors the top-level shape of KatalogResponse so control-center clients
// can pattern-match on the "source" field and merge stats by GVR key.
type GatewayKatalogResponse struct {
	Source  string `json:"source"` // always "gateway"
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`

	// Security feature flags — mirrors Katalog configuration.
	AdmissionEnabled           bool `json:"admissionEnabled"`
	ConversionEnabled          bool `json:"conversionEnabled"`
	DeletionProtectionEnabled  bool `json:"deletionProtectionEnabled"`
	NamespaceProtectionEnabled bool `json:"namespaceProtectionEnabled"`
	StrictModeEnabled          bool `json:"strictModeEnabled"`

	// Per-CRD stats — only gateway-owned fields populated.
	CRDs []GatewayCRDStatsResponse `json:"crds"`

	// Process-level stats for events not attributable to a single CRD:
	// the webhook configuration itself and Orkestra infra resources.
	InfraProtection *DeletionProtectionStatsResponse `json:"infraProtection,omitempty"`
	Housekeeper     *HousekeeperStats                `json:"housekeeper,omitempty"`
	GatewayVersion  string                           `json:"gatewayVersion"`
}

// GatewayCRDStatsResponse holds the gateway-owned stats for one CRD.
// The GVR field is the merge key used by the control center.
type GatewayCRDStatsResponse struct {
	Name                string                           `json:"name"`
	GVK                 string                           `json:"gvk"`
	GVR                 string                           `json:"gvr"` // merge key: "group/version/resource"
	Admission           *AdmissionStatsResponse          `json:"admission,omitempty"`
	Conversion          *ConversionStatsResponse         `json:"conversion,omitempty"`
	DeletionProtection  *DeletionProtectionStatsResponse `json:"deletionProtection,omitempty"`
	NamespaceProtection *NamespaceProtectionResponse     `json:"namespaceProtection,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// BuildGatewayKatalogHandler
// ─────────────────────────────────────────────────────────────────────────────

// BuildGatewayKatalogHandler returns the handler for GET /katalog on the gateway.
// It iterates the enabled CRDs from the Katalog, looks up per-CRD stats from ws,
// and returns a GatewayKatalogResponse.
func BuildGatewayKatalogHandler(kat *katalog.Katalog, ws GatewayStatsProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crds := buildGatewayCRDStats(kat, ws)

		resp := GatewayKatalogResponse{
			Source:                     "gateway",
			Name:                       kat.Meta().Name,
			Version:                    kat.Meta().Version,
			AdmissionEnabled:           kat.IsAdmissionEnabled(),
			ConversionEnabled:          kat.IsConversionEnabled(),
			DeletionProtectionEnabled:  kat.IsDeletionProtectionEnabled(),
			NamespaceProtectionEnabled: kat.IsNamespaceProtectionEnabled(),
			StrictModeEnabled:          kat.IsStrictModeEnabled(),
			CRDs:                       crds,
			GatewayVersion:             version.Short(),
		}

		if infra := ws.InfraProtectionStats(); infra != nil {
			snap := infra.GetStats()
			resp.InfraProtection = &DeletionProtectionStatsResponse{
				Enabled: kat.IsDeletionProtectionEnabled(),
				Total:   snap.TotalRequests,
				Blocked: snap.Blocked,
				Allowed: snap.Allowed,
			}
		}

		if wcs := ws.HousekeeperStats(); wcs != nil {
			snap := wcs.GetStats()
			resp.Housekeeper = &HousekeeperStats{
				Reconciled: snap.Reconciled,
				Failed:     snap.Failed,
			}
		}

		utils.WriteJSON(w, http.StatusOK, resp)
	}
}

// BuildGatewayCRDHandler returns the handler for GET /katalog/{crd} on the gateway.
// name, gvk, gvrStr, and gvrKey are derived from the CRDEntry at registration time.
func BuildGatewayCRDHandler(name, gvk, gvrStr, gvrKey string, ws GatewayStatsProvider, kat *katalog.Katalog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := buildSingleGatewayCRDStats(name, gvk, gvrStr, gvrKey, ws, kat)
		utils.WriteJSON(w, http.StatusOK, resp)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func buildGatewayCRDStats(kat *katalog.Katalog, ws GatewayStatsProvider) []GatewayCRDStatsResponse {
	enabled := kat.Enabled()
	out := make([]GatewayCRDStatsResponse, 0, len(enabled))
	for _, crd := range enabled {
		if crd.IsBuiltIn {
			continue
		}
		gvr := crd.GVR()
		gvrKey := gvrKey(gvr.Group, gvr.Version, gvr.Resource)
		out = append(out, buildSingleGatewayCRDStats(
			crd.Name,
			crd.GVKString(),
			gvr.String(),
			gvrKey,
			ws,
			kat,
		))
	}
	return out
}

func buildSingleGatewayCRDStats(name, gvk, gvrStr, gvrKey string, ws GatewayStatsProvider, kat *katalog.Katalog) GatewayCRDStatsResponse {
	resp := GatewayCRDStatsResponse{
		Name: name,
		GVK:  gvk,
		GVR:  gvrStr,
	}

	// Admission stats — only when admission webhooks are enabled for this CRD.
	if kat.IsAdmissionEnabled() {
		if adm := ws.AdmissionStatsFor(gvrKey); adm != nil {
			snap := adm.GetStats(true)
			resp.Admission = &AdmissionStatsResponse{
				WebhooksEnabled:   true,
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
	}

	// Conversion stats.
	if kat.IsConversionEnabled() {
		if conv := ws.ConversionStatsFor(gvrKey); conv != nil {
			snap := conv.GetStats()
			resp.Conversion = &ConversionStatsResponse{
				Enabled:      true,
				Total:        snap.TotalRequests,
				Success:      snap.SuccessRequests,
				Failures:     snap.FailedRequests,
				AvgLatencyMs: snap.AvgLatency.Milliseconds(),
				P95LatencyMs: snap.P95Latency.Milliseconds(),
			}
		} else {
			resp.Conversion = &ConversionStatsResponse{Enabled: true}
		}
	}

	// Deletion protection stats.
	if kat.IsDeletionProtectionEnabled() {
		if prot := ws.ProtectionStatsFor(gvrKey); prot != nil {
			snap := prot.GetStats()
			resp.DeletionProtection = &DeletionProtectionStatsResponse{
				Enabled: true,
				Total:   snap.TotalRequests,
				Blocked: snap.Blocked,
				Allowed: snap.Allowed,
			}
		} else {
			resp.DeletionProtection = &DeletionProtectionStatsResponse{Enabled: true}
		}
	}

	// Namespace protection stats.
	if kat.IsNamespaceProtectionEnabled() {
		if ns := ws.NamespaceStatsFor(gvrKey); ns != nil {
			snap := ns.GetStats()
			resp.NamespaceProtection = &NamespaceProtectionResponse{
				Enabled: true,
				Total:   snap.TotalRequests,
				Blocked: snap.Blocked,
				Allowed: snap.Allowed,
			}
		}
	}

	return resp
}

// gvrKey formats group/version/resource into the canonical stats key.
// Matches crdGVRKey in pkg/webhook so both sides produce the same string.
func gvrKey(group, version, resource string) string {
	if group == "" {
		return version + "/" + resource
	}
	return group + "/" + version + "/" + resource
}
