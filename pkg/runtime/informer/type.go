package informer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/runtime/queue"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ClientProvider interface {
	For(obj runtime.Object) (GenericClient, error)
}

type GenericClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	// ListInNamespace and WatchInNamespace scope the request to a single namespace.
	// Used by Tier 1 (namespace-scoped ListerWatcher) when IsSingleNamespace() is true.
	ListInNamespace(ctx context.Context, namespace string, opts metav1.ListOptions) (runtime.Object, error)
	WatchInNamespace(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error)
}

type Options struct {
	Name          string
	Resync        time.Duration
	Wq            *queue.Workqueue
	LabelSelector string
	FieldSelector string
	// Namespace scopes the ListerWatcher to a single namespace (Tier 1 filter).
	// "" means cluster-scoped watch (all namespaces). Set by RegisterNamespaceFilter
	// when IsSingleNamespace() is true.
	Namespace string
}

// InformerEntry holds an informer and its metadata — avoids storing
// a single shared opts on the factory which caused the name bug.
type InformerEntry struct {
	Informer cache.SharedIndexInformer
	Name     string
	Resync   time.Duration
	Missing  bool
	GVK      *schema.GroupVersionKind

	// Store the context and cancel function
	Ctx             context.Context    // stored context
	Cancel          context.CancelFunc // stored cancel function
	WasNeverStarted bool
}

// All mappings key: gvk
type Factory struct {
	clientProvider ClientProvider
	restConfig     *rest.Config
	defaultWq      *queue.Workqueue
	queueRegistry  *queue.QueueRegistry
	namespace      string
	scheme         *runtime.Scheme
	defaultResync  time.Duration             // factory-level default
	informers      map[string]*InformerEntry // per-type entry

	started atomic.Bool
	mu      sync.RWMutex
	ready   chan struct{}

	// Post start retry for missing CRDs
	missing map[string]*InformerEntry

	// namespaceFilters maps GVK string to its namespace restriction.
	// Populated by RegisterNamespaceFilter during CRD registration.
	// Checked in handleEvent before enqueue — read lock only on the hot path.
	namespaceFilters map[string]*NamespaceFilter

	// enqueueFilters maps GVK string to a pre-enqueue condition gate function.
	// Populated by RegisterEnqueueFilter during CRD registration.
	// enqueueAllowed unwraps tombstones and asserts to domain.Object before
	// calling the function — works for both dynamic and typed CRDs.
	enqueueFilters map[string]func(domain.Object) bool

	// updateFilters maps GVK string to a sentinel-aware update config.
	// The factory computes sentinels from declared names at event time and
	// calls gate to decide whether to enqueue. Splitting the two means
	// runtime_konstructor.go only passes configuration; computation stays here.
	updateFilters map[string]*updateFilterCfg
}

// updateFilterCfg holds the configuration registered for a sentinel-aware GVK.
// declared is captured at startup; gate is evaluated at event time with the
// already-computed sentinel values so the closure never needs to call sentinel.Compute.
type updateFilterCfg struct {
	declared []string
	gate     func(newObj domain.Object, sentinels map[string]string) bool
}

func SharedInformerFactory(
	cp ClientProvider,
	restConfig *rest.Config,
	queueRegistry *queue.QueueRegistry,
	defaultWq *queue.Workqueue,
	scheme *runtime.Scheme,
	kfg *konfig.Konfig,
) *Factory {
	return &Factory{
		clientProvider:   cp,
		restConfig:       restConfig,
		queueRegistry:    queueRegistry,
		defaultWq:        defaultWq,
		namespace:        kfg.Cluster().Namespace(),
		scheme:           scheme,
		defaultResync:    kfg.Katalog().DefaultResync(),
		informers:        make(map[string]*InformerEntry),
		missing:          make(map[string]*InformerEntry),
		ready:            make(chan struct{}),
		namespaceFilters: make(map[string]*NamespaceFilter),
		enqueueFilters:   make(map[string]func(domain.Object) bool),
		updateFilters:    make(map[string]*updateFilterCfg),
	}
}
