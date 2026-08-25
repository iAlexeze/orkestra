package simulate

import (
	"context"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

// loopKube is the kubeclient capability the reconcile loop requires beyond
// the standard interface: cycle tracking, op collection, and the
// deployment-ready helper needed when no controller-manager runs.
type loopKube interface {
	kubeclient.Interface
	AdvanceCycle()
	OpsForCycle(cycle int) []Op
	Ops() []Op
	MarkDeploymentReady(namespace, name string)
}

// runLoop runs the reconciler for up to maxCycles and returns the aggregate
// result. Notes must be appended by the caller.
func runLoop(ctx context.Context, r domain.Reconciler, kube loopKube, key string, maxCycles int) *Result {
	result := &Result{}
	var prevCycleOps []Op

	for cycle := 1; cycle <= maxCycles; cycle++ {
		kube.AdvanceCycle()

		cycleResult := CycleResult{Cycle: cycle}
		ns, name, _ := cache.SplitMetaNamespaceKey(key)
		_, cycleResult.Error = r.Reconcile(ctx, domain.Request{
			Key:            key,
			NamespacedName: apitypes.NamespacedName{Namespace: ns, Name: name},
		})
		cycleResult.Ops = kube.OpsForCycle(cycle)
		result.Cycles = append(result.Cycles, cycleResult)

		// Mark Deployments created this cycle as ready so the reconciler
		// can progress through state transitions on the next cycle
		// (no controller-manager runs in fake or envtest mode).
		for _, op := range cycleResult.Ops {
			if op.Verb == "create" && op.Resource == "deployments" {
				kube.MarkDeploymentReady(op.Namespace, op.Name)
			}
		}

		if !result.Steady && cycle > 1 && opsMatch(cycleResult.Ops, prevCycleOps) {
			result.Steady = true
			result.SteadyAt = cycle
		}
		prevCycleOps = cycleResult.Ops
	}

	result.AllOps = kube.Ops()
	return result
}

// compile-time check that *FakeKubeclient satisfies loopKube.
var _ loopKube = (*FakeKubeclient)(nil)

// compile-time check that cache.SharedIndexInformer is accessible from cache.
var _ cache.SharedIndexInformer = (*fakeInformer)(nil)
