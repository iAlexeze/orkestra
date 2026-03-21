// pkg/kubeclient/kubeclient.go
package kubeclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/ialexeze/orkestra/domain"
	orkerror "github.com/ialexeze/orkestra/pkg/error"
	"github.com/ialexeze/orkestra/pkg/logger"
	"github.com/ialexeze/orkestra/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubeclient defines what a kube client is
type Kubeclient struct {
	name       string
	restConfig *rest.Config
	clientset  kubernetes.Interface
	dynamic    dynamic.Interface
	scheme     *runtime.Scheme
	Config     Config
	Info       *CRDInfo
	started    atomic.Bool
}

// Config defines kube configuration
type Config struct {
	Kubeconfig string
	Masterurl  string
	Scheme     *runtime.Scheme // REQUIRED
}

// Implementing yhe Komponent interface
var _ domain.Komponent = (*Kubeclient)(nil)

// NewKubeclient returns a new Kubeclient with the correct scheme
func NewKubeclient(cfg Config) *Kubeclient {
	if cfg.Scheme == nil {
		utils.Exit(orkerror.ErrSchemeNill)
	}

	return &Kubeclient{
		name:   "kubeclient",
		scheme: cfg.Scheme,
		Config: cfg,
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

	k.started.Store(true)
	return nil
}

// buildConfig returns a *rest config or nil in error
func (k *Kubeclient) buildConfig() (*rest.Config, error) {
	if k.restConfig != nil {
		return k.restConfig, nil
	}

	if k.scheme == nil {
		return nil, orkerror.ErrSchemeNill
	}

	var restCfg *rest.Config
	var err error

	if k.Config.Kubeconfig != "" {
		logger.Debug().Msg("using kubeconfig")
		restCfg, err = clientcmd.BuildConfigFromFlags(k.Config.Masterurl, k.Config.Kubeconfig)
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

// Patch Finalizers
func (k *Kubeclient) PatchFinalizers(
	ctx context.Context,
	obj runtime.Object,
	gvr schema.GroupVersionResource,
	finalizers []string,
) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("getting accessor: %w", err)
	}

	// Build a minimal merge patch — only touch finalizers
	// Never send the full object — avoids resourceVersion conflicts
	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers": finalizers,
		},
	}

	body, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshalling finalizer patch: %w", err)
	}

	namespace := accessor.GetNamespace()
	name := accessor.GetName()

	if namespace == "" {
		// Cluster-scoped resource
		_, err = k.dynamic.Resource(gvr).Patch(
			ctx,
			name,
			types.MergePatchType,
			body,
			metav1.PatchOptions{},
		)
	} else {
		// Namespace-scoped resource
		_, err = k.dynamic.Resource(gvr).Namespace(namespace).Patch(
			ctx,
			name,
			types.MergePatchType,
			body,
			metav1.PatchOptions{},
		)
	}

	return err
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
func (k *Kubeclient) Dynamic() dynamic.Interface { return k.dynamic }

// Scheme returns the runtime scheme for yhe kubeclient
func (k *Kubeclient) Scheme() *runtime.Scheme { return k.scheme }
