# enrich

`enrich` is a list of enrichment targets that Orkestra fetches and embeds into each child resource map at reconcile time. The embedded data is available in status templates and note functions as underscore-prefixed keys on the child object.

```yaml
enrich:
  - pods       # embed live pod list into .children.deployment._pods
  - events     # embed warning events into ._warnings
```

Enrichment is opt-in and per-CRD. Each target is a no-op when its key is absent.

## Valid targets

You can write any accepted identifier — the canonical name, plural, or a shorthand. All aliases for the same target are listed below.

| Target key(s) | Applies to | Embeds | Note functions |
|---------------|-----------|--------|----------------|
| `pods`, `pod` | Deployment, StatefulSet, ReplicaSet, Job | `_pods` — list of `{name, ip, phase, ready, node, restartCount, containers}` | `podCount`, `readyPodCount`, `podNames`, `podIPs`, `hasCrashingPod`, `podMaxRestarts`, … |
| `events`, `warnings`, `event`, `ev` | any | `_warnings` — list of `{reason, message, count, lastTimestamp}` filtered to `type=Warning` | `hasWarnings`, `warningCount`, `firstWarningReason`, `firstWarningMessage` |
| `owner` | ReplicaSet | `_owner` — `{name, kind, uid}` from `metadata.ownerReferences` | `replicaSetOwnerName`, `replicaSetOwnerKind` |
| `replicasets` | Deployment | `_replicaSets` — list of full ReplicaSet objects owned by the Deployment | `deploymentReplicaSetCount`, `deploymentReplicaSets`, `oldDeploymentReplicaSets` |
| `pvcs` | StatefulSet | `_pvcs` — list of full PVC objects, resolved deterministically from `volumeClaimTemplates` | `statefulSetPVCCount` |
| `pvc`, `pvclaim`, `persistentvolumeclaim` | PersistentVolumeClaim | `_pv` — the bound PersistentVolume object | `pvcBound`, `pvcStorageClass`, `pvcCapacity`, `pvReclaimPolicy`, `pvAccessModes` |
| `pv`, `pvs`, `persistentvolume` | PersistentVolume | `_pvc` — the bound PersistentVolumeClaim object | (standard field access on `_pvc`) |
| `storageclass`, `sc`, `storageclasses` | PersistentVolumeClaim | `_storageClass` — the full StorageClass object from `spec.storageClassName` | `pvcStorageClass` |
| `backingpods` | Service | `_backingPods` — pod summary list matching `spec.selector` | `podCount`, `readyPodCount`, `hasCrashingPod` (on the backing set) |
| `endpoints`, `ep`, `endpointslice` | Service | `_endpoints` — list of `{ip, port, ready}` pairs from the EndpointSlice | `hasEndpoints`, `serviceEndpoints`, `serviceEndpointCount`, `serviceFirstEndpoint` |
| `node` | Pod | `_node` — `{name, zone, region, instanceType}` from the node the pod is scheduled on | `podNode`, node topology fields |
| `hpa`, `horizontalpodautoscaler` | HorizontalPodAutoscaler | `_currentMetrics` — normalised metric list; `_scaleTarget` — `{name, kind, apiVersion}` | `noteHPAScaling`, `noteHPAScalingActive`, `hpaCurrentReplicas`, `hpaDesiredReplicas` |
| `cronjob`, `cj`, `cronjobs` | CronJob | `_activeJobs` — ObjectReferences from `status.active`; `_lastJob` and `_lastSuccessfulJob` — full Job objects | `cronJobLastRunTime`, `cronJobNextRunTime`, `cronJobActive` |
| `ingress`, `ing`, `ingresses` | Ingress | `_loadBalancerIPs` — list of IP/hostname strings; `_tlsSecrets` — list of full TLS Secret objects | `ingressLoadBalancerIP`, `ingressHasLoadBalancer`, `ingressTLSSecrets` |

## Access pattern

Enriched data is accessible in any template expression via the child object:

```yaml
status:
  fields:
    - path: podCount
      value: "{{ podCount .children.deployment }}"
    - path: hasWarnings
      value: "{{ hasWarnings .children.deployment }}"
    - path: replicaSetCount
      value: "{{ deploymentReplicaSetCount .children.deployment }}"
```

The child key in `.children` is the CRD name as declared in `spec.crds` (e.g. `deployment`, `statefulset`, `mypod`). The underscore-prefixed enrichment keys (`_pods`, `_warnings`, etc.) are embedded directly on that object.

## Conditional enrichment

Enrichment targets support an optional `when:` gate. When the condition fails, Orkestra skips the API call for that target entirely — the enriched key simply does not appear on the child object that reconcile cycle.

```yaml
enrich:
  - pods                    # always enriches pods
  - events:                 # only fetch events when deployment is degraded
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
  - replicasets:            # only visible during rollouts or debug mode
      when:
        - field: spec.debug
          equals: "true"
```

`when:` accepts the same condition operators as any other `when:` block — including template expressions that call note functions. `anyOf:` is also supported for OR semantics.

```yaml
enrich:
  - events:
      anyOf:
        - field: "{{ hasCrashingPod .children.deployment }}"
          equals: "true"
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

**When to use conditional enrichment:** each enrichment target is one or more Kubernetes API calls per reconcile cycle. For high-frequency operators or CRDs with many replicas, skipping enrichment you only need in degraded states (events, replicasets) measurably reduces load on the API server.

**Evaluation order:** enrichment conditions are evaluated after the main reconcile runs and after children are read — so `.children.*` fields are available in the condition expressions.

## Template expressions in `when:` fields

`when:` conditions accept Go template expressions in the `field:` value. When the field contains `{{`, Orkestra evaluates it through the full note FuncMap (the same functions available in `status.fields`) and uses the string result for the operator comparison.

This applies to all `when:` blocks — resource provisioning, status fields, and enrichment gates:

```yaml
# Resource only created when deployment is fully ready
services:
  - name: "{{ .metadata.name }}-lb"
    port: "443"
    when:
      - field: "{{ replicasReady .children.deployment }}"
        equals: "true"

# Status field only written when a crashing pod exists
status:
  fields:
    - path: crashReason
      value: "{{ firstWarningMessage .children.deployment }}"
      when:
        - field: "{{ hasCrashingPod .children.deployment }}"
          equals: "true"
```

A `field:` that is a plain dot-path (e.g. `spec.replicas`) continues to use `NavigateDotPath` as before — no change in behaviour.

## enrichAll

`enrichAll: true` enables all supported enrichment targets for this CRD. It is mutually exclusive with the `enrich` list.

```yaml
enrichAll: true   # enrich everything — development / debugging shorthand
```

`enrichAll` does not support conditional gates. Use the `enrich` list with `when:` when you need per-target control.
