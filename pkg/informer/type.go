package informer

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/ialexeze/orkestra/pkg/queue"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
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

// informerEntry holds an informer and its metadata — avoids storing
// a single shared opts on the factory which caused the name bug.
type informerEntry struct {
	informer cache.SharedIndexInformer
	name     string
	resync   time.Duration
}

type Factory struct {
	clientProvider ClientProvider
	defaultWq      *queue.Workqueue
	queueRegistry  *queue.QueueRegistry
	namespace      string
	scheme         *runtime.Scheme
	defaultResync  time.Duration                   // factory-level default
	informers      map[reflect.Type]*informerEntry // per-type entry — not a shared opts
	started        bool
	mu             sync.RWMutex
	ready          chan struct{}
}

func SharedInformerFactory(
	cp ClientProvider,
	queueRegistry *queue.QueueRegistry,
	defaultWq *queue.Workqueue,
	scheme *runtime.Scheme,
	namespace string,
	defaultResync time.Duration,
) *Factory {
	return &Factory{
		clientProvider: cp,
		queueRegistry:  queueRegistry,
		defaultWq:      defaultWq,
		namespace:      namespace,
		scheme:         scheme,
		defaultResync:  defaultResync,
		informers:      make(map[reflect.Type]*informerEntry),
		ready:          make(chan struct{}),
	}
}
