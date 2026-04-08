---
title: "Versioning"
weight: 149
---

# Versioning in Orkestra

When a CRD has multiple versions — `v1alpha1`, `v1beta1`, `v1` — Kubernetes
stores all objects internally as a single storage version. When a client
reads an object, the API server converts it on the fly. Most operator
frameworks receive whatever version the API server decides to send, which
is typically the storage version regardless of what the user originally
wrote.

Orkestra works differently, and the reason comes directly from its architecture.

---

## One reconciler per version

Each version of a CRD is a separate entry in the Katalog:

```yaml
crds:
  website-v1alpha1:
    apiTypes:
      group: demo.orkestra.io
      version: v1alpha1
      kind: Website
      plural: websites
    reconciler:
      default: true
      onCreate:
        deployments:
          - image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"

  website-v1:
    apiTypes:
      group: demo.orkestra.io
      version: v1
      kind: Website
      plural: websites
    reconciler:
      default: true
      onCreate:
        deployments:
          - image: "{{ .spec.image }}"
            replicas: "{{ .spec.replicas }}"
            seo:
              enabled: "{{ .spec.seo.enabled }}"
```

{{< callout type="tip" title="the orkestra magic" >}}
`website-v1alpha1` gets its own informer, its own workqueue, its own worker
pool, and its own reconciler. So does `website-v1`. They are independent
operators that happen to share the same Kind.
{{< /callout >}}

## How each reconciler gets its version

When Orkestra registers the informer for `website-v1alpha1`, it registers it
specifically for `demo.orkestra.io/v1alpha1`. The informer watches the API
server using that exact GVK. The Kubernetes API server, when it delivers
objects to this watch, converts them to `v1alpha1` — because that is the
version being watched.

The reconciler for `website-v1alpha1` therefore always receives objects in
`v1alpha1` format. It sees `spec.theme`. It does not see `spec.seo`. It
reconciles exactly the schema the user declared.

The reconciler for `website-v1` always receives objects in `v1` format. It
sees `spec.seo`. It does not see `spec.theme`.

{{< callout type="tip" title="the orkestra magic" >}}
Neither reconciler needs to know about the other.
{{< /callout >}}

    Neither needs to handle version negotiation. 
    
    The informer factory takes care of it by watching the declared version.

## Why this matters

This behaviour has a practical consequence: **your reconcile templates are
written against one version and they stay valid for that version**.

If you add a field in `v1` — say, `spec.autoscaling.enabled` — the `v1alpha1`
reconciler is not affected. It never sees that field. Its templates never
need to account for it. You write the `v1alpha1` templates against the
`v1alpha1` schema and they remain correct indefinitely.

The alternative — a single reconciler that receives the storage version and
must handle all versions — creates a reconciler that grows with every version
addition. Fields from `v1alpha1` may be absent in `v1`. Fields in `v1` may
have no equivalent in `v1alpha1`. The reconciler becomes a version-handling
tangle.

Orkestra's per-version operator model avoids this entirely.

## Conversion sits alongside reconciliation

When you add conversion rules to the `v1` entry, those rules are evaluated
by the same template resolver that evaluates reconcile templates — against
the same object representation, in the same runtime:

```yaml
- name: website-v1
  conversion:
    storageVersion: v1
    paths:
      - from: v1alpha1
        to: v1
        spec:
          image: "{{ .spec.image }}"
          seo:
            enabled: false    # v1alpha1 objects have no seo field
```

The conversion path is just another template — field values from the source
object are resolved into the target object's format. The resolver already
knows how to evaluate `{{ .spec.image }}` against any object map. 

{{< callout type="tip" >}}
Conversion is not a separate infrastructure concern. It lives where it belongs: next to the reconcile logic for that version.
{{< /callout >}}

## The informer's role

The informer is what makes all of this work at the infrastructure level.
When Orkestra starts, it creates one informer per Katalog entry. Each
informer opens a watch connection to the API server for its specific GVK.
The API server is responsible for serving objects in the requested version —
converting from storage as needed.

The result is that each informer's cache contains objects in exactly the
version it declared. The reconciler reads from that cache. It never sees a
different version. It never needs to detect or handle version mismatches.

{{< callout type="tip" title="the orkestra magic" >}}
This is the architectural reason versioning in Orkestra is clean: the version
boundary is enforced at the informer level, before objects reach the
reconciler. 

Each version is its own complete operator. They share the same underlying infrastructure.
The logic is always version-specific.
{{< /callout >}}
