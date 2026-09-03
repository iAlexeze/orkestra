# Enrichment Targets

Each target fetches one or more secondary Kubernetes objects and embeds them as underscore-prefixed keys on the child map. Notes that read these keys require the corresponding target to be declared.

---

## pods

Embeds `_pods` — a list of pod summaries — on Deployment, StatefulSet, ReplicaSet, or Job children.

```yaml
enrich:
  - pods
```

Each pod summary: `{name, ip, phase, ready, node, restartCount, containers}`.

**Notes unlocked:** `podCount`, `readyPodCount`, `podNames`, `podIPs`, `podPhases`, `podNodes`, `podMaxRestarts`, `hasCrashingPod`, `podCrashLoopDetected`, `podImagePullBackOffDetected`, `podOOMKilledDetected`, `podContainerReasons`, and more.

```yaml
status:
  fields:
    - path: podCount
      value: "{{ podCount .children.deployment }}"
    - path: readyPodCount
      value: "{{ readyPodCount .children.deployment }}"
    - path: hasCrashingPod
      value: "{{ hasCrashingPod .children.deployment }}"
    - path: podMaxRestarts
      value: "{{ podMaxRestarts .children.deployment }}"
```

**Try it:**
```bash
ork init --pack use-cases
cd enrich/01-pod-health
ork run
```

---

## events

Embeds `_warnings` — Kubernetes events filtered to `type=Warning` — on any child resource.

```yaml
enrich:
  - events
```

Each warning: `{reason, message, count, lastTimestamp}`.

**Notes unlocked:** `hasWarnings`, `warningCount`, `firstWarning`, `firstWarning`, `firstWarningReason`, `warningMessages`, `warningReasons`.

```yaml
status:
  fields:
    - path: hasWarnings
      value: "{{ hasWarnings .children.deployment }}"
    - path: firstWarning
      value: "{{ firstWarning .children.deployment }}"
      when:
        - field: "{{ hasWarnings .children.deployment }}"
          equals: "true"
```

Best used with a `when:` gate — events are expensive to fetch and only useful when something is wrong:

```yaml
enrich:
  - events:
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

**Try it:**
```bash
ork init --pack use-cases
cd enrich/02-warning-events
ork run
kubectl apply -f cr-broken.yaml   # bad image → warning events appear in status
```

---

## replicasets

Embeds `_replicaSets` — list of full ReplicaSet objects owned by the Deployment.

```yaml
enrich:
  - replicasets
```

**Notes unlocked:** `deploymentReplicaSetCount`, `deploymentReplicaSets`, `oldDeploymentReplicaSets`.

Useful for surfacing rollout progress — old ReplicaSets still scaling down while the new one scales up.

```yaml
status:
  fields:
    - path: replicaSetCount
      value: "{{ deploymentReplicaSetCount .children.deployment }}"
    - path: oldReplicaSets
      value: "{{ oldDeploymentReplicaSets .children.deployment }}"
```

Gate behind rollout detection — zero cost in steady state:

```yaml
enrich:
  - replicasets:
      when:
        - field: "{{ replicasReady .children.deployment }}"
          equals: "false"
```

**Try it:**
```bash
ork init --pack use-cases
cd enrich/03-rollout-observer
ork run
```

---

## owner

Embeds `_owner` — `{name, kind, uid}` from `metadata.ownerReferences` — on ReplicaSet children.

```yaml
enrich:
  - owner
```

**Notes unlocked:** `replicaSetOwnerName`, `replicaSetOwnerKind`.

```yaml
status:
  fields:
    - path: ownerDeployment
      value: "{{ replicaSetOwnerName .children.replicaset }}"
```

---

## endpoints

Embeds `_endpoints` — list of `{ip, port, ready}` pairs — on Service children.

```yaml
enrich:
  - endpoints
```

**Notes unlocked:** `hasEndpoints`, `serviceEndpoints`, `serviceEndpointCount`, `serviceFirstEndpoint`.

```yaml
status:
  fields:
    - path: hasEndpoints
      value: "{{ hasEndpoints .children.service }}"
    - path: endpointCount
      value: "{{ serviceEndpointCount .children.service }}"
    - path: firstEndpoint
      value: "{{ serviceFirstEndpoint .children.service }}"
```

---

## pvcs

Embeds `_pvcs` — list of full PVC objects — on StatefulSet children. PVCs are resolved from `volumeClaimTemplates`.

```yaml
enrich:
  - pvcs
```

**Notes unlocked:** `statefulSetPVCCount`.

---

## pvc / pvclaim / persistentvolumeclaim

Embeds `_pv` — the bound PersistentVolume — on PersistentVolumeClaim children.

```yaml
enrich:
  - pvc
```

**Notes unlocked:** `pvcBound`, `pvcStorageClass`, `pvcCapacity`, `pvReclaimPolicy`, `pvAccessModes`.

---

## storageclass / sc

Embeds `_storageClass` — the full StorageClass object — on PersistentVolumeClaim children.

```yaml
enrich:
  - storageclass
```

**Notes unlocked:** `pvcStorageClass` (reads from embedded StorageClass).

---

## backingpods

Embeds `_backingPods` — pod summary list matching `spec.selector` — on Service children.

```yaml
enrich:
  - backingpods
```

**Notes unlocked:** `podCount`, `readyPodCount`, `hasCrashingPod` (operating on the backing pod set).

---

## node

Embeds `_node` — `{name, zone, region, instanceType}` — on Pod children.

```yaml
enrich:
  - node
```

**Notes unlocked:** `podNode`, zone and region topology fields.

---

## hpa / horizontalpodautoscaler

Embeds `_currentMetrics` and `_scaleTarget` on HorizontalPodAutoscaler children.

```yaml
enrich:
  - hpa
```

**Notes unlocked:** `noteHPAScaling`, `noteHPAScalingActive`, `hpaCurrentReplicas`, `hpaDesiredReplicas`.

---

## cronjob / cj

Embeds `_activeJobs`, `_lastJob`, and `_lastSuccessfulJob` on CronJob children.

```yaml
enrich:
  - cronjob
```

**Notes unlocked:** `cronJobLastJobName`, `cronJobLastJobSucceeded`, `cronJobLastSuccessfulJobName`.

---

## ingress / ing

Embeds `_loadBalancerIPs` and `_tlsSecrets` on Ingress children.

```yaml
enrich:
  - ingress
```

**Notes unlocked:** `ingressLoadBalancerIP`, `ingressHasLoadBalancer`, `ingressTLSSecrets`.

---

## pv / pvs / persistentvolume

Embeds `_pvc` — the bound PersistentVolumeClaim — on PersistentVolume children.

```yaml
enrich:
  - pv
```

---

## Quick reference

| Target | Aliases | Applies to | Embeds | Cost |
|---|---|---|---|---|
| `pods` | `pod` | Deployment, StatefulSet, ReplicaSet, Job | `_pods` | 1 list-pods |
| `events` | `warnings`, `event`, `ev` | any | `_warnings` | 1 list-events |
| `replicasets` | — | Deployment | `_replicaSets` | 1 list-replicasets |
| `owner` | — | ReplicaSet | `_owner` | parsed from metadata |
| `endpoints` | `ep`, `endpointslice` | Service | `_endpoints` | 1 list-endpointslices |
| `pvcs` | — | StatefulSet | `_pvcs` | 1 list-pvcs |
| `pvc` | `pvclaim`, `persistentvolumeclaim` | PVC | `_pv` | 1 get-pv |
| `storageclass` | `sc`, `storageclasses` | PVC | `_storageClass` | 1 get-storageclass |
| `backingpods` | — | Service | `_backingPods` | 1 list-pods |
| `node` | — | Pod | `_node` | 1 get-node |
| `hpa` | `horizontalpodautoscaler` | HPA | `_currentMetrics`, `_scaleTarget` | 1 get-hpa |
| `cronjob` | `cj`, `cronjobs` | CronJob | `_activeJobs`, `_lastJob`, `_lastSuccessfulJob` | 1-2 gets |
| `ingress` | `ing`, `ingresses` | Ingress | `_loadBalancerIPs`, `_tlsSecrets` | 1 get + N gets |
| `pv` | `pvs`, `persistentvolume` | PV | `_pvc` | 1 get-pvc |

---

**Next →** [Conditional Enrichment](03-conditional.md)
