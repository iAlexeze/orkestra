package kubeclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ialexeze/orkestra/domain"
	crderror "github.com/ialexeze/orkestra/pkg/error"
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

type Kubeclient struct {
	name       string
	restConfig *rest.Config
	clientset  kubernetes.Interface
	dynamic    dynamic.Interface
	scheme     *runtime.Scheme
	Config     Config
	Info       *CRDInfo
	started    bool
}

type Config struct {
	Kubeconfig string
	Masterurl  string
	Scheme     *runtime.Scheme // REQUIRED
}

var _ domain.Komponent = (*Kubeclient)(nil)

func NewKubeclient(cfg Config) *Kubeclient {
	if cfg.Scheme == nil {
		utils.Exit(crderror.ErrSchemeNill)
	}

	return &Kubeclient{
		name:   "kubeclient",
		scheme: cfg.Scheme,
		Config: cfg,
	}
}

func (k *Kubeclient) Start(ctx context.Context) error {
	cfg, err := k.buildConfig()
	if err != nil {
		return err
	}

	// Store config
	k.restConfig = cfg

	// Build core clientset
	logger.Info().Msg("creating core clientset")
	k.clientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Build dynamic client
	logger.Info().Msg("creating dynamic client")
	k.dynamic, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	k.started = true
	return nil
}

func (k *Kubeclient) buildConfig() (*rest.Config, error) {
	if k.restConfig != nil {
		return k.restConfig, nil
	}

	if k.scheme == nil {
		return nil, fmt.Errorf("scheme is nil in kubeclient")
	}

	var restCfg *rest.Config
	var err error

	if k.Config.Kubeconfig != "" {
		logger.Info().Msg("using kubeconfig")
		restCfg, err = clientcmd.BuildConfigFromFlags(k.Config.Masterurl, k.Config.Kubeconfig)
	} else {
		restCfg, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	// Ensure the config uses our global scheme
	// restCfg.NegotiatedSerializer = serializer.NewCodecFactory(k.scheme)

	return restCfg, nil
}

// On-demand rest client
func (k *Kubeclient) RestClientFor(apiPath, group, version string) (*rest.RESTClient, error) {
	return k.SharedClientFactory(apiPath, group, version)
}

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

func (k *Kubeclient) Started() bool { return k.started }

func (k *Kubeclient) Shutdown(ctx context.Context) {}

func (k *Kubeclient) Name() string { return k.name }

func (k *Kubeclient) RestConfig() *rest.Config { return k.restConfig }

func (k *Kubeclient) Clientset() kubernetes.Interface { return k.clientset }

func (k *Kubeclient) Dynamic() dynamic.Interface { return k.dynamic }

func (k *Kubeclient) Scheme() *runtime.Scheme { return k.scheme }
