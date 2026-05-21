package simulate

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/katalog"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/reconciler"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

// Result is the output of one simulation run.
type Result struct {
	Cycles   []CycleResult
	Steady   bool
	SteadyAt int // cycle number where steady state was first detected (0 if not reached)
}

// CycleResult is the output of one reconcile cycle.
type CycleResult struct {
	Cycle int
	Ops   []Op
	Error error
}

// Run simulates the operator against an in-memory cluster.
//
// kat is the parsed Katalog.
// crdName is the CRD entry to simulate.
// cr is the CR to reconcile.
// maxCycles is the maximum number of reconcile cycles.
func Run(ctx context.Context, kat *katalog.Katalog, crdName string, cr *unstructured.Unstructured, maxCycles int) (*Result, error) {
	// Silence the reconciler's JSON logs — simulation output is structured by the caller.
	prev := log.Logger
	log.Logger = log.Output(io.Discard)
	defer func() { log.Logger = prev }()

	crdEntry, ok := kat.CRDEntry(crdName)
	if !ok {
		return nil, fmt.Errorf("CRD %q not found in Katalog", crdName)
	}

	scheme, err := kat.Scheme()
	if err != nil {
		return nil, fmt.Errorf("building scheme: %w", err)
	}

	// Build the fake cluster
	fakeKube := NewFakeKubeclient(scheme)

	// Pre-seed the CR with managed labels and annotations so the reconciler's
	// idempotency guards skip those patches in every cycle. Without this, the
	// reconciler deep-copies from the indexer each cycle and always sees them
	// as missing, producing noise in every cycle's op list.
	// KatalogName is only set after ValidateConfig, which ParseFile doesn't call.
	// Use the Katalog metadata name directly — it is set by KomposeRuntimeKatalog.
	seedManagedMeta(cr, kat.Metadata().Name)

	// Build a fake informer backed by a static indexer containing the CR
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	if err := indexer.Add(cr); err != nil {
		return nil, fmt.Errorf("adding CR to indexer: %w", err)
	}
	informer := newFakeInformer(indexer)

	// Build the reconciler with the fake cluster.
	// event.NoopRecorder discards all events — simulation doesn't emit events.
	r := reconciler.NewGenericReconciler[domain.Object](
		crdEntry,
		informer,
		&event.NoopRecorder{},
		fakeKube,
		nil, // no Go hooks
		func() domain.Object { return &unstructured.Unstructured{} },
		nil, nil, nil, nil,
		kat, // Katalog for notification wiring
	)

	key, err := cache.MetaNamespaceKeyFunc(cr)
	if err != nil {
		return nil, fmt.Errorf("computing CR key: %w", err)
	}

	result := &Result{}
	var prevCycleOps []Op

	for cycle := 1; cycle <= maxCycles; cycle++ {
		fakeKube.AdvanceCycle()

		cycleResult := CycleResult{Cycle: cycle}
		cycleResult.Error = r.Reconcile(ctx, key)
		cycleResult.Ops = fakeKube.OpsForCycle(cycle)
		result.Cycles = append(result.Cycles, cycleResult)

		// Mark Deployments created this cycle as ready so the reconciler
		// can progress through state transitions on the next cycle.
		for _, op := range cycleResult.Ops {
			if op.Verb == "create" && op.Resource == "deployments" {
				fakeKube.MarkDeploymentReady(op.Namespace, op.Name)
			}
		}

		// Record the first cycle where ops stabilise. Do not break — run all
		// requested cycles so --cycles N is honoured exactly.
		if !result.Steady && cycle > 1 && opsMatch(cycleResult.Ops, prevCycleOps) {
			result.Steady = true
			result.SteadyAt = cycle
		}
		prevCycleOps = cycleResult.Ops
	}

	return result, nil
}

// seedManagedMeta pre-populates managed labels and annotations on the CR so the
// reconciler's idempotency guards skip those patches in every cycle.
func seedManagedMeta(cr *unstructured.Unstructured, katalogName string) {
	labels := cr.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[orklabels.Managed] = orklabels.ManagedValue
	cr.SetLabels(labels)

	ann := cr.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	ann[orklabels.AnnotationManagedBy] = katalogName
	ann[orklabels.AnnotationManagedSince] = time.Now().UTC().Format(time.RFC3339)
	cr.SetAnnotations(ann)
}

// opsMatch returns true when two op slices have the same verb+resource sequence.
func opsMatch(a, b []Op) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Verb != b[i].Verb || a[i].Resource != b[i].Resource {
			return false
		}
	}
	return true
}

// newFakeInformer wraps a static indexer as a SharedIndexInformer.
// The reconciler only calls GetIndexer() — all other methods are no-ops.
type fakeInformer struct {
	indexer cache.Indexer
	cache.SharedIndexInformer
}

func newFakeInformer(idx cache.Indexer) cache.SharedIndexInformer {
	return &fakeInformer{indexer: idx}
}

func (f *fakeInformer) GetIndexer() cache.Indexer { return f.indexer }
