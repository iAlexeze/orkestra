---
title: "Status Management"
weight: 145
---

# Status Management

Orkestra writes CR status automatically. Every managed CR gets a machine-readable
health signal after every reconcile. For operators that need richer status — replica
counts, endpoints, phase strings, child resource state — a declarative status block
in the Katalog provides it without writing a single line of Go.

Status in Orkestra works in three layers, each building on the previous one.

---

## Layer 1 — Standard conditions (automatic, always)

After every reconcile cycle — success or failure — Orkestra patches the CR's
`/status` subresource with a standard Kubernetes `Ready` condition and
`observedGeneration`. No Katalog declaration required.

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: ""
      lastTransitionTime: "2026-03-29T09:02:33Z"
      observedGeneration: 1
  observedGeneration: 1
```

On failure:

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: ReconcileError
      message: "deployment: image pull failed: rpc error..."
      lastTransitionTime: "2026-03-29T09:03:01Z"
      observedGeneration: 1
```

This makes `kubectl get websites` immediately informative without any Katalog changes.
External tools that watch for `Ready=True` on custom resources — ArgoCD health checks,
custom dashboards, monitoring scripts — work out of the box.

{{< callout type="note" title="CRD must declare the status subresource" >}}
Layer 1 requires `subresources: status: {}` in the CRD definition.
Without it, Orkestra's status patch receives a 404 from the API server
and is silently ignored — reconciliation is not affected.
{{< /callout >}}

    Add to each version in the CRD:

```yaml
spec:
  versions:
    - name: v1alpha1
      subresources:
        status: {}   # enables /status subresource
```

To opt out of automatic conditions for a specific CRD:

```yaml
operatorBox:
  status:
    conditions: false
```

This is rarely needed. Only use it when the CRD schema explicitly forbids a
`conditions` field or when conditions are managed entirely by Go hooks.

---

## Layer 2 — Declarative status fields

Declare status fields in the Katalog. Values support the same Go template
expressions as `onCreate` templates — resolved against the live CR at reconcile time.

```yaml
operatorBox:
  status:
    fields:
      - path: phase
        value: "Running"

      - path: observedReplicas
        value: "{{ .spec.replicas }}"

      - path: endpoint
        value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

      - path: version
        value: "{{ .spec.version }}"

      - path: database.host        # nested — becomes status.database.host
        value: "{{ .spec.host }}"

      - path: database.port
        value: "{{ .spec.port }}"
```

After a successful reconcile:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      ...
  observedGeneration: 1
  phase: Running
  observedReplicas: "2"
  endpoint: my-site.orkestra.svc.cluster.local
  version: "1.25"
  database:
    host: db.platform.svc
    port: "5432"
```

**Paths are relative to `status`.** `phase` writes to `status.phase`.
`database.host` writes to `status.database.host`. Dot-notation works at any depth.

**Fields are only written on successful reconcile.** If the reconcile fails
partway through, the declarative fields are not updated — only the Ready condition
is written (as `False`). This prevents misleading status when the cluster state
is partial.

{{< callout type="tip" title="Document your status fields in the CRD schema" >}}
Declare the status fields in the CRD's OpenAPIV3Schema to enable `kubectl`
validation and avoid `unknown field` warnings. Use
`x-kubernetes-preserve-unknown-fields: true` on the status object to accept
any field without enumerating every one:
{{< /callout >}}

```yaml
status:
  type: object
  x-kubernetes-preserve-unknown-fields: true
```
---

## Layer 3 — Child resource status propagation

After reconcile templates create child resources, Orkestra reads them back and
makes their status available in the template resolver under the `children` key.
Status fields can then reference the live state of child resources.

```yaml
operatorBox:
  status:
    fields:
      # From the Deployment
      - path: readyReplicas
        value: "{{ readyReplicas .children.deployment }}"

      - path: availableReplicas
        value: "{{ availableReplicas .children.deployment }}"

      # From the Service
      - path: loadBalancerIP
        value: "{{ serviceLoadBalancerIP .children.service }}"
```

**Access patterns:**

`{{ .children.deployment }}` — the first (or only) Deployment created by this CR. For operators with a single Deployment this is the ergonomic path.

`{{ readyReplicas (index .children.deployments "my-site-api") }}` — access by exact Kubernetes name when multiple Deployments exist.

The same patterns work for all resource types:

| Singular | Plural |
|---|---|
| `.children.deployment` | `.children.deployments` |
| `.children.service` | `.children.services` |
| `.children.secret` | `.children.secrets` |
| `.children.configMap` | `.children.configMaps` |
| `.children.job` | `.children.jobs` |
| `.children.cronJob` | `.children.cronJobs` |
| `.children.pod` | `.children.pods` |
| `.children.serviceAccount` | `.children.serviceAccounts` |

**First reconcile behaviour:** When a CR is first created, the Deployment exists
but Kubernetes has not yet populated `status.readyReplicas` — it is zero or absent.
Missing fields resolve to `""`. The status field writes an empty string. On the next
reconcile cycle (triggered by the Deployment's status change event), the value is
populated. This is expected eventual consistency.

{{< callout type="note" title="API calls per reconcile" >}}
Layer 3 makes one GET per child resource declared in the Katalog — bounded
by the number of template declarations, not by the number of CRs. An operator
with one Deployment and one Service declaration makes two GETs per reconcile.
These are reads against the API server, not the informer cache, because child
resource informers are not registered by default.
{{< /callout >}}

---

## Complete example

CRD with status subresource declared:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: websites.demo.orkestra.io
spec:
  group: demo.orkestra.io
  versions:
    - name: v1alpha1
      served: true
      storage: true
      subresources:
        status: {}
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                image:
                  type: string
                replicas:
                  type: integer
            status:
              type: object
              x-kubernetes-preserve-unknown-fields: true
  names:
    kind: Website
    plural: websites
    singular: website
  scope: Namespaced
```

Katalog with all three layers:

```yaml
- name: website
  apiTypes:
    group: demo.orkestra.io
    version: v1alpha1
    kind: Website
    plural: websites

  operatorBox:
    default: true

    status:
      # conditions: true ← default, no need to declare
      fields:
        # Layer 2 — from the CR spec
        - path: phase
          value: "Running"
        - path: observedReplicas
          value: "{{ .spec.replicas }}"
        - path: endpoint
          value: "{{ .metadata.name }}.{{ .metadata.namespace }}.svc.cluster.local"

        # Layer 3 — from child resource status
        - path: readyReplicas
          value: "{{ readyReplicas .children.deployment }}"
        - path: availableReplicas
          value: "{{ availableReplicas .children.deployment }}"

    onCreate:
      deployments:
        - image: "{{ .spec.image }}"
          replicas: "{{ .spec.replicas }}"
          reconcile: true
      services:
        - port: "80"
          targetPort: "80"
          reconcile: true
```

After a successful reconcile with two ready replicas:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: ""
      lastTransitionTime: "2026-03-29T09:02:33Z"
      observedGeneration: 1
  observedGeneration: 1
  phase: Running
  observedReplicas: "2"
  endpoint: my-site.orkestra.svc.cluster.local
  readyReplicas: "2"
  availableReplicas: "2"
```

---

## Status in hooks

Go hooks have full access to the CR object and the Kubernetes client. Write
status directly from a hook using the dynamic client or the typed client:

```go
func (r *websiteReconciler) OnReconcile(ctx context.Context, obj *apiv1.Website) error {
    // ... reconcile logic ...

    // Update status with rich information from the reconcile
    obj.Status.Phase = "Running"
    obj.Status.ReadyReplicas = actualReadyCount

    // Patch the status subresource
    _, err := r.client.Resource(websiteGVR).
        Namespace(obj.Namespace).
        UpdateStatus(ctx, toUnstructured(obj), metav1.UpdateOptions{})
    return err
}
```

Hooks and declarative status fields are not mutually exclusive. When both are
declared, Orkestra applies the declarative fields after the hook completes — the
hook's status writes are preserved (the patch is merged, not replaced).

---

## Disabling status management

To disable all automatic status management for a CRD:

```yaml
operatorBox:
  status:
    conditions: false
    fields: []
```

With both disabled, Orkestra makes no status patches. The CR's status is entirely
managed by Go hooks or left empty. Use this when the CRD's schema strictly controls
status through a typed struct that Orkestra's merge patch would conflict with.
