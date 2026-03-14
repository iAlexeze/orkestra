// crdkatalog/katalog.go

// KomposeKatalogFromGo returns your CRD entries.
//
// The entries below are EXAMPLES shipped with Orkestra.
// Replace them with your own CRDs before running in production.
package crdkatalog

import (
	"time"

	"github.com/ialexeze/orkestra/domain"
	applicationTypev1 "github.com/ialexeze/orkestra/example-crds/api/types/application/v1alpha1"
	managednsTypeV1 "github.com/ialexeze/orkestra/example-crds/api/types/managedNamespace/v1alpha1"
	projectTypev1 "github.com/ialexeze/orkestra/example-crds/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/reconciler/hooks"
)

// KomposeKatalogFromGo returns the full list of CRD entries for Go mode.
//
// Rules:
//   - ReconcilerConfig.Default = true  → GenericReconciler used automatically.
//     Add a HookFactory for lifecycle callbacks. nil = zero code.
//   - ReconcilerConfig.Default = false → Constructor must be provided.
//     Constructor is called after Orkestra starts — kube and ev are live.
//   - DependsOn controls startup and shutdown order via the dependency graph.
//     Validation rejects missing or cyclic dependencies at startup.
//   - Disabled CRDs (Enabled: false) never enter the runtime — safe to leave in place.
//   - APITypes mirrors the YAML apiTypes block — ork generate reads this in YAML mode.
//     In Go mode it drives setGroupVersionKind() and scheme registration.
func KomposeKatalogFromGo() []initialize.CRDEntry {
	return []initialize.CRDEntry{

		// ── PROJECT ───────────────────────────────────────────────────────────
		// Root CRD — no dependencies.
		// Critical: Orkestra marks itself degraded if Project workers fail.
		// GenericReconciler with hooks — implement only what you need.
		{
			Name:        "project",
			Enabled:     true,
			Critical:    true,
			Namespaced:  true,
			Namespace:   "default",
			Workers:     3,
			Resync:      10 * time.Minute,
			DependsOn:   []string{},
			Description: "Defines an isolated logical boundary for applications, teams, or workloads.",

			// APITypes mirrors crd-katalog.yaml apiTypes block.
			// setGroupVersionKind() derives GroupVersionKind/GroupVersionResource from here.
			APITypes: initialize.APITypes{
				Object:   projectTypev1.Kind,
				List:     projectTypev1.Kind + "List",
				Alias:    "projv1",
				Group:    projectTypev1.Group,
				Version:  projectTypev1.Version,
				Kind:     projectTypev1.Kind,
				Plural:   projectTypev1.Plural,
				APIPath:  projectTypev1.APIPath,
				Location: "github.com/ialexeze/orkestra/example-crds/api/types/project/v1alpha1",
			},

			// Go mode runtime objects — set directly here.
			// YAML mode uses ObjectYamlMode/ListObjectYamlMode from addRuntimeObjects().
			ObjectGoMode:     &projectTypev1.Project{},
			ListObjectGoMode: &projectTypev1.ProjectList{},
			Scheme:           projectTypev1.AddToScheme,

			ReconcilerConfig: initialize.ReconcilerConfig{
				Default: true,
				Finalizers: []string{
					"my-default-finalizer/platform-katalog",
				},
				// HookFactory is optional — implement only the hooks you need.
				// OnReconcile, OnDelete, OnNotFound are all independently optional.
				// nil HookFactory = pure GenericReconciler, zero user code.
				HookFactory: func() domain.AnyReconcileHooks {
					return hooks.ProjectHooks()
				},
			},

			Queue: initialize.Queue{
				MaxQueueDepth: 1000,
			},
		},

		// ── MANAGED NAMESPACE ─────────────────────────────────────────────────
		// Depends on Project — workers will not start until Project is confirmed.
		// Pure default reconciler — no hooks, no custom code whatsoever.
		{
			Name:        "managednamespace",
			Enabled:     true,
			Critical:    false,
			Namespaced:  false, // cluster-scoped
			Workers:     2,
			DependsOn:   []string{"project"},
			Description: "Represents a namespace managed by Orkestra.",
			// No Resync — uses Orkestra-level default (DEFAULT_RESYNC env var)

			APITypes: initialize.APITypes{
				Object:   managednsTypeV1.Kind,
				List:     managednsTypeV1.Kind + "List",
				Alias:    "mnsv1",
				Group:    managednsTypeV1.Group,
				Version:  managednsTypeV1.Version,
				Kind:     managednsTypeV1.Kind,
				Plural:   managednsTypeV1.Plural,
				APIPath:  managednsTypeV1.APIPath,
				Location: "github.com/ialexeze/orkestra/example-crds/api/types/managedNamespace/v1alpha1",
			},

			ObjectGoMode:     &managednsTypeV1.ManagedNamespace{},
			ListObjectGoMode: &managednsTypeV1.ManagedNamespaceList{},
			Scheme:           managednsTypeV1.AddToScheme,

			ReconcilerConfig: initialize.ReconcilerConfig{
				Default: true,
				// No HookFactory — GenericReconciler runs silently:
				// finalizers managed, events emitted, metrics recorded. Zero code.
			},

			Queue: initialize.Queue{
				MaxQueueDepth: 2000,
			},
		},

		// ── APPLICATION ───────────────────────────────────────────────────────
		// Depends on both Project and ManagedNamespace.
		// Uses shared default workqueue (Queue.Default: true) — no per-CRD queue.
		// Default reconciler with finalizer override.
		{
			Name:        "application",
			Enabled:     true,
			Critical:    false,
			Namespaced:  true,
			Namespace:   "default",
			Workers:     2,
			Resync:      3 * time.Second,
			DependsOn:   []string{"project", "managednamespace"},
			Description: "Represents a deployable application workload managed by Orkestra.",

			APITypes: initialize.APITypes{
				Object:  applicationTypev1.Kind,
				List:    applicationTypev1.Kind + "List",
				Alias:   "appv1",
				Group:   applicationTypev1.Group,
				Version: applicationTypev1.Version,
				Kind:    applicationTypev1.Kind,
				Plural:  applicationTypev1.Plural,
				APIPath: applicationTypev1.APIPath,
				// Location: "github.com/ialexeze/orkestra/example-crds/api/types/application/v1alpha1",  // Optional if scheme is already defined
			},

			ObjectGoMode:     &applicationTypev1.Application{},
			ListObjectGoMode: &applicationTypev1.ApplicationList{},
			Scheme:           applicationTypev1.AddToScheme,

			ReconcilerConfig: initialize.ReconcilerConfig{
				Default: true,
				Finalizers: []string{
					"my-finalizer/application",
				},
				// To switch to a custom reconciler:
				// Default: false,
				// Constructor: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				//     return reconciler.NewApplicationReconciler(kube, inf, ev)
				// },
			},

			Queue: initialize.Queue{
				Default:       true, // shared default queue — no per-CRD queue isolation
				MaxQueueDepth: 500,
			},
		},

		// ── EXAMPLE: DISABLED CRD ─────────────────────────────────────────────
		// Set Enabled: false to exclude from runtime without deleting the entry.
		// WARNING: Only disable CRDs with no live resources, or strip Orkestra
		// finalizers first — disabled CRDs with live finalizers cause stuck objects.
		//
		// {
		//     Name:    "legacyresource",
		//     Enabled: false,
		//     APITypes: APITypes{...},
		//     ...
		// },
	}
}
