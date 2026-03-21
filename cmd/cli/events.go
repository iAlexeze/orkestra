// cmd/cli/events.go
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/ialexeze/orkestra/pkg/inspect"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

var eventsCmd = &cobra.Command{
	Use:   "events <crd> [name]",
	Short: "Show Kubernetes events for a CRD type or a specific CR",
	Long: `List Kubernetes events for Orkestra-managed resources.

Without a name, shows events for all CRs of the given type.
With a name, shows events for only that specific CR.

Events include:
  Reconciled          Successful reconcile
  ReconcileError      Failed reconcile — includes the error message
  FinalizerAdded      Finalizer was added to the CR
  FinalizerRemoved    Finalizer was removed (during deletion)
  Deleting            CR deletion is in progress

Useful for debugging reconcile failures without reading operator logs.`,

	Example: `  ork events website
  ork events website my-blog
  ork events website my-blog -n production
  ork events platformnamespace --tail 20`,

	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")
		tail, _ := cmd.Flags().GetInt("tail")

		clients, err := inspect.NewClients(kubeconfig)
		if err != nil {
			return err
		}

		crd, err := inspect.DiscoverCRD(clients.Discovery, args[0])
		if err != nil {
			return err
		}

		ctx := context.Background()

		// Resolve the namespace for event listing
		listNamespace := namespace
		if !crd.Namespaced {
			listNamespace = metav1.NamespaceAll
		} else if listNamespace == "" {
			listNamespace = metav1.NamespaceAll
		}

		// Build field selector
		// With a specific name: filter by name and kind
		// Without a name: filter by kind only
		var fieldSelector string
		if len(args) == 2 {
			fieldSelector = fields.AndSelectors(
				fields.OneTermEqualSelector("involvedObject.name", args[1]),
				fields.OneTermEqualSelector("involvedObject.kind", crd.Kind),
			).String()
			fmt.Printf("Events for %s/%s:\n\n", crd.Kind, args[1])
		} else {
			fieldSelector = fields.OneTermEqualSelector("involvedObject.kind", crd.Kind).String()
			fmt.Printf("Events for all %s resources:\n\n", crd.Kind)
		}

		events, err := clients.Core.CoreV1().Events(listNamespace).List(
			ctx,
			metav1.ListOptions{FieldSelector: fieldSelector},
		)
		if err != nil {
			return fmt.Errorf("listing events: %w", err)
		}

		if len(events.Items) == 0 {
			inspect.PrintInfo("No events found.")
			return nil
		}

		// Apply --tail limit (take from end — most recent events)
		items := events.Items
		if tail > 0 && len(items) > tail {
			items = items[len(items)-tail:]
		}

		// Build rows
		header := []string{"TYPE", "REASON", "OBJECT", "MESSAGE", "AGE"}
		rows := make([][]string, 0, len(items))

		for _, ev := range items {
			typeIcon := inspect.HealthIcon("ready")
			if ev.Type == string(corev1.EventTypeWarning) {
				typeIcon = inspect.HealthIcon("error")
			}

			object := ev.InvolvedObject.Name
			if ev.InvolvedObject.Namespace != "" {
				object = ev.InvolvedObject.Namespace + "/" + ev.InvolvedObject.Name
			}

			rows = append(rows, []string{
				typeIcon + " " + ev.Type,
				ev.Reason,
				object,
				truncate(ev.Message, 55),
				inspect.HumanAge(ev.LastTimestamp),
			})
		}

		inspect.PrintTable(os.Stdout, header, rows)

		if len(events.Items) > len(items) {
			fmt.Printf("\n%s (showing last %d of %d events — use --tail to see more)\n",
				"\033[90m", tail, len(events.Items))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.Flags().StringP("namespace", "n", "", "Namespace (namespaced CRDs only)")
	eventsCmd.Flags().Int("tail", 25, "Number of most recent events to show (0 = all)")
}
