package informer

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ialexeze/orkestra/pkg/queue"
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
}

type Options struct {
	Name   string
	Resync time.Duration
	Wq     *queue.Workqueue
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
	Ctx    context.Context    // stored context
	Cancel context.CancelFunc // stored cancel function
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
}

func SharedInformerFactory(
	cp ClientProvider,
	restConfig *rest.Config,
	queueRegistry *queue.QueueRegistry,
	defaultWq *queue.Workqueue,
	scheme *runtime.Scheme,
	namespace string,
	defaultResync time.Duration,
) *Factory {
	return &Factory{
		clientProvider: cp,
		restConfig:     restConfig,
		queueRegistry:  queueRegistry,
		defaultWq:      defaultWq,
		namespace:      namespace,
		scheme:         scheme,
		defaultResync:  defaultResync,
		informers:      make(map[string]*InformerEntry),
		missing:        make(map[string]*InformerEntry),
		ready:          make(chan struct{}),
	}
}
