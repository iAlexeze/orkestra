package initialize

import (
	"fmt"

	applicationTypev1 "github.com/ialexeze/orkestra/api/types/application/v1alpha1"
	managednsTypeV1 "github.com/ialexeze/orkestra/api/types/managedNamespace/v1alpha1"
	projectTypev1 "github.com/ialexeze/orkestra/api/types/project/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Register Runtime Objects
func RegisterRuntimeObjects() {
	// Project
	ObjectRegistry[projectTypev1.GroupVersionKind] = func() runtime.Object { return &projectTypev1.Project{} }
	ListRegistry[projectTypev1.GroupVersionKind] = func() runtime.Object { return &projectTypev1.ProjectList{} }

	// ManagedNamespace
	ObjectRegistry[managednsTypeV1.GroupVersionKind] = func() runtime.Object { return &managednsTypeV1.ManagedNamespace{} }
	ListRegistry[managednsTypeV1.GroupVersionKind] = func() runtime.Object { return &managednsTypeV1.ManagedNamespaceList{} }

	// Application
	ObjectRegistry[applicationTypev1.GroupVersionKind] = func() runtime.Object { return &applicationTypev1.Application{} }
	ListRegistry[applicationTypev1.GroupVersionKind] = func() runtime.Object { return &applicationTypev1.ApplicationList{} }

	// ...
}

// Register all CRDs
func RegisterScheme(scheme *runtime.Scheme) (*runtime.Scheme, error) {
	// Project
	if err := projectTypev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register Projects scheme %v", err)
	}

	// ManagedNamespace
	if err := managednsTypeV1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register ManagedNamespace scheme %v", err)
	}

	// Application
	if err := applicationTypev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to register Application scheme %v", err)
	}

	// ...

	return scheme, nil

}
