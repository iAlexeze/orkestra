package informer

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/queue"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type ClientProvider interface {
	// Returns a client that knows how to List/Watch a specific type
	For(obj runtime.Object) (GenericClient, error)
}

type GenericClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type Factory struct {
	clientProvider ClientProvider
	queue          *queue.Workqueue
	namespace      string
	scheme         *runtime.Scheme
	resync         time.Duration
	informers      map[reflect.Type]cache.SharedIndexInformer
	started        bool
	mu             sync.RWMutex  // Mmutex for thread safety
	ready          chan struct{} // Signal when factory is ready
}

func SharedInformerFactory(cp ClientProvider, wq *queue.Workqueue, scheme *runtime.Scheme, namespace string, resync time.Duration) *Factory {
	return &Factory{
		clientProvider: cp,
		queue:          wq,
		namespace:      namespace,
		scheme:         scheme,
		resync:         resync,
		informers:      make(map[reflect.Type]cache.SharedIndexInformer),
		ready:          make(chan struct{}),
	}
}
