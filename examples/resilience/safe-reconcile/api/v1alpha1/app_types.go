//go:build ignore

// api/v1alpha1/app_types.go
//
// The Go type for the App CRD.
// Used by the typed hook — gives type-safe access to spec fields.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AppConfig holds optional configuration for the App.
// When omitted from the CR, this field is nil — accessing it without a nil
// check panics. That is the intentional bug this example demonstrates.
type AppConfig struct {
	Endpoint string `json:"endpoint,omitempty"`
}

// AppSpec defines the desired state of App
type AppSpec struct {
	// Image is the container image to run
	Image string `json:"image"`

	// Replicas is the desired replica count
	Replicas int32 `json:"replicas,omitempty"`

	// Config is optional. When omitted, this pointer is nil.
	Config *AppConfig `json:"config,omitempty"`
}

// AppStatus defines the observed state of App
type AppStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// App is the Schema for the apps API
type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSpec   `json:"spec,omitempty"`
	Status AppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppList contains a list of App
type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

var (
	GroupVersion  = schema.GroupVersion{Group: "safe.demo.orkestra.io", Version: "v1alpha1"}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&App{},
		&AppList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

// DeepCopyObject implements runtime.Object
func (a *App) DeepCopyObject() runtime.Object {
	if a == nil {
		return nil
	}
	out := new(App)
	a.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all fields into out
func (a *App) DeepCopyInto(out *App) {
	*out = *a
	out.TypeMeta = a.TypeMeta
	a.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = a.Spec
	if a.Spec.Config != nil {
		cfg := *a.Spec.Config
		out.Spec.Config = &cfg
	}
	a.Status.DeepCopyInto(&out.Status)
}

func (s *AppStatus) DeepCopyInto(out *AppStatus) {
	*out = *s
	if s.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(s.Conditions))
		copy(out.Conditions, s.Conditions)
	}
}

// AppList DeepCopyObject
func (al *AppList) DeepCopyObject() runtime.Object {
	if al == nil {
		return nil
	}
	out := new(AppList)
	*out = *al
	if al.Items != nil {
		out.Items = make([]App, len(al.Items))
		for i := range al.Items {
			al.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
	return out
}
