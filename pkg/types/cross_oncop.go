package types

import (
	"fmt"
	"strings"
)

// ONCOPType defines the category of cross-operator data being fetched.
type ONCOPType string

const (
	ONCOPMetrics ONCOPType = "metrics"
	ONCOPHealth  ONCOPType = "health"
	ONCOPInfo    ONCOPType = "info"
	ONCOPCR      ONCOPType = "cr"
	ONCOPEvents  ONCOPType = "events"
)

func MetricsType() ONCOPType { return ONCOPMetrics }
func HealthType() ONCOPType  { return ONCOPHealth }
func InfoType() ONCOPType    { return ONCOPInfo }
func CRType() ONCOPType      { return ONCOPCR }
func EventsType() ONCOPType  { return ONCOPEvents }

// String returns the string representation of the ONCOPType.
// This ensures fmt.Printf, logs, and YAML marshaling behave predictably.
func (t ONCOPType) String() string {
	switch t {
	case ONCOPMetrics:
		return "metrics"
	case ONCOPHealth:
		return "health"
	case ONCOPInfo:
		return "info"
	case ONCOPCR:
		return "cr"
	case ONCOPEvents:
		return "events"
	default:
		return string(t)
	}
}

//
// ────────────────────────────────────────────────────────────────────────────────
//   ONCOP URL BUILDER
// ────────────────────────────────────────────────────────────────────────────────
//

// BuildONCOPURL constructs an Orkestra‑native cross‑operator URL using the
// Orkestra Native Cross‑Operator Protocol (ONCOP).
//
// ONCOP allows cross‑binary CRD observation without requiring callers to
// hard‑code full URLs. When a CrossSource specifies a Host (and no Endpoint),
// the final URL is inferred from:
//
//   - the CRD name extracted from the field path (cross.<crd>.…)
//   - the Source.Type ("info", "metrics", "health", "events")
//   - the Source.Namespace override (optional)
//   - the CR's own namespace/name when Type requires it
//
// URL shapes:
//
//	Type: "cr"    → <host>/katalog/<crd>/cr/<ns>/<name>
//	Type: "info"    → <host>/katalog/<crd>
//	Type: "metrics" → <host>/katalog/<crd>
//	Type: "health"  → <host>/katalog/<crd>/health
//	Type: "events"  → <host>/katalog/<crd>/cr/<ns>/<name>/events
//
// If Source.Endpoint is provided, ONCOP is bypassed entirely and the raw
// endpoint is used as‑is. This enables integration with non‑Orkestra operators
// or arbitrary JSON‑producing services.
//
// BuildONCOPURL centralises this logic so all cross‑CRD resolution paths
// (metrics, health, info, autoscale conditions, status.fields) share the same
// URL construction semantics.
func BuildONCOPURL(decl CrossCRDDeclaration) string {
	src := decl.Source
	crd := decl.Crd
	name := decl.Selector.Name
	ns := decl.Selector.Namespace

	host := strings.TrimSuffix(src.Host, "/")

	switch src.Type {
	case ONCOPMetrics, ONCOPInfo, "":
		return fmt.Sprintf("%s/katalog/%s", host, crd)

	case ONCOPHealth:
		return fmt.Sprintf("%s/katalog/%s/health", host, crd)

	case ONCOPEvents:
		if ns == "" {
			return fmt.Sprintf("%s/katalog/%s/cr/%s/events", host, crd, name)
		}
		return fmt.Sprintf("%s/katalog/%s/cr/%s/%s/events", host, crd, ns, name)

	case ONCOPCR:
		if ns == "" {
			return fmt.Sprintf("%s/katalog/%s/cr/%s", host, crd, name)
		}
		return fmt.Sprintf("%s/katalog/%s/cr/%s/%s", host, crd, ns, name)

	default:
		return fmt.Sprintf("%s/katalog/%s", host, crd)
	}
}
