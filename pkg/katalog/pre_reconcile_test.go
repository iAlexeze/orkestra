package katalog

import (
	"context"
	// "testing"

	"github.com/orkspace/orkestra/domain"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	// "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type preReconcileTest struct {
	ctx                    context.Context
	katalog                *Katalog
	prSentinels            []string
	object                 metav1.Object
	enqueuGateSentinels    []string
	reconcileGateSentinels []string
	entry                  *orktypes.CRDEntry
}

func newCRSkeleton(crd *orktypes.CRDEntry, labels, annotations, spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": crd.APIVersion(),
			"kind":       crd.Kind(),
			"metadata": map[string]interface{}{
				"labels":      labels,
				"annotations": annotations,
			},
			"spec": spec,
		},
	}
}

func domObj(obj interface{}) domain.Object {
	domObj, ok := domain.ToDomainObject(obj)
	if !ok {
		return nil
	}
	return domObj
}

// katalogWithPreReconcile returns the katalog and sentinels
func katalogWithPreReconcile(pr orktypes.PreReconcileConfig) (rec *preReconcileTest) {
	kat := &Katalog{
		enabledCRDs: map[string]orktypes.CRDEntry{
			"app": {
				APITypes: orktypes.APITypes{
					Kind:    "Application",
					Version: "v1",
					Group:   "test.orkestra.katalog",
				},
				OperatorBox: orktypes.OperatorBoxConfig{
					PreReconcile: &pr,
				},
			},
		},
	}
	entry := rec.katalog.enabledCRDs["app"]
	if !pr.Empty() {
		rec.prSentinels = entry.OperatorBox.PreReconcile.Sentinels
		if pr.HasEnqueueGate() {
			rec.enqueuGateSentinels = pr.EnqueueGate.Sentinels
		}
		if pr.HasReconcileGate() {
			rec.enqueuGateSentinels = pr.EnqueueGate.Sentinels
		}
	}
	rec.entry = &entry
	rec.ctx = context.Background()
	rec.katalog = kat
	return rec
}

// func TestEvaluatePreReconcile_Nil(t *testing.T) {
// 	k := katalogWithPreReconcile(orktypes.PreReconcileConfig{})

// 	ob, ok := domain.ToDomainObject(newCRSkeleton(k.entry, nil, nil, nil))

// 	allowed, _ := k.katalog.EvaluatePreReconcile(k.ctx, k.entry.GVKString(), domObj(ob), nil, nil)

// 	assert.True(t, allowed)
// }
