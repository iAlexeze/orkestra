// pkg/utils/gvk.go
package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Consistent Keys
type GroupVersionKind string

func GVKFor(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gvk, nil
}

// Mimicks runtime.Object.GetObjectKind().GroupVersionKind()
// https://pkg.go.dev/k8s.io/apimachinery@v0.35.2/pkg/runtime/schema#ObjectKind.GroupVersionKind
func SetGroupVersionKind(group, version, kind string) GroupVersionKind {
	return GroupVersionKind(fmt.Sprintf("%s/%s, Kind=%s", group, version, kind))
}

func SetGroupVersionKindObj(gvk schema.GroupVersionKind) string {
	return SetGroupVersionKind(gvk.Group, gvk.Version, gvk.Kind).String()
}

func (g GroupVersionKind) String() string {
	return string(g)
}
