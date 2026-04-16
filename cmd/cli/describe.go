// cmd/cli/describe.go
package cli

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/pkg/inspect"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/clientcmd"
)

var describeCmd = &cobra.Command{
	Use:   "describe <crd> <name>",
	Short: "Describe a specific CR with full spec, status, and events",
	Long: `Show full details of a named Custom Resource.

Output sections:
  Metadata   Name, namespace, labels, annotations, age, finalizers
  Spec       Full spec as YAML-style indented tree
  Status     Full status as YAML-style indented tree
  Events     Kubernetes events for this resource (last 10)`,

	Example: `  ork describe website my-blog
  ork describe website my-blog -n production
  ork describe platformnamespace payments-prod`,

	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")

		clients, err := inspect.NewClients(kubeconfig)
		if err != nil {
			return err
		}

		crd, err := inspect.DiscoverCRD(clients.Discovery, args[0])
		if err != nil {
			return err
		}

		name := args[1]

		var obj interface{ GetObject() interface{} }
		_ = obj

		var resource interface {
			Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error)
		}
		_ = resource

		// Fetch the CR
		var cr interface{}
		if crd.Namespaced && namespace != "" {
			result, err := clients.Dynamic.Resource(crd.GVR).Namespace(namespace).Get(
				context.Background(), name, metav1.GetOptions{},
			)
			if err != nil {
				return fmt.Errorf("getting %s/%s: %w", namespace, name, err)
			}
			cr = result
		} else {
			result, err := clients.Dynamic.Resource(crd.GVR).Get(
				context.Background(), name, metav1.GetOptions{},
			)
			if err != nil {
				return fmt.Errorf("getting %s: %w", name, err)
			}
			cr = result
		}

		u, ok := cr.(interface {
			GetName() string
			GetNamespace() string
			GetCreationTimestamp() metav1.Time
			GetLabels() map[string]string
			GetAnnotations() map[string]string
			GetFinalizers() []string
			GetUID() interface{}
			Object() map[string]interface{}
		})
		if !ok {
			return fmt.Errorf("unexpected type from cluster")
		}

		// ── Metadata ─────────────────────────────────────────────────────────
		inspect.PrintSection("Metadata")
		inspect.PrintField("Name", u.GetName())
		if u.GetNamespace() != "" {
			inspect.PrintField("Namespace", u.GetNamespace())
		}
		inspect.PrintField("Kind", crd.Kind)
		inspect.PrintField("Group", crd.Group)
		inspect.PrintField("Version", crd.Version)
		inspect.PrintField("Age", inspect.HumanAge(u.GetCreationTimestamp()))

		if len(u.GetLabels()) > 0 {
			inspect.PrintField("Labels", "")
			for k, v := range u.GetLabels() {
				fmt.Printf("  %-30s %s\n", k+":", v)
			}
		}

		if len(u.GetAnnotations()) > 0 {
			inspect.PrintField("Annotations", "")
			for k, v := range u.GetAnnotations() {
				// Skip the reconcile trigger annotation — it's noisy
				if k == inspect.ReconcileAnnotation {
					continue
				}
				fmt.Printf("  %-30s %s\n", k+":", v)
			}
		}

		if len(u.GetFinalizers()) > 0 {
			inspect.PrintField("Finalizers", "")
			for _, f := range u.GetFinalizers() {
				fmt.Printf("  - %s\n", f)
			}
		}

		// ── Spec ─────────────────────────────────────────────────────────────
		if spec, ok := u.Object()["spec"].(map[string]interface{}); ok && len(spec) > 0 {
			inspect.PrintSection("Spec")
			inspect.PrintNestedMap(spec, 0)
		}

		// ── Status ────────────────────────────────────────────────────────────
		if status, ok := u.Object()["status"].(map[string]interface{}); ok && len(status) > 0 {
			inspect.PrintSection("Status")
			inspect.PrintNestedMap(status, 0)
		}

		// ── Events ────────────────────────────────────────────────────────────
		inspect.PrintSection("Events")

		ns := u.GetNamespace()
		if ns == "" {
			ns = metav1.NamespaceAll
		}

		// Field selector: involvedObject.name=<name> involvedObject.kind=<Kind>
		fieldSelector := fields.AndSelectors(
			fields.OneTermEqualSelector("involvedObject.name", name),
			fields.OneTermEqualSelector("involvedObject.kind", crd.Kind),
		).String()

		events, err := clients.Core.CoreV1().Events(ns).List(
			context.Background(),
			metav1.ListOptions{FieldSelector: fieldSelector},
		)
		if err != nil || len(events.Items) == 0 {
			inspect.PrintInfo("  No events found.")
			return nil
		}

		// Show the last 10 events, most recent last (Events are already sorted)
		start := 0
		if len(events.Items) > 10 {
			start = len(events.Items) - 10
		}
		recent := events.Items[start:]

		rows := make([][]string, 0, len(recent))
		for _, ev := range recent {
			typeIcon := inspect.HealthIcon("ready")
			if ev.Type == string(corev1.EventTypeWarning) {
				typeIcon = inspect.HealthIcon("error")
			}

			rows = append(rows, []string{
				typeIcon + " " + ev.Type,
				ev.Reason,
				truncate(ev.Message, 60),
				inspect.HumanAge(ev.LastTimestamp),
			})
		}

		fmt.Println()
		inspect.PrintTable(nil, []string{"TYPE", "REASON", "MESSAGE", "AGE"}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
	describeCmd.Flags().StringP("namespace", "n", "", "Namespace of the resource")
}

// truncate shortens a string to max chars, adding "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// getCurrentNamespace returns:
//   - the namespace from --namespace flag (if provided)
//   - otherwise the namespace from the current kubeconfig context
//   - "" for cluster-scoped CRDs
func getCurrentNamespace(kubeconfig string, crd *inspect.CRDInfo, flagNS string) (string, error) {
	if !crd.Namespaced {
		return "", nil
	}

	// If user explicitly passed -n, use it
	if flagNS != "" {
		return flagNS, nil
	}

	// Otherwise load namespace from kubeconfig
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

	ns, _, err := config.Namespace()
	if err != nil {
		return "", err
	}

	return ns, nil
}
