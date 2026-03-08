package registry

import (
	"reflect"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/reconciler"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)


const (
	DefaultAPIPath = "/apis"
)

type crd struct {
	Object     runtime.Object
	ListObject runtime.Object
	Scheme func(*runtime.Scheme) error
	Reconciler reconciler.NewReconcilerFunc
	Info       CRDInfo
}

var resourceTypeMap = map[reflect.Type]string{}

type CRDInfo struct {
	Kind             string                  // Required by Registry
	Group            string                  // Required if GroupVersion is not specified
	Version          string                  // Required if GroupVersion is not specified
	GroupVersion     *schema.GroupVersion    // Optional (can be used if Group and Version are not specified)
	GroupVersionKind schema.GroupVersionKind //	Useful for some manipulations and Required by Registry
	NamePlural       string
	ClusterScoped    bool // Required for cluster-scoped resources
	APIPath          string
	Namespace        string
}

