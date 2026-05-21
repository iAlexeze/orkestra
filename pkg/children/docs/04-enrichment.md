# 04 — Enrichment layers

Enrichment embeds additional Kubernetes state directly into each child object map, under underscore-prefixed keys. Template expressions and note functions can then navigate these keys without making additional API calls at expression evaluation time.

All enrichment is opt-in — controlled by the `enrich` list on the CRD entry in the Katalog:

```yaml
crds:
  myApp:
    enrich: [pods, events, endpoints, pvc, pv, owner, replicasets, pvcs,
             storageclass, backingpods, ingress, node, hpa, cronjob]
```

Each layer is a no-op when its key is absent from `enrich`.

## `_pods` — pod enrichment

**File:** [enrich_pods.go](../enrich_pods.go)  
**Enabled by:** `enrich: [pods]`  
**Applies to:** Deployment, StatefulSet, ReplicaSet, Job

Lists pods matching `spec.selector.matchLabels`, filters to those whose immediate `ownerReference.kind` matches the expected controller kind (so Job pods don't appear in a Deployment's list when they share the same `orkestra-owner` label), and embeds a summary slice:

```
_pods: [
  { name, ip, phase, ready, node, restartCount, ordinal, exitCode, containers: [...] }
]
```

`ordinal` is parsed from the pod name suffix (StatefulSet pods: `name-0`, `name-1`). Non-ordinal pods return `-1`.

`containers` is a per-container summary: `{ name, image, state, reason, ready, restartCount }`. `state` is one of `"Running"`, `"Waiting"`, `"Terminated"`. `reason` is the `waiting.reason` or `terminated.reason` (e.g. `"CrashLoopBackOff"`).

Note functions that consume `_pods`: `podNames`, `podIPs`, `podPhases`, `podNodes`, `podCount`, `readyPodCount`, `podMaxRestarts`, `hasCrashingPod`, `podByOrdinal`, `jobActivePodNames`, and others.

## `_warnings` — warning event enrichment

**File:** [enrich_warnings.go](../enrich_warnings.go)  
**Enabled by:** `enrich: [events]`  
**Applies to:** any resource type

Fetches Kubernetes Warning events scoped to the resource by field selector (`involvedObject.name=<name>,type=Warning`). For workload kinds (Deployment, StatefulSet, ReplicaSet), pod-level Warning events are also aggregated — container failures such as `ImagePullBackOff` and `OOMKilled` are recorded on the Pod, not on the workload itself, so without this aggregation `enrich: [events]` on a Deployment would produce no results for the most common failure modes.

```
_warnings: [
  { reason, message, count, lastTimestamp }
]
```

Note functions: `hasWarnings`, `warningCount`, `firstWarningReason`, `firstWarningMessage`.

## `_endpoints` — endpoint enrichment

**File:** [enrich_endpoints.go](../enrich_endpoints.go)  
**Enabled by:** `enrich: [endpoints]`  
**Applies to:** Service

Fetches the EndpointSlice for the Service (by `kubernetes.io/service-name` label) and embeds a flat list of ready/unready address+port pairs:

```
_endpoints: [
  { ip, port, ready }
]
```

Note functions: `hasEndpoints`, `serviceEndpoints`, `serviceEndpointCount`, `serviceFirstEndpoint`.

## `_pv` — PersistentVolumeClaim enrichment

**File:** [enrich_pvc.go](../enrich_pvc.go)  
**Enabled by:** `enrich: [pvc]`  
**Applies to:** PersistentVolumeClaim

Reads the bound PV from `spec.volumeName` and embeds the full PV object:

```
_pv: { apiVersion, kind, metadata, spec, status }
```

Note functions: `pvcBound`, `pvcStorageClass`, `pvcCapacity`, `pvReclaimPolicy`, `pvAccessModes`.

## `_pvc` — PersistentVolume enrichment

**File:** [enrich_pv.go](../enrich_pv.go)  
**Enabled by:** `enrich: [pv]`  
**Applies to:** PersistentVolume

Reads the bound PVC from `spec.claimRef` (namespace + name) and embeds the full PVC object:

```
_pvc: { apiVersion, kind, metadata, spec, status }
```

Symmetrical to `_pv` enrichment on PVCs. Useful when the operator manages PVs directly and needs to surface the bound PVC's storage class, access modes, or phase.

## `_owner` — ownerReference enrichment

**File:** [enrich_replicasets.go](../enrich_replicasets.go)  
**Enabled by:** `enrich: [owner]`  
**Applies to:** ReplicaSet (any resource with ownerReferences)

Extracts the controller owner from `metadata.ownerReferences` and embeds a concise summary. No API call — data is already present in the object:

```
_owner: { name, kind, uid }
```

## `_replicaSets` — Deployment ReplicaSet enrichment

**File:** [enrich_replicasets.go](../enrich_replicasets.go)  
**Enabled by:** `enrich: [replicasets]`  
**Applies to:** Deployment

Lists all ReplicaSets in the namespace and filters to those owned by the Deployment (by UID). Embeds the full ReplicaSet objects:

```
_replicaSets: [ { ...full ReplicaSet object... } ]
```

## `_pvcs` — StatefulSet PVC enrichment

**File:** [enrich_statefulsets.go](../enrich_statefulsets.go)  
**Enabled by:** `enrich: [pvcs]`  
**Applies to:** StatefulSet

Constructs PVC names deterministically (`<templateName>-<stsName>-<ordinal>` for each `volumeClaimTemplate` × replica count) and fetches each PVC. No label selector needed:

```
_pvcs: [ { ...full PVC object... } ]
```

## `_storageClass` — PVC StorageClass enrichment

**File:** [enrich_storageclass.go](../enrich_storageclass.go)  
**Enabled by:** `enrich: [storageclass]`  
**Applies to:** PersistentVolumeClaim

Reads `spec.storageClassName` and fetches the cluster-scoped StorageClass object:

```
_storageClass: { apiVersion, kind, metadata, provisioner, reclaimPolicy, ... }
```

## `_backingPods` — Service pod enrichment

**File:** [enrich_service_pods.go](../enrich_service_pods.go)  
**Enabled by:** `enrich: [backingpods]`  
**Applies to:** Service

Lists pods matching the Service's `spec.selector` (a flat label map, unlike Deployment's `matchLabels`). Headless and ExternalName services with no selector are skipped. Embeds pod summaries identical in shape to `_pods`:

```
_backingPods: [
  { name, ip, phase, ready, node, restartCount, ordinal, exitCode, containers: [...] }
]
```

## `_loadBalancerIPs` and `_tlsSecrets` — Ingress enrichment

**File:** [enrich_ingress.go](../enrich_ingress.go)  
**Enabled by:** `enrich: [ingress]`  
**Applies to:** Ingress

`_loadBalancerIPs` is a flat list of IP/hostname strings from `status.loadBalancer.ingress`. `_tlsSecrets` fetches each Secret named in `spec.tls[*].secretName`:

```
_loadBalancerIPs: ["1.2.3.4", "example.com"]
_tlsSecrets: [ { ...full Secret object... } ]
```

## `_node` — Pod node enrichment

**File:** [enrich_node.go](../enrich_node.go)  
**Enabled by:** `enrich: [node]`  
**Applies to:** Pod

Reads `spec.nodeName` and fetches the Node (cluster-scoped). Embeds a summary with topology labels commonly needed for status display:

```
_node: { name, zone, region, instanceType }
```

`zone` and `region` come from `topology.kubernetes.io/zone` and `topology.kubernetes.io/region`. `instanceType` comes from `node.kubernetes.io/instance-type`. Node objects are cached within one `ReadChildren` call when multiple pods share the same node.

## `_activeJobs`, `_lastJob`, `_lastSuccessfulJob` — CronJob enrichment

**File:** [enrich_cronjobs.go](../enrich_cronjobs.go)  
**Enabled by:** `enrich: [cronjob]`  
**Applies to:** CronJob

`_activeJobs` comes directly from `status.active` — the ObjectReferences Kubernetes maintains for currently running jobs. `_lastJob` and `_lastSuccessfulJob` are full Job objects found by listing owned Jobs, sorted by `creationTimestamp` descending:

```
_activeJobs: [ { apiVersion, kind, namespace, name, uid } ]
_lastJob: { ...full Job object... }
_lastSuccessfulJob: { ...full Job object... }
```

## `_currentMetrics` and `_scaleTarget` — HPA enrichment

**File:** [enrich_hpa.go](../enrich_hpa.go)  
**Enabled by:** `enrich: [hpa]`  
**Applies to:** HorizontalPodAutoscaler

`_currentMetrics` reshapes `status.currentMetrics` into a normalised list across all four metric source types (Resource, External, Pods, Object). `_scaleTarget` extracts `spec.scaleTargetRef`:

```
_currentMetrics: [ { type, name, current } ]
_scaleTarget: { name, kind, apiVersion }
```

## Adding a new layer

1. Create [enrich_\<name\>.go](../) in this package.
2. Write an `enrichGroupWith<Name>(ctx, kube, m, crd)` function that iterates over the map and embeds data under a `_<name>` key. Gate it with `enrichmentEnabled("<key>", crd)`.
3. Add a call in [children.go](../children.go) `ReadChildren` for the relevant resource type(s).
4. Register the enrichment key in the `enrichmentMeta` map in [builtins.go](../builtins.go) so that `enrichmentEnabled`, `IsValidEnrichmentTarget`, and `SupportedEnrichmentGroups` all know about it:
   - If the resource is new to enrichment entirely, add a `"<kind>": {Target: true}` entry.
   - If the key is a context-specific name that differs from the kind's own name, plural, or shorthands (e.g. `"owner"` on ReplicaSet, `"pvcs"` on StatefulSet, `"replicasets"` on Deployment, `"backingpods"` on Service) — add it to `EnrichKeys` in the `enrichmentMeta` entry for that kind.
   - `enrichmentMeta` is intentionally separate from `BuiltInKind` so Kubernetes API identity and enrichment configuration can evolve independently.
5. Add note functions in the matching `kube_<name>.go` file under [pkg/note](../../note/) that navigate the new key.
6. Document the note functions in the corresponding `docs/` file under [pkg/note/docs](../../note/docs/) so they appear in `ork notes catalog` (generated by `make generate-notes`).
7. Add a fixture katalog in [pkg/note/fixture](../../note/fixture/) to exercise the notes end-to-end.

→ Next: [05-builtins.md](05-builtins.md)
