// Written without controller-gen
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "platform.orkestra.io"
	Version = "v1alpha1"
	Kind    = "Project"
)

var (
	// GroupVersion is the group and version of your API
	GroupVersion = schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}

	// GroupVersionKind is the group, version, and kind of your API
	GroupVersionKind = schema.GroupVersionKind{
		Group:   Group,
		Version: Version,
		Kind:    Kind,
	}

	// API PATH
	APIPath = "/apis"

	// spec.names.plural
	Plural = "projects"

	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Add known types
func addKnownTypes(scheme *runtime.Scheme) error {
	// External version
	scheme.AddKnownTypes(GroupVersion,
		&Project{},
		&ProjectList{},
	)

	// Internal version (required for watch decoding)
	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   Group,
		Version: runtime.APIVersionInternal,
	},
		&Project{},
		&ProjectList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
