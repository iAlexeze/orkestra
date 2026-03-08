// Written without controller-gen
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "platform.ialexeze.io"
	Version = "v1alpha1"
	Kind    = "Project"
)

var (
	GroupVersion = schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}

	// API PATH
	APIPath = "/apis"

	// spec.names.plural
	NamePlural = "projects"

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
