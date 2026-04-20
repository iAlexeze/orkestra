// pkg/reconciler/children_gvr.go
package reconciler

import (
	"github.com/orkspace/orkestra/pkg/katalog"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ── Child resource GVRs ───────────────────────────────────────────────────
// These are the GVRs for every resource type the OrkestraRegistry creates.
// Used to read back child resources after reconcile completes.
// When you add a new resource, make it avaialble here to be read by orkestra
var (
	deploymentGVR     = gvrOrPanic("deployment")
	serviceGVR        = gvrOrPanic("service")
	secretGVR         = gvrOrPanic("secret")
	configMapGVR      = gvrOrPanic("configmap")
	jobGVR            = gvrOrPanic("job")
	cronJobGVR        = gvrOrPanic("cronjob")
	podGVR            = gvrOrPanic("pod")
	serviceAccountGVR = gvrOrPanic("serviceaccount")
	statefulSetGVR    = gvrOrPanic("statefulset")
	ingressGVR        = gvrOrPanic("ingress")
	pvcGVR            = gvrOrPanic("persistentvolumeclaim")
	pvGVR             = gvrOrPanic("persistentvolume")
	hpaGVR            = gvrOrPanic("horizontalpodautoscaler")
	pdbGVR            = gvrOrPanic("poddisruptionbudget")
)

// gvrOrPanic is a small helper for static initialization.
// Built-ins are guaranteed to exist in katalog.builtInRegistry.
func gvrOrPanic(kind string) schema.GroupVersionResource {
	gvr, ok := katalog.GVRForBuiltIn(kind)
	if !ok {
		panic("reconciler: built-in kind not found: " + kind)
	}
	return gvr
}
