// // File: api/v1alpha1/groupversion_info.go
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ManagedNamespaceConditionType string

const (
	Active      ManagedNamespaceConditionType = "Active"
	Ready       ManagedNamespaceConditionType = "Ready"
	Failed      ManagedNamespaceConditionType = "Failed"
	Error       ManagedNamespaceConditionType = "Error"
	Warning     ManagedNamespaceConditionType = "Warning"
	Progressing ManagedNamespaceConditionType = "Progressing"
	Reconciled  ManagedNamespaceConditionType = "Reconciled"

	Group   = "platform.ialexeze.io"
	Version = "v1alpha1"
	Kind    = "ManagedNamespace"
)

var (
	// GroupVersion is the group and version of your API
	GroupVersion = schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}

	// API PATH
	APIPath = "/apis"

	// spec.names.plural
	NamePlural = "managednamespaces"

	// SchemeBuilder is used to add Go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds all types in this package to the scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// Add known types
func addKnownTypes(scheme *runtime.Scheme) error {
	// External version
	scheme.AddKnownTypes(GroupVersion,
		&ManagedNamespace{},
		&ManagedNamespaceList{},
	)

	// Internal version (required for watch decoding)
	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   Group,
		Version: runtime.APIVersionInternal,
	},
		&ManagedNamespace{},
		&ManagedNamespaceList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
