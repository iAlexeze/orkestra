package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	Group      = "platform.orkestra.io" // Using same group as existing CRDs
	Version    = "v1alpha1"
	APIPath    = "/apis"
	Kind       = "Application"
	NamePlural = "applications"

	GroupVersion = schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}
	GroupVersionKind = schema.GroupVersionKind{
		Group:   Group,
		Version: Version,
		Kind:    Kind,
	}
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Application{},
		&ApplicationList{},
	)

	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   Group,
		Version: runtime.APIVersionInternal,
	},
		&Application{},
		&ApplicationList{},
	)

	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
