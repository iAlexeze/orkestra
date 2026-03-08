// pkg/kubeclient/provider.go
package kubeclient

import (
	"fmt"
	"reflect"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/informer"
	"k8s.io/apimachinery/pkg/runtime"
)

type ClientProvider struct {
	kube    *Kubeclient
	clients map[reflect.Type]ClientFactory
}

func (k *Kubeclient) ClientProvider() *ClientProvider {
	return &ClientProvider{
		kube:    k,
		clients: make(map[reflect.Type]ClientFactory),
	}
}

func (p *ClientProvider) Register(obj runtime.Object, factory ClientFactory) {
	p.clients[reflect.TypeOf(obj)] = factory
}

func (p *ClientProvider) For(obj runtime.Object) (informer.GenericClient, error) {
	factory, ok := p.clients[reflect.TypeOf(obj)]
	if !ok {
		return nil, fmt.Errorf("no client registered for %T", obj)
	}
	return factory(p.kube)
}

type ClientFactory func(*Kubeclient) (informer.GenericClient, error)
