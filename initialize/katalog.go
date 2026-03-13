// initialize/katalog.go
package initialize

import (
	"time"

	applicationTypev1 "github.com/ialexeze/orkestra/api/types/application/v1alpha1"
	managednsTypeV1 "github.com/ialexeze/orkestra/api/types/managedNamespace/v1alpha1"
	projectTypev1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/reconciler/hooks"
)

// BuildKatalogFromGo returns the full list of CRD entries for Go mode.
//
// Rules:
//   - ReconcilerConfig.Default = true  → GenericReconciler is used automatically.
//     Add a HookFactory if you need lifecycle callbacks. Leave it nil for zero code.
//   - ReconcilerConfig.Default = false → You must provide a Constructor.
//     Constructor is called after manager starts — kube and ev are live.
//   - DependsOn controls startup and shutdown order via the dependency graph.
//     Validation rejects missing or cyclic dependencies at startup.
//   - Disabled CRDs (Enabled: false) never enter the runtime — safe to leave in place.
func BuildKatalogFromGo() []CRDEntry {
	return []CRDEntry{

		// ── PROJECT ───────────────────────────────────────────────────────────
		// Root CRD — no dependencies.
		// Critical: controller is marked degraded if Project workers fail.
		// Uses GenericReconciler with hooks for custom reconcile logic.
		{
			Name:             "project",
			Enabled:          true,
			Critical:         true, // controller health reflects this CRD's health
			ObjectGoMode:     &projectTypev1.Project{},
			ListObjectGoMode: &projectTypev1.ProjectList{},
			Group:            projectTypev1.Group,
			Kind:             projectTypev1.Kind,
			Version:          projectTypev1.Version,
			APIPath:          projectTypev1.APIPath,
			Plural:           projectTypev1.Plural,
			Namespace:        "default",
			Namespaced:       true,
			Workers:          3,
			Resync:           10 * time.Minute,
			Scheme:           projectTypev1.AddToScheme,
			DependsOn:        []string{},

			ReconcilerConfig: ReconcilerConfig{
				Default: true,
				Finalizers: []string{
					"my-default-finalizer/platform-katalog",
				},
				// HookFactory is optional — implement only the hooks you need.
				// OnReconcile, OnDelete, OnNotFound are all independently optional.
				HookFactory: func() domain.AnyReconcileHooks {
					return hooks.ProjectHooks()
				},
			},

			Queue: Queue{
				MaxQueueDepth: 1000,
			},

			Description: `
				Defines an isolated logical boundary for applications, teams, or workloads.
				Similar in concept to ArgoCD Projects — provides a high-level grouping
				mechanism for access control, policies, and resource scoping.
			`,
		},

		// ── MANAGED NAMESPACE ─────────────────────────────────────────────────
		// Depends on Project — Orkestra will not start ManagedNamespace workers
		// until Project CRD is confirmed present and workers have started.
		// Pure default reconciler — no hooks, no custom code.
		{
			Name:             "managednamespace",
			Enabled:          true,
			Critical:         false,
			ObjectGoMode:     &managednsTypeV1.ManagedNamespace{},
			ListObjectGoMode: &managednsTypeV1.ManagedNamespaceList{},
			Group:            managednsTypeV1.Group,
			Kind:             managednsTypeV1.Kind,
			Version:          managednsTypeV1.Version,
			APIPath:          managednsTypeV1.APIPath,
			Plural:           managednsTypeV1.Plural,
			Namespaced:       false, // cluster-scoped
			Scheme:           managednsTypeV1.AddToScheme,
			Workers:          2,
			DependsOn:        []string{"project"},
			// No Resync — uses Orkestra-level default (set via DEFAULT_RESYNC env)

			ReconcilerConfig: ReconcilerConfig{
				Default: true,
				// No HookFactory — GenericReconciler runs silently:
				// finalizers managed, events emitted, metrics recorded. Zero code.
			},

			Queue: Queue{
				MaxQueueDepth: 2000,
			},

			Description: `
				Represents a namespace managed by Orkestra. Provisions standardized
				namespaces with quotas, network policies, RBAC bindings, and baseline
				configurations for multi-tenant platforms.
			`,
		},

		// ── APPLICATION ───────────────────────────────────────────────────────
		// Depends on both Project and ManagedNamespace.
		// Uses the shared default workqueue (Queue.Default: true) instead of
		// a per-CRD queue. Suitable for lower-volume CRDs or when queue
		// isolation is not required.
		// Custom reconciler via Constructor — full control over reconcile logic.
		{
			Name:             "application",
			Enabled:          true,
			Critical:         false,
			ObjectGoMode:     &applicationTypev1.Application{},
			ListObjectGoMode: &applicationTypev1.ApplicationList{},
			Group:            "platform.orkestra.io",
			Version:          "v1alpha1",
			Kind:             "Application",
			Plural:           "applications",
			Namespace:        "default",
			Namespaced:       true,
			Scheme:           applicationTypev1.AddToScheme,
			Workers:          2,
			Resync:           3 * time.Second,
			DependsOn:        []string{"project", "managednamespace"},

			ReconcilerConfig: ReconcilerConfig{
				// Default: false, // custom reconciler — Constructor is required
				Default: true,
				Finalizers: []string{
					"my-finalizer/application",
				},
			},

			Queue: Queue{
				Default:       true, // uses shared default queue, not per-CRD queue
				MaxQueueDepth: 500,
			},

			Description: `
				Represents a deployable application workload managed by Orkestra.
				Depends on both Project and ManagedNamespace — ensures namespace
				and project boundaries exist before application resources are reconciled.
			`,
		},

		// ── EXAMPLE: DISABLED CRD ─────────────────────────────────────────────
		// Set Enabled: false to exclude from runtime without deleting the entry.
		// Existing CRs with Orkestra finalizers will be stuck if disabled —
		// only disable CRDs with no live resources, or strip finalizers first.
		//
		// {
		//     Name:    "legacyresource",
		//     Enabled: false,
		//     ...
		// },
	}
}
