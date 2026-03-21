// cmd/cli/status.go
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ialexeze/orkestra/pkg/inspect"
	"github.com/spf13/cobra"
)

// ── Health API response types ──────────────────────────────────────────────────
// These mirror the JSON shape of GET /katalog.
// All fields from CRDHealth are represented here — nothing omitted.

type katalogResponse struct {
	Name           string    `json:"name"`
	Healthy        bool      `json:"healthy"`
	Ready          bool      `json:"ready"`
	Total          int       `json:"total"`
	TotalEnabled   int       `json:"totalEnabled"`
	DegradedReason string    `json:"degradedReason"`
	Uptime         string    `json:"uptime"`
	CRDs           []crdStat `json:"crds"`
}

// crdStat mirrors the per-CRD entry in the /katalog JSON response.
// Every field maps directly to a CRDHealth method or Katalog config value.
type crdStat struct {
	// Identity
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Group   string `json:"group"`
	Version string `json:"version"`

	// Worker pool — from Katalog config + DependencyKontroller
	Workers       int `json:"workers"`       // crd.Workers from Katalog
	WorkersActive int `json:"workersActive"` // currently running workers

	// Workqueue — from QueueRegistry
	QueueDepth    int `json:"queueDepth"`    // current items waiting for a worker
	MaxQueueDepth int `json:"maxQueueDepth"` // configured max depth from Katalog

	// Health — all driven by CRDHealth
	Healthy          bool   `json:"healthy"`          // CRDHealth.IsHealthy()
	Started          bool   `json:"started"`          // CRDHealth.Started()
	StartedAt        string `json:"startedAt"`        // CRDHealth.StartedAt()
	Uptime           string `json:"uptime"`           // CRDHealth.Uptime()
	DegradeThreshold int    `json:"degradeThreshold"` // from Katalog — fails before unhealthy

	// Reconcile counters — from CRDHealth atomics
	TotalReconciles  int64   `json:"totalReconciles"`  // CRDHealth.TotalReconciles()
	FailedReconciles int64   `json:"failedReconciles"` // CRDHealth.FailedReconciles()
	ConsecutiveFails int64   `json:"consecutiveFails"` // CRDHealth.ConsecutiveFails() — resets on success
	ErrorRate        float64 `json:"errorRate"`        // CRDHealth.ErrorRate()
	LastReconcile    string  `json:"lastReconcile"`    // CRDHealth.LastReconcile()

	// Error detail
	LastError string `json:"lastError,omitempty"` // CRDHealth.LastError()

	// Informer cache
	ResourceCount int64 `json:"resourceCount"` // live CR count from informer cache
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show health and reconcile stats of a running Orkestra operator",
	Long: `Connect to a running Orkestra operator and show its full health status.

Connects to the health API (default: localhost:8080).
Port-forward to access a cluster deployment:

  kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system
  ork status

What each column shows:
  CRD          CRD name from the Katalog
  WORKERS      active workers / configured workers
  QUEUE        current depth / max depth
  HEALTH       ● healthy  ⚠ has consecutive failures  ● degraded  · not started
  CONSEC       consecutive failures / degrade threshold (resets on success)
  RECONCILES   total reconcile attempts
  FAILED       total failed reconciles
  ERR%         failed / total as a percentage
  RESOURCES    live CR count from the informer cache
  UPTIME       how long this CRD's reconciler has been running`,

	Example: `  ork status
  ork status --url https://orkestra.platform.myorg.io
  ork status --crd website`,

	RunE: func(cmd *cobra.Command, args []string) error {
		url, _ := cmd.Flags().GetString("url")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		crdFilter, _ := cmd.Flags().GetString("crd")

		client := &http.Client{Timeout: timeout}

		body, err := fetchJSON(client, url+"/katalog")
		if err != nil {
			return fmt.Errorf(
				"cannot reach Orkestra at %s: %w\n\n"+
					"Hint: kubectl port-forward svc/orkestra 8080:8080 -n orkestra-system",
				url, err,
			)
		}

		var status katalogResponse
		if err := json.Unmarshal(body, &status); err != nil {
			return fmt.Errorf("parsing /katalog response: %w", err)
		}

		// ── Header ────────────────────────────────────────────────────────────
		overallHealth := inspect.HealthIcon("ready") + " healthy"
		var degradedReason string
		if !status.Healthy {
			overallHealth = inspect.HealthIcon("error") + " degraded"
			degradedReason = status.DegradedReason
		}

		fmt.Printf("\n%s\n", inspect.Bold("Orkestra Operator Status"))
		fmt.Printf("%-20s %s\n", "Operator:", inspect.Bold(status.Name))
		fmt.Printf("%-20s %s\n", "Health:", overallHealth)
		if degradedReason != "" {
			fmt.Printf("%-20s %s\n", "Reason:", degradedReason)
		}
		fmt.Printf("%-20s %d total, %d enabled\n", "CRDs:", status.Total, status.TotalEnabled)
		if status.Uptime != "" {
			fmt.Printf("%-20s %s\n", "Uptime:", status.Uptime)
		}
		fmt.Println()

		if len(status.CRDs) == 0 {
			inspect.PrintInfo("No enabled CRDs.")
			return nil
		}

		// Optional filter
		crds := status.CRDs
		if crdFilter != "" {
			filtered := make([]crdStat, 0)
			for _, c := range crds {
				if strings.EqualFold(c.Name, crdFilter) || strings.EqualFold(c.Kind, crdFilter) {
					filtered = append(filtered, c)
				}
			}
			if len(filtered) == 0 {
				return fmt.Errorf("CRD %q not found in running operator", crdFilter)
			}
			crds = filtered
		}

		// ── CRD table ─────────────────────────────────────────────────────────
		header := []string{
			"CRD",
			"WORKERS",
			"QUEUE",
			"HEALTH",
			"CONSEC",
			"RECONCILES",
			"FAILED",
			"ERR%",
			"RESOURCES",
			"UPTIME",
		}

		rows := make([][]string, 0, len(crds))
		hasErrors := false

		for _, crd := range crds {
			// WORKERS: active/configured
			workers := fmt.Sprintf("%d/%d", crd.WorkersActive, crd.Workers)

			// QUEUE: depth/maxDepth — shows queue pressure at a glance
			queue := fmt.Sprintf("%d", crd.QueueDepth)
			if crd.MaxQueueDepth > 0 {
				queue = fmt.Sprintf("%d/%d", crd.QueueDepth, crd.MaxQueueDepth)
			}

			// HEALTH: derived from started state, healthy flag, and consecutive fails
			healthIcon := healthIconFromStat(crd)

			// CONSEC: consecutive fails / degrade threshold
			// Shows how far this CRD is from being marked unhealthy.
			// Resets to 0 on every successful reconcile.
			consec := fmt.Sprintf("%d", crd.ConsecutiveFails)
			if crd.DegradeThreshold > 0 {
				consec = fmt.Sprintf("%d/%d", crd.ConsecutiveFails, crd.DegradeThreshold)
			}

			// ERR%: only shown once reconciles have occurred
			errPct := "-"
			if crd.TotalReconciles > 0 {
				errPct = fmt.Sprintf("%.1f%%", crd.ErrorRate*100)
			}

			// UPTIME: how long this CRD has been active
			uptime := crd.Uptime
			if !crd.Started {
				uptime = inspect.Cyan("not started")
			}

			rows = append(rows, []string{
				crd.Name,
				workers,
				queue,
				healthIcon,
				consec,
				fmt.Sprintf("%d", crd.TotalReconciles),
				fmt.Sprintf("%d", crd.FailedReconciles),
				errPct,
				fmt.Sprintf("%d", crd.ResourceCount),
				uptime,
			})

			if crd.LastError != "" {
				hasErrors = true
			}
		}

		inspect.PrintTable(os.Stdout, header, rows)

		// ── Last errors ───────────────────────────────────────────────────────
		// Full error message per CRD, below the table — not truncated.
		if hasErrors {
			fmt.Println()
			inspect.PrintSection("Last Errors")
			for _, crd := range crds {
				if crd.LastError != "" {
					fmt.Printf("  %s %s\n", inspect.Bold(crd.Name+":"), crd.LastError)
				}
			}
		}

		// ── Legend ────────────────────────────────────────────────────────────
		minThreshold := findMinDegradeThreshold(crds)
		fmt.Printf(
			"\n\n\n%s %s=healthy  %s=consecutive fails present  %s=degraded (≥%d consecutive)  %s=not started\n",
			inspect.Bold("Legend:"),
			inspect.HealthIcon("ready"),
			inspect.HealthIcon("pending"),
			inspect.HealthIcon("error"),
			minThreshold,
			inspect.Cyan("·"),
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().String("url", "http://localhost:8080", "Orkestra operator URL")
	statusCmd.Flags().Duration("timeout", 5*time.Second, "Request timeout")
	statusCmd.Flags().String("crd", "", "Filter output to a specific CRD name or Kind")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// healthIconFromStat derives the health icon from the full set of CRDHealth data.
//
// Priority:
//
//	· (cyan)  — not started yet
//	● (red)   — healthy=false, crossed degradeThreshold
//	⚠ (yellow) — healthy=true but has consecutive failures (approaching threshold)
//	● (green)  — healthy, no consecutive failures
func healthIconFromStat(crd crdStat) string {
	if !crd.Started {
		return inspect.Cyan("·")
	}
	if !crd.Healthy {
		// Crossed the degrade threshold — fully degraded
		return inspect.HealthIcon("error")
	}
	if crd.ConsecutiveFails > 0 {
		// Has failures but hasn't crossed threshold yet — early warning
		return inspect.HealthIcon("pending")
	}
	return inspect.HealthIcon("ready")
}

// findMinDegradeThreshold returns the smallest non-zero degradeThreshold
// across all CRDs. Used in the legend so the threshold value is meaningful.
func findMinDegradeThreshold(crds []crdStat) int {
	min := 0
	for _, c := range crds {
		if c.DegradeThreshold > 0 && (min == 0 || c.DegradeThreshold < min) {
			min = c.DegradeThreshold
		}
	}
	if min == 0 {
		return 3 // sensible default when not explicitly configured
	}
	return min
}

// fetchJSON performs a GET request and returns the response body.
// Returns a descriptive error on non-200 responses.
func fetchJSON(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
