# `pkg/registry` – Internal Resource Registry

The `pkg/registry` package is the **standard library** of Kubernetes resource operations used by Orkestra’s `GenericReconciler`. It provides idempotent, owner‑reference‑aware Create, Update, Delete, and Resolve functions for common resource types (Deployment, Service, Secret, ConfigMap, etc.).

This package is not meant for direct end‑user consumption; it is the runtime engine that powers declarative templates.

---

## Structure

```
pkg/registry/
├── deployments/
│   ├── deployment.go
│   └── types.go
├── ...
└── template/
    ├── resolver.go
    └── resolver_test.go
```

Each resource subdirectory follows a consistent pattern:

- `xxx.go` – exports four functions:

```go
  - Create(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedXxxSpec) error

  - Update(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedXxxSpec) error
  
  - Delete(ctx context.Context, kube kubeclient.KubeClient, owner domain.Object, spec ResolvedXxxSpec) error
  
  - Resolve(src orktypes.XxxTemplateSource, ownerName string) ResolvedXxxSpec
```

- `test/xxx_test.go` – unit tests for those functions.

The `template/` subdirectory contains the `Resolver` that evaluates Go templates against live CR objects.

---

## Adding a New Resource Type

To add a new resource type to the registry (e.g., `HorizontalPodAutoscaler`):

1. **Create a subdirectory** `pkg/registry/hpas/`.
2. **Define the template source struct** in `pkg/types/hooks.go` (add `HPAs []HPATemplateSource` to `HookTemplates`).
3. **Implement the four functions** in `hpas/hpa.go`.
4. **Add a resolver method** in `template/resolver.go` (e.g., `ResolveHPATemplate`).
5. **Add the runner** in `pkg/reconciler/run_hpas.go`.
6. **Update the generic reconciler** to call the runner.
7. **Write tests** for the new functions.

See the existing `deployments` package for a complete example.

---

## The Contract

All registry packages adhere to the same contract:

- **Create** – idempotent. If the resource already exists, it does nothing and returns `nil`. It sets owner references and system labels (`managed-by: orkestra`, `orkestra-owner: <cr-name>`).
- **Update** – drift correction. If the resource does not exist, it creates it. If it exists but has drifted (e.g., image changed), it updates it.
- **Delete** – idempotent. If the resource does not exist, it returns `nil`.
- **Resolve** – takes a template source (with unresolved field values) and returns a resolved spec after applying defaults and system labels.

All functions receive a `kubeclient.Kubeclient` and a `domain.Object` (the owner CR). They never make assumptions about the owner’s type.

---

## Contributing

We welcome contributions to the internal registry! Whether you’re adding a new resource type, improving an existing implementation, or fixing bugs, please follow the guidelines in **[CONTRIBUTING.md](./CONTRIBUTING.md)**.

---

**See also:** [Orkestra Architecture](../../docs/runtime-manual/architecture/index.md) for how the registry fits into the runtime.
