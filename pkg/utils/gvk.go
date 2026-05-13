// pkg/utils/gvk.go
package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersionKind is a consistent string key for GVK-keyed maps and logs.
type GroupVersionKind string

// SetGroupVersionKind formats a GVK as the canonical string used internally.
// Mirrors runtime.Object.GetObjectKind().GroupVersionKind().String() output.
func SetGroupVersionKind(group, version, kind string) GroupVersionKind {
	return GroupVersionKind(fmt.Sprintf("%s/%s, Kind=%s", group, version, kind))
}

func SetGroupVersionKindObj(gvk schema.GroupVersionKind) string {
	return SetGroupVersionKind(gvk.Group, gvk.Version, gvk.Kind).String()
}

func (g GroupVersionKind) String() string {
	return string(g)
}
