// cmd/cli/reconcile.go
package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/orkspace/orkestra/pkg/inspect"
	"github.com/spf13/cobra"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ─────────────────────────────────────────────────────────────────────────────
// PARENT COMMAND — ork reconcile
//
// This command acts as a router:
//   - If the first argument is "all", run the "all" subcommand.
//   - Otherwise, forward args to the hidden <crd> subcommand.
//
// This preserves the UX:
//
//	ork reconcile website
//	ork reconcile website my-blog
//	ork reconcile all
//
// ─────────────────────────────────────────────────────────────────────────────
var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Trigger reconciliation for Orkestra-managed resources",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// No args → show help
		if len(args) == 0 {
			return cmd.Help()
		}

		// If first arg is "all", run the all subcommand
		if args[0] == "all" {
			return reconcileAllCmd.RunE(cmd, args[1:])
		}

		// Otherwise treat args as <crd> [name]
		return reconcileCRDCmd.RunE(cmd, args)
	},
	Long: `Trigger reconciliation by patching the orkestra.konductor.io/reconcile-at
annotation on one or more Custom Resources.

Examples:
  ork reconcile website
  ork reconcile website my-blog
  ork reconcile all`,
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBCOMMAND — ork reconcile <crd> [name]
// Hidden from help, but receives all unknown args from parent.
// ─────────────────────────────────────────────────────────────────────────────
var reconcileCRDCmd = &cobra.Command{
	Use:    "<crd> [name]",
	Hidden: true, // critical — prevents Cobra from treating <crd> as literal
	Args:   cobra.RangeArgs(1, 2),
	Short:  "Reconcile one or all CRs of a given CRD type",
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

		ctx := context.Background()

		// Single CR
		if len(args) == 2 {
			name := args[1]
			ns := resolveNamespace(crd, namespace)

			fmt.Printf("Triggering reconcile for %s/%s... ", crd.Kind, name)
			err := inspect.TriggerReconcile(ctx, clients.Dynamic, crd.GVR, ns, name)
			if err != nil {
				inspect.PrintError(err.Error())
				return err
			}
			inspect.PrintSuccess("")
			return nil
		}

		// All CRs of this type
		ns := resolveNamespace(crd, namespace)
		fmt.Printf("Triggering reconcile for all %s resources...\n\n", crd.Kind)

		results, err := inspect.TriggerReconcileAll(ctx, clients.Dynamic, crd.GVR, ns)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			inspect.PrintInfo("No resources found.")
			return nil
		}

		failures := 0
		for _, r := range results {
			label := qualifiedName(r.Namespace, r.Name)
			fmt.Printf("  %-45s", label)
			if r.Error != nil {
				inspect.PrintError(r.Error.Error())
				failures++
			} else {
				inspect.PrintSuccess("")
			}
		}

		fmt.Println()
		if failures > 0 {
			inspect.PrintWarning(fmt.Sprintf("%d/%d resources failed to trigger", failures, len(results)))
		} else {
			inspect.PrintSuccess(fmt.Sprintf("Triggered %d resources", len(results)))
		}

		return nil
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBCOMMAND — ork reconcile all
// ─────────────────────────────────────────────────────────────────────────────
var reconcileAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Reconcile every Orkestra-managed resource across all CRD types",
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		sleepDuration, _ := cmd.Flags().GetDuration("sleep")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		ns, _ := cmd.Flags().GetString("namespace")

		clients, err := inspect.NewClients(kubeconfig)
		if err != nil {
			return err
		}

		ctx := context.Background()

		crds, err := inspect.DiscoverOrkestraCRDs(clients.Discovery)
		if err != nil {
			return err
		}

		if len(crds) == 0 {
			inspect.PrintInfo("No Orkestra-managed CRDs found.")
			return nil
		}

		if dryRun {
			fmt.Println(inspect.Bold("Dry run — no changes will be made\n"))
		}

		totalResources := 0
		totalFailures := 0

		for i, crd := range crds {
			fmt.Printf("%s[%d/%d] %s%s\n", "\033[1m", i+1, len(crds), crd.Kind, "\033[0m")

			if dryRun {
				inspect.PrintInfo(fmt.Sprintf("  Would reconcile all %s resources", crd.Kind))
				fmt.Println()
				continue
			}

			results, err := inspect.TriggerReconcileAll(ctx, clients.Dynamic, crd.GVR, ns)
			if err != nil {
				inspect.PrintError(fmt.Sprintf("  Failed to trigger %s: %v", crd.Kind, err))
				fmt.Println()
				continue
			}

			if len(results) == 0 {
				inspect.PrintInfo("  No resources found.")
			} else {
				for _, r := range results {
					label := "  " + qualifiedName(r.Namespace, r.Name)
					fmt.Printf("  %-45s", label)
					if r.Error != nil {
						inspect.PrintError(r.Error.Error())
						totalFailures++
					} else {
						inspect.PrintSuccess("")
						totalResources++
					}
				}
			}

			if i < len(crds)-1 {
				fmt.Printf("  %sWaiting %s before next CRD...%s\n", "\033[90m", sleepDuration, "\033[0m")
				time.Sleep(sleepDuration)
			}

			fmt.Println()
		}

		fmt.Println(strings.Repeat("─", 50))
		if totalFailures > 0 {
			inspect.PrintWarning(fmt.Sprintf("Reconciled %d resources across %d CRDs (%d failures)", totalResources, len(crds), totalFailures))
		} else {
			inspect.PrintSuccess(fmt.Sprintf("Reconciled %d resources across %d CRDs", totalResources, len(crds)))
		}

		return nil
	},
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(reconcileCmd)

	reconcileCmd.AddCommand(reconcileCRDCmd)
	reconcileCRDCmd.Flags().StringP("namespace", "n", "", "Namespace (namespaced CRDs only)")

	reconcileCmd.AddCommand(reconcileAllCmd)
	reconcileAllCmd.Flags().Duration("sleep", 3*time.Second, "Pause between CRD types")
	reconcileAllCmd.Flags().Bool("dry-run", false, "Print what would be reconciled without making changes")
}

func resolveNamespace(crd *inspect.CRDInfo, flagNamespace string) string {
	if !crd.Namespaced {
		return ""
	}
	return flagNamespace
}

func qualifiedName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}
