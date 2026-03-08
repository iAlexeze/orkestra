package kubeclient

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

type CRDInfo struct {
	Kind             string                  // Required by Registry
	Group            string                  // Required if GroupVersion is not specified
	Version          string                  // Required if GroupVersion is not specified
	GroupVersion     *schema.GroupVersion    // Optional (can be used if Group and Version are not specified)
	GroupVersionKind schema.GroupVersionKind //	Useful for some manipulations
	NamePlural       string
	ClusterScoped    bool // Required for cluster-scoped resources
	APIPath          string
	Namespace        string
}

// SharedClientFactory provides a simple way to build clients from config
func (k *Kubeclient) SharedClientFactory(info CRDInfo) (*rest.RESTClient, error) {
	switch {
	case info.APIPath == "":
		info.APIPath = "/apis"
	case info.GroupVersion == nil:
		if info.Group == "" && info.Version == "" {
			return nil, fmt.Errorf("required variables: Group, Version")
		}
		info.GroupVersion = &schema.GroupVersion{
			Group:   info.Group,
			Version: info.Version,
		}
	}

	// Build restclient
	cfg := rest.CopyConfig(k.RestConfig())
	cfg.GroupVersion = info.GroupVersion

	cfg.APIPath = info.APIPath
	cfg.NegotiatedSerializer = serializer.NewCodecFactory(k.Scheme())
	cfg.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.RESTClientFor(cfg)

}

func (k *Kubeclient) RuntimeParameterCodec() runtime.ParameterCodec {
	return runtime.NewParameterCodec(k.Scheme())
}
