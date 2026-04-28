---
title: "Katalog"
weight: 161
---

# Katalog

`pkg/katalog` is the package responsible for loading a merged CRD entry list into a running Orkestra instance. It takes the output of the Merger, enriches it with cluster-authoritative API metadata, populates the conversion and admission registries, and produces the final configuration that the runtime uses to start per-CRD operator stacks.

---

## Responsibilities

- Parsing Katalog and Komposer YAML documents
- Enriching CRD entries with group, version, plural, and scope from the cluster's discovery API
- Enriching built-in Kubernetes kinds from the internal built-in registry
- Validating the dependency graph for cycles
- Populating the `InMemoryConversionRegistry`
- Populating the `InMemoryAdmissionRegistry`
- Wiring hook factories and constructors from the runtime registries
- Setting typed vs dynamic mode per CRD

---

## KomposeRuntimeKatalog

```go
func KomposeRuntimeKatalog(ctx context.Context, paths []string, kube *kubeclient.Kubeclient) (*Katalog, error)
```

This is the main entry point. It performs the following sequence:

```
paths []string
  │
  ▼
Merger.Merge(paths...)
  │  resolves all sources → []CRDEntry (raw, unenriched)
  │
  ▼
EnrichCRDEntry(entry, discoveryClient)
  │  for each CRD entry:
  │    built-in kind? → enrich from internal registry
  │    custom kind?   → enrich from cluster discovery API
  │    sets: Group, Version, Plural, APIPath, Namespaced
  │    sets: GroupVersion, GroupVersionKind, GroupVersionResource
  │
  ▼
setMode(entry)
  │  apiTypes.location set → typed mode
  │  apiTypes.location empty → dynamic mode
  │
  ▼
addRuntimeObjects(entry)
  │  dynamic mode: sets DynamicModeObject and ListDynamicModeObject
  │                to factory functions returning *unstructured.Unstructured
  │  typed mode:   looks up ObjectRegistry[gvk] and ListRegistry[gvk]
  │                set by ork generate registry
  │
  ▼
addHooks(entry)
  │  looks up HookRegistry[gvk]
  │  sets ReconcilerConfig.HookFactory
  │
  ▼
addReconcilers(entry)
  │  looks up ReconcilerRegistry[gvk]
  │  sets ReconcilerConfig.Constructor
  │
  ▼
validateDependencyGraph(entries)
  │  topological sort — fatal error on cycle
  │
  ▼
conversionRegistry.RegisterConversionRulesFromEntry(entry)
  │  populates InMemoryConversionRegistry
  │  only for entries with conversion.paths declared
  │
  ▼
admissionRegistry.registerAdmissionRulesFromEntry(entry)
  │  populates InMemoryAdmissionRegistry
  │  only for entries with validation or mutation rules
  │
  ▼
*Katalog ready — passed to Orkestra runtime
```

---

## Built-in kind enrichment

When a CRD entry declares only `apiTypes.kind` (e.g. `kind: Deployment`), Orkestra checks its internal built-in registry before making any API calls:

```go
// pkg/katalog/builtins.go
var BuiltInKinds = map[string]BuiltInKindInfo{
    "Deployment":             {Group: "apps",                  Version: "v1", Plural: "deployments",              Namespaced: true},
    "StatefulSet":            {Group: "apps",                  Version: "v1", Plural: "statefulsets",             Namespaced: true},
    "DaemonSet":              {Group: "apps",                  Version: "v1", Plural: "daemonsets",               Namespaced: true},
    "ReplicaSet":             {Group: "apps",                  Version: "v1", Plural: "replicasets",              Namespaced: true},
    "Pod":                    {Group: "",                       Version: "v1", Plural: "pods",                     Namespaced: true},
    "Service":                {Group: "",                       Version: "v1", Plural: "services",                 Namespaced: true},
    "ConfigMap":              {Group: "",                       Version: "v1", Plural: "configmaps",               Namespaced: true},
    "Secret":                 {Group: "",                       Version: "v1", Plural: "secrets",                  Namespaced: true},
    "ServiceAccount":         {Group: "",                       Version: "v1", Plural: "serviceaccounts",          Namespaced: true},
    "PersistentVolumeClaim":  {Group: "",                       Version: "v1", Plural: "persistentvolumeclaims",   Namespaced: true},
    "Namespace":              {Group: "",                       Version: "v1", Plural: "namespaces",               Namespaced: false},
    "Node":                   {Group: "",                       Version: "v1", Plural: "nodes",                    Namespaced: false},
    "Job":                    {Group: "batch",                  Version: "v1", Plural: "jobs",                     Namespaced: true},
    "CronJob":                {Group: "batch",                  Version: "v1", Plural: "cronjobs",                 Namespaced: true},
    "Ingress":                {Group: "networking.k8s.io",      Version: "v1", Plural: "ingresses",                Namespaced: true},
    "NetworkPolicy":          {Group: "networking.k8s.io",      Version: "v1", Plural: "networkpolicies",          Namespaced: true},
    "HorizontalPodAutoscaler":{Group: "autoscaling",           Version: "v2", Plural: "horizontalpodautoscalers", Namespaced: true},
    "CustomResourceDefinition":{Group:"apiextensions.k8s.io",  Version: "v1", Plural: "customresourcedefinitions",Namespaced: false},
    "ClusterRole":            {Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "clusterroles",         Namespaced: false},
    "Role":                   {Group: "rbac.authorization.k8s.io", Version: "v1", Plural: "roles",               Namespaced: true},
    // ... and more
}
```

If the kind is found in the built-in registry, the fields are set without an API call. `IsBuiltIn` is set to `true` on the entry for informational purposes.

If the kind is not in the built-in registry, Orkestra queries the cluster's discovery API:

```go
lists, err := discoveryClient.ServerPreferredResources()
// searches for the matching resource group and version
```

{{< callout type="note" title="Why a built-in registry?" >}}
The discovery API call is reliable but adds startup latency proportional
to the number of CRD entries that need enrichment. The built-in registry
eliminates API calls for the 30+ most common Kubernetes resource types.
For custom CRDs, the discovery call is unavoidable.
{{< /callout >}}

{{< callout type="warning" title="Enrichment fails for unknown custom kinds" >}}
If a CRD entry declares `kind: MyCustomThing` but does not declare
`apiTypes.group`, `apiTypes.version`, and `apiTypes.plural`, and the
kind is not installed in the cluster, enrichment will fail:
{{< /callout >}}

    ```
    error: enrichment failed for "my-crd": kind "MyCustomThing" not found
    in cluster discovery — ensure the CRD is installed and apiTypes is complete
    ```

    Always run `ork validate --katalog katalog.yaml` to surface enrichment
    failures before deploying.

---

## Mode detection

```go
func setMode(entry *orktypes.CRDEntry) {
    if entry.Mode != "" {
        return // explicit — respect the declared mode
    }
    if entry.APITypes.Location != "" {
        entry.Mode = orktypes.CRDModeTyped
    } else {
        entry.Mode = orktypes.CRDModeDynamic
    }
}
```

Mode detection is a single field check. The default is always dynamic. Setting `apiTypes.location` opts into typed mode. Setting `mode: dynamic` explicitly overrides typed mode even when `location` is set — useful when you want the import for hooks but do not want the informer to use typed decoding.

---

## Runtime object wiring

In dynamic mode, `DynamicModeObject` and `ListDynamicModeObject` are set to factory functions that return `*unstructured.Unstructured` and `*unstructured.UnstructuredList`. These are always safe — no generated types needed.

In typed mode, `DynamicModeObject` is looked up from `ObjectRegistry`:

```go
factory, ok := orktypes.ObjectRegistry[gvk]
if !ok {
    return fmt.Errorf("no object factory for %s — run ork generate registry", gvk)
}
entry.DynamicModeObject = factory
```

The `ObjectRegistry` is populated by `zz_generated_runtime_registry.go`, which `ork generate registry` produces. If this file is missing or stale, typed mode CRDs fail to start with the above error.

---

## Dependency graph validation

```go
func validateDependencyGraph(entries []orktypes.CRDEntry) ([]orktypes.CRDEntry, error)
```

Takes the full entry list and performs a topological sort using Kahn's algorithm. Returns entries in start order (dependencies before dependents). Returns an error describing the cycle if one exists:

```
error: circular dependency detected: application → namespace → project → application
```

The sorted order is stored on the `Katalog` struct and used by the runtime startup sequence to start CRDs in the correct order.

---

## Conversion registry population

For each CRD entry with `conversion.paths` declared:

```go
reg.RegisterConversionRulesFromEntry(entry)
```

This builds a `ConversionRules` struct containing the storage version and all declared conversion paths, and stores it keyed by Kind. The conversion handler looks up rules by `ConversionReview.Request.Kind.Kind`.

{{< callout type="note" title="Keyed by Kind, not GVK" >}}
Conversion is always intra-group — all versions of `Website` belong to
`demo.orkestra.io`. The Kind alone is sufficient to identify the conversion
rules. This differs from the admission registry, which is keyed by GVR to
handle the case where the same Kind exists in multiple groups.
{{< /callout >}}

---

## Admission registry population

For each CRD entry with `validation` or `mutation` rules declared:

```go
reg.registerAdmissionRulesFromEntry(entry)
```

Reads `entry.Validation.Rules` and `entry.Mutation.Rules` directly (they live on `CRDEntry`, not under `ReconcilerConfig`). Builds the GVR key from `entry.APITypes.Group`, `.Version`, `.Plural`. Skips entries where `apiTypes.plural` is empty (enrichment not complete — should not occur after `EnrichCRDEntry`).

---

## The Katalog struct

```go
type Katalog struct {
    meta               KatalogMeta
    crds               []orktypes.CRDEntry   // all entries including disabled
    enabled            []orktypes.CRDEntry   // only enabled entries, in start order
    startOrder         []string              // name order after dependency sort
    conversionRegistry katalog.ConversionRegistry
    admissionRegistry  katalog.AdmissionRegistry
}
```

The `Katalog` is passed to:
- `konstructOrkestra` — which uses `enabled` to start per-CRD stacks
- `NewHealthServer` — which uses the registries for /convert, /validate, /mutate
- `ork validate` — which uses `crds` (all) for validation output

---

## ork validate

```bash
ork validate --katalog katalog.yaml
```

Runs the full Katalog loading sequence but does not start the runtime. Outputs:

```
✓ website
    kind: Website
    group: demo.orkestra.io / version: v1alpha1 / plural: websites
    mode: dynamic / workers: 3 / resync: 15s
    validation: 2 rules / mutation: 1 rule / conversion: 2 paths

✓ deployment-governance
    kind: Deployment → enriched from built-in registry
    group: apps / version: v1 / plural: deployments
    mode: dynamic / workers: 2

✗ application
    error: circular dependency detected: application → namespace → application
```

Exit code is 0 when all entries validate successfully. Non-zero on any error. Safe to run in CI against any Katalog or Komposer.
