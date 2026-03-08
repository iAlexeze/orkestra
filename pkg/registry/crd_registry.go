package registry

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
func buildCRDs() []crd {
	crds := []crd{
		newCRD(
			&projectTypev1.Project{},
			&projectTypev1.ProjectList{},
			cRDInfoFrom(
				projectTypev1.Group,
				projectTypev1.Version,
				projectTypev1.Kind,
				projectTypev1.APIPath,
				projectTypev1.NamePlural,
				"default",
				false,
			),
		    projectTypev1.AddToScheme,
			func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewProjectReconciler(inf, ev)
			},
		),
		newCRD(
			&managednsTypeV1.ManagedNamespace{},
			&managednsTypeV1.ManagedNamespaceList{},
			cRDInfoFrom(
				managednsTypeV1.Group,
				managednsTypeV1.Version,
				managednsTypeV1.Kind,
				managednsTypeV1.APIPath,
				managednsTypeV1.NamePlural,
				"default", // This will be ignored since it is clusterscoped
				true,
			),
			managednsTypeV1.AddToScheme,
			func(kube *kubeclient.Kubeclient, inf cache.SharedIndexInformer, ev *event.Event) domain.Reconciler {
				return reconciler.NewManagedNamespaceReconciler(kube, inf, ev)
			},
		),
	}

	return crds
}
