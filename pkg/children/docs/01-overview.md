# 01 — What ReadChildren produces

`ReadChildren` is called once per reconcile cycle, after the child resources have been created or updated. It returns a `map[string]interface{}` that is injected into the template resolver under the `.children` key.

## Shape of the map

For each declared resource type the map gets two entries:

```
children["deployments"]   → map[name → objectMap]   (all resources of this type)
children["deployment"]    → objectMap                (first/only resource, shorthand)
```

`objectMap` is the raw `unstructured.Unstructured.Object` as returned by the Kubernetes dynamic client, with enrichment layers embedded under underscore-prefixed keys.

## Using it in status expressions

**Single resource** — the shorthand key:

```yaml
status:
  fields:
    - path: readyReplicas
      value: "{{ .children.deployment.status.readyReplicas }}"
    - path: image
      value: "{{ containerImage .children.deployment }}"
```

**Multiple resources** — index into the plural map by name:

```yaml
- path: apiReady
  value: '{{ (index .children.deployments "my-site-api").status.readyReplicas }}'
```

**Enriched data** — accessed via note functions or directly:

```yaml
- path: podNames
  value: "{{ podNames .children.deployment }}"
- path: warnings
  value: "{{ hasWarnings .children.deployment }}"
```

## Missing keys

The resolver is configured with `missingkey=zero` — any absent path resolves to `""` rather than raising an error. This is intentional: child resources may not have a status subresource populated yet on the first reconcile cycle. The reconciler will retry and the value will appear once Kubernetes populates it.

## Available child types

Every resource type declared in a Katalog's `operatorBox` (onCreate or onReconcile) gets a map entry. If a type is not declared, its key is absent — `ReadChildren` never lists resources it has not been told about.

Supported built-in types: Deployment, StatefulSet, ReplicaSet, Service, Secret, ConfigMap, Job, CronJob, Pod, ServiceAccount, Namespace, Ingress, HorizontalPodAutoscaler, PersistentVolumeClaim, PersistentVolume, Role, RoleBinding, ClusterRole, ClusterRoleBinding, and custom resources.

→ Next: [02-reading.md](02-reading.md)
