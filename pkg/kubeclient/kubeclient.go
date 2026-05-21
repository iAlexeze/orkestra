// pkg/kubeclient/kubeclient.go
package kubeclient

import (
	"context"
	"fmt"
	"sync/atomic"

	"errors"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/konfig"
	"github.com/orkspace/orkestra/pkg/logger"
	"github.com/orkspace/orkestra/pkg/utils"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubeclient defines what a kube client is
type Kubeclient struct {
	name       string
	restConfig *rest.Config
	clientset  kubernetes.Interface
	dynamic    dynamic.Interface
	apiext     apiextclientset.Interface
	Info       *CRDInfo
	started    atomic.Bool
	mapper     meta.RESTMapper

	// Starter konfig
	konfig *konfig.Konfig
	scheme *runtime.Scheme

	// Testing
	FakeClientset kubernetes.Interface
}

func (k *Kubeclient) Mapper() meta.RESTMapper {
	return k.mapper
}

// RefreshMapper forces the deferred mapper to refresh its discovery cache.
// Call this after creating or updating CRDs so RESTMapping will pick them up.
func (k *Kubeclient) RefreshMapper() {
	if k.mapper == nil {
		return
	}
	if dm, ok := k.mapper.(*restmapper.DeferredDiscoveryRESTMapper); ok {
		dm.Reset()
	}
}

// Implementing yhe Komponent interface
var _ domain.Komponent = (*Kubeclient)(nil)

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------
// NewKubeclient returns a new Kubeclient with the correct scheme
func NewKubeclient(kfg *konfig.Konfig, scheme *runtime.Scheme) *Kubeclient {
	if scheme == nil {
		utils.Exit(errors.New("scheme cannot be nil"))
	}

	return &Kubeclient{
		name:   "kubeclient",
		scheme: scheme,
		konfig: kfg,
	}
}

// Start is called by orkestra.Start() to start kube client
func (k *Kubeclient) Start(ctx context.Context) error {
	cfg, err := k.buildConfig()
	if err != nil {
		return err
	}

	// Store config
	k.restConfig = cfg

	// Build core clientset
	logger.Debug().Msg("creating core clientset")
	k.clientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubeclient -- failed to create clientset: %w", err)
	}

	// Build dynamic client
	logger.Debug().Msg("creating dynamic client")
	k.dynamic, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubeclient -- failed to create dynamic client: %w", err)
	}

	// Build apiextensions clientset (for CRD patching)
	logger.Debug().Msg("creating apiextensions clientset")
	k.apiext, err = apiextclientset.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubeclient -- failed to create apiextensions clientset: %w", err)
	}

	// Build a cached discovery client and deferred RESTMapper
	logger.Debug().Msg("creating discovery client and RESTMapper")
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("kubeclient -- failed to create discovery client: %w", err)
	}
	// memory cache for discovery
	cached := memory.NewMemCacheClient(dc)
	// deferred mapper that lazily queries discovery and caches mappings
	k.mapper = restmapper.NewDeferredDiscoveryRESTMapper(cached)

	k.started.Store(true)
	return nil
}

// buildConfig returns a *rest config or nil in error
func (k *Kubeclient) buildConfig() (*rest.Config, error) {
	if k.restConfig != nil {
		return k.restConfig, nil
	}

	if k.scheme == nil {
		return nil, errors.New("scheme cannot be nil")
	}

	var restCfg *rest.Config
	var err error

	if k.konfig.Cluster().KubekonfigPath != "" {
		logger.Debug().Msg("using kubeconfig")
		restCfg, err = clientcmd.BuildConfigFromFlags(k.konfig.Cluster().MasterURL, k.konfig.Cluster().KubekonfigPath)
	} else {
		logger.Debug().Msg("using incluster configuration")
		restCfg, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	return restCfg, nil
}

// On-demand rest client
func (k *Kubeclient) RestClientFor(apiPath, group, version string) (*rest.RESTClient, error) {
	return k.SharedClientFactory(apiPath, group, version)
}

// On-demand dynamic client
func (k *Kubeclient) DynamicClientFor(apiPath, group, version string) (dynamic.Interface, error) {
	return k.dynamic, nil
}

// Notes
// Why merge patch and not strategic merge patch or Update? Three reasons:
// 1. First, you only touch metadata.finalizers — the rest of the object is untouched,
// which avoids resourceVersion conflicts if the object was updated between your cache read and this call.
// 2. Second, strategic merge patch requires the object type to be registered and understood by the API server's
// strategy engine — dynamic clients should use merge patch.
// 3. Third, Update sends the full object and requires a current resourceVersion — more fragile, more data over the wire.

// Started is called by orkestra for healthcheck
func (k *Kubeclient) Started() bool { return k.started.Load() }

// Shutdown is called by orkestra fir graceful shutdown
func (k *Kubeclient) Shutdown(ctx context.Context) {}

// Name returns the name of the kubeclient
func (k *Kubeclient) Name() string { return k.name }

// RestConfig returns the rest confif for the kube client
func (k *Kubeclient) RestConfig() *rest.Config { return k.restConfig }

// Clientset returns the kubernetes interface
func (k *Kubeclient) Clientset() kubernetes.Interface { return k.clientset }

// Dynamic returns yhe dynamic interface. Useful in 'dynamic' reconciler mode
func (k *Kubeclient) DynamicClient() dynamic.Interface { return k.dynamic }

// Scheme returns the runtime scheme for yhe kubeclient
func (k *Kubeclient) Scheme() *runtime.Scheme { return k.scheme }

// ApiextensionsClient returns the apiextensions clientset for CRD operations.
func (k *Kubeclient) ApiextensionsClient() apiextclientset.Interface { return k.apiext }

// New fake client
func NewFakeClientset() kubernetes.Interface {
	return fake.NewClientset()
}
