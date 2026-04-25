# 01 — GenericClient

## What it is

`GenericClient` is the interface the informer factory uses to build ListerWatchers for typed CRDs — CRDs that have a registered Go struct (as opposed to dynamic `*unstructured.Unstructured` CRDs).

```go
type GenericClient interface {
    List(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error)
    Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
    ListInNamespace(ctx context.Context, namespace string, opts metav1.ListOptions) (runtime.Object, error)
    WatchInNamespace(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error)
}
```

`List`/`Watch` use the namespace baked into the `Client` at construction time. `ListInNamespace`/`WatchInNamespace` override that namespace — used by the Tier 1 namespace filter when `allowedNamespaces` has exactly one entry.

## Client struct

`*Client` in `genericclient.go` is the only concrete implementation:

```
Client
├── restClient   rest.Interface   — scoped REST client from SharedClientFactory
├── namespace    string           — namespace from CRDInfo (empty = all namespaces)
├── plural       string           — resource plural from CRDInfo
├── codec        runtime.ParameterCodec
└── objList      runtime.Object   — used as the list type template via reflect.New
```

`List` calls `restClient.Get().Namespace(c.namespace).Resource(c.plural).VersionedParams(...).Do(ctx).Into(list)`.

`ListInNamespace` is identical but replaces `c.namespace` with the caller-supplied `namespace` argument. This is the Tier 1 scoping mechanism for typed informers.

## ClientProvider

`ClientProvider` in `provider.go` is a deferred constructor registry — it maps `runtime.Object` type → a `ClientFactory` function that builds a `GenericClient` when first called.

Registration happens in `konstructOrkestra` before informers are created:

```go
provider.Register(object, func(k *kubeclient.Kubeclient) (informer.GenericClient, error) {
    return k.NewClient(list, kubeclient.CRDInfo{
        Kind:       crd.APITypes.Kind,
        Group:      crd.APITypes.Group,
        Version:    crd.APITypes.Version,
        APIPath:    crd.APITypes.APIPath,
        Plural:     crd.APITypes.Plural,
        Namespace:  crd.Namespace,
        Namespaced: crd.IsNamespaced(),
    })
})
```

`provider.For(obj)` is called inside the `newListWatch` closures — at the moment `List`/`Watch` is first invoked by the informer reflector. At that point the kubeclient is fully started and the REST config is valid.

## NewClient

```go
func (k *Kubeclient) NewClient(objList runtime.Object, info CRDInfo) (*Client, error) {
    restClient, err := k.SharedClientFactory(info.APIPath, info.Group, info.Version)
    return &Client{
        restClient: restClient,
        objList:    objList,
        namespace:  info.Namespace,
        plural:     info.Plural,
        codec:      k.RuntimeParameterCodec(),
    }, err
}
```

`SharedClientFactory` returns a `rest.Interface` scoped to the CRD's API group and version. The `objList` is stored (not used immediately) so `List` can produce a typed list object via `reflect.New`.

## When to add a method to GenericClient

Only extend `GenericClient` when the informer factory's ListerWatcher needs a new calling mode. Currently the only cases are:
- Cluster-scoped watch (`List`/`Watch`)
- Single-namespace watch (`ListInNamespace`/`WatchInNamespace`)

Do **not** add methods for one-off operations outside the ListerWatcher context — use `kube.Clientset()` or `kube.DynamicClient()` directly.

---

**Next →** [02 — DynamicListerWatcher](02-dynamic.md)
