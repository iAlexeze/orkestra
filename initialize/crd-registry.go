package initialize

import (
	managednsTypeV1 "github.com/ialexeze/multi-crd-controller/pkg/config/api/types/managedNamespace/v1alpha1"
	projectTypev1 "github.com/ialexeze/multi-crd-controller/pkg/config/api/types/project/v1alpha1"
	"github.com/ialexeze/multi-crd-controller/pkg/config/domain"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/event"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/kubeclient"
	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/reconciler"
	"k8s.io/client-go/tools/cache"
)

// buildCRDs returns a list of CRDs.
//
// You can add as many CRDs as needed following the same pattern
func BuildCRDRegistryFromGo() []CRDEntry {
	return []CRDEntry{
		{
			Name:             "project",
			ObjectGoMode:     &projectTypev1.Project{},
			ListObjectGoMode: &projectTypev1.ProjectList{},
			Group:            projectTypev1.Group,
			Kind:             projectTypev1.Kind,
			Version:          projectTypev1.Version,
			APIPath:          projectTypev1.APIPath,
			NamePlural:       projectTypev1.NamePlural,
			Namespace:        "default",
			Namespaced:       true,
			Workers:          2,
			Scheme:           projectTypev1.AddToScheme,
			Reconciler: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewProjectReconciler(inf, ev)
			},
		},

		{
			Name:             "managednamespace",
			ObjectGoMode:     &managednsTypeV1.ManagedNamespace{},
			ListObjectGoMode: &managednsTypeV1.ManagedNamespaceList{},
			Group:            managednsTypeV1.Group,
			Kind:             managednsTypeV1.Kind,
			Version:          managednsTypeV1.Version,
			APIPath:          managednsTypeV1.APIPath,
			NamePlural:       managednsTypeV1.NamePlural,
			Namespace:        "default", // ignored because cluster-scoped
			Namespaced:       false,     // or false depending on your CRD
			Scheme:           managednsTypeV1.AddToScheme,
			Workers:          1,
			DependsOn:        []string{"project"},
			Reconciler: func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewManagedNamespaceReconciler(kube, inf, ev)
			},
		},
	}
}
