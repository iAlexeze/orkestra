# 🧩 **1. Discovery Client**
```go
disco, err := discovery.NewDiscoveryClientForConfig(cfg)
```

### What it is  
The **DiscoveryClient** talks to the Kubernetes API server’s **/api** and **/apis** endpoints.

### What it retrieves  
- All API groups  
- All versions  
- All resources (kinds)  
- Whether they are namespaced  
- Whether they support list/watch  

### Why you need it  
Because the API server only knows about CRDs **after they are installed**, and discovery is the canonical way to check what exists.

---

# 🧩 **2. Cached Discovery (memory.NewMemCacheClient)**
```go
mapper := restmapper.NewDeferredDiscoveryRESTMapper(
    memory.NewMemCacheClient(disco),
)
```

### What this does  
- Wraps the discovery client in a **local in-memory cache**  
- Avoids hammering the API server repeatedly  
- Automatically refreshes when needed  

### Why it matters  
Without this, every RESTMapping call would hit the API server directly.  
With it, Kubernetes behaves more like a real controller-runtime mapper.

---

# 🧩 **3. GroupKind**
```go
gk := schema.GroupKind{Group: group, Kind: kind}
```

### What this represents  
A **GroupKind** is the identity of a Kubernetes type *without* a version.

Example:

- Group: `platform.ialexeze.io`
- Kind: `Project`

This is the “type name” of your CRD.

### Why version is separate  
Because a CRD can have multiple versions:

- v1alpha1  
- v1beta1  
- v1  

RESTMapping resolves **GroupKind + Version** → actual API endpoint.

---

# 🧩 **4. RESTMapping**
```go
_, err = mapper.RESTMapping(gk, version)
```

### What RESTMapping does  
It asks Kubernetes:

> “Given this GroupKind and Version, what is the REST endpoint for it?”

If the CRD exists, you get a mapping like:

```
/apis/platform.ialexeze.io/v1alpha1/projects
```

If the CRD does **not** exist, you get:

- `NoMatchError`  
- or a wrapped error saying “no matches for kind …”

### Why this is the perfect CRD-existence check  
Because the API server only exposes CRDs **after** they are installed and accepted.

---

# 🧩 **5. Detecting missing CRD**
```go
if meta.IsNoMatchError(err) {
    return fmt.Errorf("CRD %s.%s/%s not installed", kind, group, version)
}
```

### What this means  
If the CRD is not installed, Kubernetes literally says:

> “I don’t know this GroupKind/Version.”

This is the cleanest, most idiomatic way to detect missing CRDs.

---

# 🧩 **6. Returning the error**
```go
return err
```

### Why this matters  
You want your retry logic to handle the waiting, not this function.

This function:

- returns nil → CRD exists  
- returns NoMatchError → CRD missing  
- returns other errors → something else is wrong  

This separation of concerns is exactly how Kubernetes controllers are structured.

---

# 🎉 **Putting it all together**

`WaitForCRD()` function:

- Builds a discovery client  
- Wraps it in a cached RESTMapper  
- Asks Kubernetes if the CRD exists  
- Returns a clean error if it doesn’t  
- Lets your retry logic handle the waiting  

This is exactly how some upstream controllers (cert-manager, external-dns, kube-state-metrics) detect CRD readiness.