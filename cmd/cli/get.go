// cmd/cli/get.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ialexeze/orkestra/pkg/inspect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

var getCmd = &cobra.Command{
	Use:   "get <crd>",
	Short: "List all CRs of a given CRD type",
	Long: `List all Custom Resources of the specified CRD type.

The CRD name can be plural, singular, or Kind (case-insensitive):
  ork get websites
  ork get website
  ork get Website

When the Orkestra operator is reachable, a health banner is printed above
the resource list showing what only Orkestra knows about this CRD:
workers, queue depth/max, consecutive failures, degrade threshold, error rate.

If the operator is not reachable the resource list is shown without the banner.
Use --no-health to suppress the banner explicitly.`,

	Example: `  ork get websites
  ork get website -n production
  ork get Website -A
  ork get website --no-health`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")
		allNamespaces, _ := cmd.Flags().GetBool("all-namespaces")
		operatorURL, _ := cmd.Flags().GetString("operator-url")
		noHealth, _ := cmd.Flags().GetBool("no-health")

		clients, err := inspect.NewClients(kubeconfig)
		if err != nil {
			return err
		}

		crd, err := inspect.DiscoverCRD(clients.Discovery, args[0])
		if err != nil {
			return err
		}

		// ── Optional health banner ────────────────────────────────────────────
		// Fetches /katalog/{crd} from the operator. Silently skipped if the
		// operator is not reachable — resource list still works without it.
		if !noHealth && operatorURL != "" {
			if stat := fetchCRDHealthStat(operatorURL, crd.Name); stat != nil {
				printCRDHealthBanner(stat)
			}
		}

		// ── List CRs from the cluster ─────────────────────────────────────────
		listNamespace := namespace
		if !crd.Namespaced {
			listNamespace = "" // cluster-scoped — namespace is meaningless
		} else if allNamespaces {
			listNamespace = "" // empty = across all namespaces
		}

		var resourceIface dynamic.ResourceInterface
		if listNamespace != "" {
			resourceIface = clients.Dynamic.Resource(crd.GVR).Namespace(listNamespace)
		} else {
			resourceIface = clients.Dynamic.Resource(crd.GVR)
		}

		list, err := resourceIface.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("listing %s: %w", crd.Plural, err)
		}

		if len(list.Items) == 0 {
			inspect.PrintInfo(fmt.Sprintf("No %s resources found.", crd.Kind))
			return nil
		}

		// ── Table ─────────────────────────────────────────────────────────────
		showNamespace := crd.Namespaced && (allNamespaces || namespace == "")

		header := []string{"NAME", "STATUS", "AGE"}
		if showNamespace {
			header = []string{"NAMESPACE", "NAME", "STATUS", "AGE"}
		}

		rows := make([][]string, 0, len(list.Items))
		for _, item := range list.Items {
			status := inspect.ExtractStatus(&item)
			age := inspect.HumanAge(item.GetCreationTimestamp())

			if showNamespace {
				rows = append(rows, []string{item.GetNamespace(), item.GetName(), status, age})
			} else {
				rows = append(rows, []string{item.GetName(), status, age})
			}
		}

		inspect.PrintTable(os.Stdout, header, rows)
		fmt.Printf("\n\033[90m%d resource(s)\033[0m\n", len(list.Items))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringP("namespace", "n", "", "Namespace to list resources in")
	getCmd.Flags().BoolP("all-namespaces", "A", false, "List across all namespaces")
	getCmd.Flags().String("operator-url", "http://localhost:8080",
		"Orkestra operator URL for health banner (empty = skip)")
	getCmd.Flags().Bool("no-health", false, "Skip the operator health banner")
}

// ── Health banner ─────────────────────────────────────────────────────────────

// fetchCRDHealthStat fetches /katalog/{crd} from the operator.
// Returns nil silently on any error — the resource list works without it.
func fetchCRDHealthStat(operatorURL, crdName string) *crdStat {
	client := &http.Client{Timeout: 2 * time.Second}

	body, err := fetchJSON(client, operatorURL+"/katalog/"+crdName)
	if err != nil {
		return nil
	}

	var stat crdStat
	if err := json.Unmarshal(body, &stat); err != nil {
		return nil
	}

	return &stat
}

// printCRDHealthBanner prints a one-line summary of Orkestra-specific health data.
// This is information kubectl cannot show — it comes from the operator runtime.
//
// Example:
//
//	●  Website  ·  2/2 workers  ·  queue 0/500  ·  3 reconciles  ·  0.0% errors  ·  12 tracked
func printCRDHealthBanner(stat *crdStat) {
	icon := healthIconFromStat(*stat)

	workers := fmt.Sprintf("%d/%d workers", stat.WorkersActive, stat.Workers)

	queue := fmt.Sprintf("queue %d", stat.QueueDepth)
	if stat.MaxQueueDepth > 0 {
		queue = fmt.Sprintf("queue %d/%d", stat.QueueDepth, stat.MaxQueueDepth)
	}

	// Only show consecutive fails when non-zero — avoids noise when healthy
	consec := ""
	if stat.ConsecutiveFails > 0 {
		if stat.DegradeThreshold > 0 {
			consec = fmt.Sprintf("  %s %d/%d consecutive",
				inspect.HealthIcon("pending"),
				stat.ConsecutiveFails, stat.DegradeThreshold)
		} else {
			consec = fmt.Sprintf("  %s %d consecutive fails",
				inspect.HealthIcon("pending"), stat.ConsecutiveFails)
		}
	}

	reconciles := fmt.Sprintf("%d reconciles", stat.TotalReconciles)

	errRate := ""
	if stat.TotalReconciles > 0 {
		errRate = fmt.Sprintf("  %.1f%% errors", stat.ErrorRate*100)
	}

	tracked := fmt.Sprintf("%d tracked", stat.ResourceCount)

	fmt.Printf("%s  %s  ·  %s  ·  %s  ·  %s%s%s  ·  %s\n\n",
		icon,
		inspect.Bold(stat.Kind),
		workers,
		queue,
		reconciles,
		consec,
		errRate,
		tracked,
	)
}
