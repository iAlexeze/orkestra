# 02 — Bundle, RBAC, and ConfigMap

These three generators produce the Kubernetes resources that deploy Orkestra into a cluster. They share internal rendering functions and always output documents in a fixed order so that `kubectl apply -f` works without `--server-side`.

## Document order guarantee

Every generator that writes namespaced resources ensures the Namespace appears first:

```
Namespace
└── ServiceAccount (orkestra)
└── ServiceAccount (orkestra-gateway)        [when IncludeGateway]
└── ServiceAccount (orkestra-cc)             [when IncludeControlCenter]
└── ClusterRole (orkestra)                   [when IncludeRuntime]
└── ClusterRole (orkestra-gateway)           [when IncludeGateway]
└── ClusterRoleBinding (orkestra)            [when IncludeRuntime]
└── ClusterRoleBinding (orkestra-gateway)    [when IncludeGateway]
```

`bundle` appends the ConfigMap after the RBAC block:

```
... RBAC documents above ...
└── ConfigMap (orkestra-config)
```

A workload Namespace is also prepended when `--workload-namespace` differs from the system namespace.

## RBAC generation

```sh
ork generate rbac -f katalog.yaml [-n namespace] [--for runtime|gateway|cc]
```

`RBACWithOptions` calls `katalog.GenerateRuntimeRBACRules()` and `katalog.GenerateGatewayRBACRules()` upstream (in `cmd/cli/generate.go`) and passes the resulting `[]rbacv1.PolicyRule` slices here. The generator does not inspect CRD entries directly — it only assembles the RBAC objects from pre-computed rules.

Each ClusterRole uses `labels.OrkestraBaseLabels()` so resources are identifiable and manageable by `ork doctor`.

## ConfigMap generation

```sh
ork generate configmap -f katalog.yaml [-n namespace]
```

The ConfigMap embeds the fully-expanded Katalog YAML (after all merges and inheritance) under the key `katalog.yaml`. The Orkestra runtime reads this key at startup — it is the primary mechanism for injecting a Katalog into a running cluster without baking it into the operator image.

`ConfigMap(expandedYAML []byte, namespace string)` is a thin wrapper: it marshals a `corev1.ConfigMap` with the YAML as a data value and prepends the namespace document.

## Bundle generation

```sh
ork generate bundle -f katalog.yaml [-n namespace] [-w workload-namespace] [--for ...]
```

`RenderBundle` calls `renderNamespaceAndRBAC` and `renderConfigMapBytes` internally and joins the outputs with YAML document separators. It is the single output path that produces a complete, apply-ready bundle in one file.

The `--for` flag is particularly useful here:

```sh
# Deploy only the gateway component (e.g. in a split runtime/gateway topology)
ork generate bundle -f katalog.yaml --for gateway

# Deploy runtime and control center but not the gateway
ork generate bundle -f katalog.yaml --for runtime,cc
```

→ Next: [03-type-registry.md](03-type-registry.md)
