package initialize

import (
	"time"

	managednsTypeV1 "github.com/ialexeze/orkestra/api/types/managedNamespace/v1alpha1"
	projectTypev1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/pkg/event"
	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/reconciler"
	"k8s.io/client-go/tools/cache"
)

// buildCRDs returns a list of CRDs.
//
// You can add as many CRDs as needed following the same pattern
func BuildCRDRegistryFromGo() []CRDEntry {
	return []CRDEntry{
		{
			Name:             "project",
			Enabled:          true,
			ObjectGoMode:     &projectTypev1.Project{},
			ListObjectGoMode: &projectTypev1.ProjectList{},
			Group:            projectTypev1.Group,
			Kind:             projectTypev1.Kind,
			Version:          projectTypev1.Version,
			APIPath:          projectTypev1.APIPath,
			Plural:           projectTypev1.Plural,
			Namespace:        "project",
			Namespaced:       true,
			Workers:          2,
			Resync:           10 * time.Minute, // Has to be in time.Duration
			Scheme:           projectTypev1.AddToScheme,
			Reconciler: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewProjectReconciler(inf, ev)
			},
			Description: `
				Defines an isolated logical boundary for applications, teams, or workloads.
				Similar in concept to ArgoCD Projects, it provides a high‑level grouping
				mechanism for managing access, policies, and resource scoping.
			`,
		},

		{
			Name:             "managednamespace",
			Enabled:          true,
			ObjectGoMode:     &managednsTypeV1.ManagedNamespace{},
			ListObjectGoMode: &managednsTypeV1.ManagedNamespaceList{},
			Group:            managednsTypeV1.Group,
			Kind:             managednsTypeV1.Kind,
			Version:          managednsTypeV1.Version,
			APIPath:          managednsTypeV1.APIPath,
			Plural:           managednsTypeV1.Plural,
			Namespace:        "default", // ignored because cluster-scoped
			Namespaced:       false,     // or false depending on your CRD
			Scheme:           managednsTypeV1.AddToScheme,
			Workers:          1, // Use default resync
			DependsOn:        []string{"project"},
			Reconciler: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewManagedNamespaceReconciler(kube, inf, ev)
			},
			Description: `
				Represents a namespace managed by Orkestra. Useful for provisioning
				standardized namespaces with quotas, network policies, RBAC bindings,
				and other baseline configurations required for multi‑tenant platforms.
			`,
		},
		{
			Name: "applications",
			Description: `
				Legacy Application CRD. Retained for compatibility and future migration
				scenarios but disabled by default. Can be re‑enabled when transitioning
				older workloads or performing controlled rollouts.
			`,
			Workers:    1,
			Resync:     10 * time.Minute,
			Plural:     "applications",
			Namespaced: true,
			Namespace:  "application",
			Group:      "platform.orkestra.io",
			Version:    "v1alpha1",
			Kind:       "Application",
			DependsOn: []string{
				"project", "managednamespace",
			},
		},
	}
}
