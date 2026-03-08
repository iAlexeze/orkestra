// pkg/kubeclient/genericclient.go
package kubeclient

import (
	"context"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type Client struct {
	restClient rest.Interface
	namespace  string
	plural     string
	codec      runtime.ParameterCodec
	listType   runtime.Object
}

func (k *Kubeclient) NewClient(listType runtime.Object, info CRDInfo) (*Client, error) {
	restClient, err := k.SharedClientFactory(info)
	if err != nil {
		return nil, err
	}

	if info.ClusterScoped {
		info.Namespace = ""
	}

	return &Client{
		restClient: restClient,
		listType:   listType,
		namespace:  info.Namespace,
		plural:     info.NamePlural,
		codec:      k.RuntimeParameterCodec(),
	}, nil
}

// List returns runtime.Object - exactly what GenericClient needs!
func (c *Client) List(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
	// Create a new instance of the list type
	list := reflect.New(reflect.TypeOf(c.listType).Elem()).Interface().(runtime.Object)

	err := c.restClient.Get().
		Namespace(c.namespace).
		Resource(c.plural).
		VersionedParams(&opts, c.codec).
		Do(ctx).
		Into(list)

	return list, err
}

func (c *Client) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.Get().
		Namespace(c.namespace).
		Resource(c.plural).
		VersionedParams(&opts, c.codec).
		Watch(ctx)
}
