package registry

import (
	"fmt"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/reconciler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// NewCRDRegistry returns a list of CRD data
func NewCRDRegistry() []crd {
	return updateResourceMapAndReturn(buildCRDs())
}

// Define new CRD for consistency in CRD creation
func newCRD(
	obj runtime.Object,
	listObj runtime.Object,
	info CRDInfo,
	scheme func(*runtime.Scheme) error,
	newRec reconciler.NewReconcilerFunc,
) crd {
	return crd{
		Object:     obj,
		ListObject: listObj,
		Info:       info,
		Scheme: scheme,
		Reconciler: newRec,
	}
}

// NewSchemeRegistry returns a new scheme
func NewSchemeRegistry() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()

	// 1. Register built-in Kubernetes types
	metav1.AddToGroupVersion(scheme, metav1.SchemeGroupVersion)

	// 2. Register core Kubernetes types
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	// 3. Register all CRDs
	crds := buildCRDs()
	for _, c := range crds {
		if err := c.Scheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to register %s/%s: %w", c.Info.Group, c.Info.Version, err)
        }
    }

	return scheme, nil
}
