package simulate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/event"
	"github.com/orkspace/orkestra/pkg/katalog"
	orklabels "github.com/orkspace/orkestra/pkg/labels"
	"github.com/orkspace/orkestra/pkg/reconciler"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// noopTransport returns an empty 200 response for every request.
// Used to stub out external: HTTP calls during simulation so they
// execute the full call path but produce empty result fields
// rather than real network errors.
type noopTransport struct{}

func (noopTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

// Result is the output of one simulation run.
type Result struct {
	Cycles   []CycleResult
	Steady   bool
	SteadyAt int      // cycle number where steady state was first detected (0 if not reached)
	Notes    []string // informational notes about blocks that could not execute (e.g. constructor body)
}

// CycleResult is the output of one reconcile cycle.
type CycleResult struct {
	Cycle int
	Ops   []Op
	Error error
}

// RunOptions controls optional behaviour for a simulation run.
type RunOptions struct {
	// SkipExternal stubs all external: HTTP calls with an empty 200 response.
	// When false (the default), external calls are attempted against the real network.
	SkipExternal bool
}

// Run simulates the operator against an in-memory cluster.
//
// kat is the parsed Katalog.
// crdName is the CRD entry to simulate.
// cr is the CR to reconcile.
// maxCycles is the maximum number of reconcile cycles.
func Run(ctx context.Context, kat *katalog.Katalog, crdName string, cr *unstructured.Unstructured, maxCycles int, opts RunOptions) (*Result, error) {
	// Silence the reconciler's JSON logs — simulation output is structured by the caller.
	prev := log.Logger
	log.Logger = log.Output(io.Discard)
	defer func() { log.Logger = prev }()

	// Stub out external: HTTP calls only when --skip-external is requested.
	// Without it, external calls hit the real network — useful when targeting a
	// local mock server or a staging API from a development machine.
	if opts.SkipExternal {
		prevTransport := reconciler.ExternalHTTPTransport
		reconciler.ExternalHTTPTransport = noopTransport{}
		defer func() { reconciler.ExternalHTTPTransport = prevTransport }()
	}

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

	gvk := schema.GroupVersionKind{
		Group:   crdEntry.APITypes.Group,
		Version: crdEntry.APITypes.Version,
		Kind:    crdEntry.APITypes.Kind,
	}

	// Look up hooks from the registry. In the standard ork binary the map is
	// empty for custom types and hookBinder stays nil.
	// In a custom operator binary produced by `make registry && make build`,
	// the init() in zz_generated_typeregistry.go populates HookRegistry so
	// the actual hook function runs inside the fake cluster.
	var hookBinder domain.AnyReconcileHooks
	if fn, ok := orktypes.HookRegistry[gvk]; ok {
		hookBinder = fn()
	}

	result := &Result{}

	// Build the reconciler. If a constructor is registered (custom binary with
	// ork generate registry output), use it directly — it receives the fake
	// kubeclient and runs its full reconcile loop against the in-memory cluster.
	// nil is passed for *event.Event; constructors should guard against nil ev.
	// Otherwise fall back to GenericReconciler with any registered hook binder.
	var r domain.Reconciler
	if factoryFn, ok := orktypes.ReconcilerRegistry[gvk]; ok {
		r = factoryFn(fakeKube, informer, event.Discard())
	} else {
		r = reconciler.NewGenericReconciler[domain.Object](
			crdEntry,
			informer,
			nil,
			fakeKube,
			hookBinder,
			func() domain.Object { return &unstructured.Unstructured{} },
			nil, nil, nil, nil,
			kat,
		)
	}

	key, err := cache.MetaNamespaceKeyFunc(cr)
	if err != nil {
		return nil, fmt.Errorf("computing CR key: %w", err)
	}
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
	labels[orklabels.ManagedKey] = orklabels.ManagedValue
	labels[orklabels.DeletionProtectionLabel] = orklabels.DeletionProtectionValue
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
