# The ConfigMap Operator: Kubernetes Without CRDs

*Orkestra Project — April 2026*

---

## Abstract

Every Kubernetes operator requires a Custom Resource Definition. The CRD
defines the API schema, enables kubectl discovery, and provides the object
storage that operators reconcile against. This requirement is a structural
assumption, not a fundamental necessity. Orkestra's data-driven operator
model allows ConfigMaps — resources present in every Kubernetes cluster since
version 1.0 — to serve as the input surface for operator behavior. A ConfigMap
with a label is an operator input. The Katalog attaches behavior to it. The
operator creates derived resources, tracks status in annotations, and
reconciles on every change. No CRD is required. No CRD versioning is required.
No CRD deletion risk exists. This paper describes the model, its properties,
and the conditions under which it is preferable to the standard CRD-based
approach.

---

## 1. The CRD tax

Building an operator with a Custom Resource Definition imposes a fixed cost
that is independent of the operator's complexity.

The CRD schema must be designed and maintained. Fields must be typed. Arrays
must specify item schemas. Required fields must be declared. Schema changes
require careful consideration of backward compatibility. For simple operators
encoding straightforward platform automation — "given this configuration, create
these resources" — the schema design work can exceed the reconcile logic work.

The CRD must be installed before the operator is deployed. In environments with
strict RBAC policies, installing a CRD requires cluster-admin access. The
operator binary and the CRD manifest have a version coupling: a mismatch between
the deployed CRD schema and the schema the operator expects can cause the
operator to malfunction silently.

The CRD creates a cascade deletion risk. `kubectl delete crd pipelines.platform.io`
deletes every `Pipeline` CR in the cluster. An accidental or malicious CRD deletion
destroys all the resources the operator was managing, potentially including
production workloads. This is a structural property of the CRD model, not an
operator implementation deficiency.

The ConfigMap model eliminates all three costs for the operators where it is
appropriate.

---

## 2. ConfigMaps as operator input

A ConfigMap is structured — it holds `data: map[string]string`. It is
namespaced. It is watchable via the Kubernetes watch API. It is available in
every cluster. It is not deletable in a way that cascades to other resources.
It has been stable since Kubernetes 1.0.

Orkestra watches ConfigMaps with a label selector:

```yaml
spec:
  crds:
    pipeline-runner:
      apiTypes:
        kind: ConfigMap
      labelSelector:
        orkestra.io/katalog: pipeline-runner
```

Only ConfigMaps with `orkestra.io/katalog: pipeline-runner` are reconciled.
A user creates an operator input by creating a ConfigMap with this label.
The `data` fields are the spec. The operator creates derived resources — Jobs,
Deployments, Services, Secrets — from the ConfigMap's data values.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: payments-build
  labels:
    orkestra.io/katalog: pipeline-runner
data:
  repo: https://github.com/org/payments
  image: registry.internal/payments
  branch: main
  replicas: "3"
```

The Katalog reconciler accesses `{{ .data.repo }}`, `{{ .data.image }}`, and
`{{ .data.replicas }}` directly. The full template context — `.data.*`, `.metadata.*`,
`.spec.*` — is the unstructured ConfigMap object. `.data.*` replaces `.spec.*` as
the primary spec access path. Everything else is identical to a CRD-based operator.

---

## 3. Status without a status subresource

ConfigMaps do not have a `/status` subresource. Orkestra cannot call
`PATCH /api/v1/namespaces/default/configmaps/payments-build/status` — the
API server returns 404. Status for ConfigMap-based operators is written to
annotations instead.

```yaml
status:
  fields:
    - path: annotations.orkestra.io/phase
      value: "Ready"
      when:
        - field: children.deployment.status.readyReplicas
          operator: exists
    - path: annotations.orkestra.io/lastBuildCommit
      value: "{{ .git.commit }}"
```

The Katalog's `validateStatus()` function detects that `ConfigMap` is a
`Statusless` type and routes status writes to annotation patches rather than
status subresource patches. The Control Center reads annotations alongside
status fields and displays them in the same panel. From the user's perspective,
the phase appears in the same location whether the CR is a custom type or a
ConfigMap.

This is not a limitation but a feature: annotation-based status is readable
without any special tooling. `kubectl get configmap payments-build -o jsonpath=
'{.metadata.annotations.orkestra\.io/phase}'` returns the phase value directly.

---

## 4. Schema evolution

The most significant property of the ConfigMap model is its schema evolution
characteristic. ConfigMap data is `map[string]string`. It has no OpenAPI schema.
Every key is optional from Kubernetes' perspective. The Katalog declares which
keys it uses and what happens when they are absent.

Adding a new field to a ConfigMap-based operator is adding a new optional key.
Existing ConfigMaps without the new key continue to reconcile correctly — Orkestra
evaluates the key as `notExists`, and `when:` conditions or mutation defaults
handle the absent case. No CRD version bump. No migration job. No `kubectl
convert` invocations. The change takes effect on the next reconcile cycle after
the new Katalog is applied.

Deprecating a field follows the same pattern: stop using the key in the Katalog.
Existing ConfigMaps that still include the deprecated key are unaffected — the
unused key is silently ignored.

This schema evolution model is absolute when the ConfigMap is the input surface.
The version concept does not apply. There is no v1, no v2, no conversion. The
Katalog evolves; the ConfigMap format is whatever users put in it.

---

## 5. The ConfigMap is permanent

The cascade deletion risk of CRDs is not a hypothetical concern. In a production
environment with twenty CRDs, `kubectl delete crd` is a destructive operation
that an operator or a script can invoke against the wrong target. Helm uninstall
without `--keep-history` deletes CRDs and all CRs. Cluster decommissioning
scripts that delete operator deployments frequently also delete their CRDs.

A ConfigMap cannot be deleted in a way that cascades. `kubectl delete configmap
payments-build` deletes one ConfigMap and the Kubernetes resources that had owner
references pointing to it. The operator's other ConfigMaps are unaffected. The
CRD — which does not exist in this model — cannot be accidentally deleted.

For operators managing critical infrastructure, this permanence is a reliability
property. The input surface for a payments pipeline operator or a compliance
enforcement operator should not be deletable by a routine cluster cleanup script.
ConfigMaps are not.

---

## 6. When the ConfigMap model is appropriate

The ConfigMap model is the right choice under specific conditions.

**The operator is internal.** If the operator's input resources will only be
created by team members with cluster access and knowledge of the label convention,
the lack of schema enforcement at admission time is acceptable. Validation
happens at reconcile time via Orkestra's validation rules.

**Rapid iteration is required.** A team building a new internal operator — a
namespace provisioner, a secret rotation manager, a certificate renewal operator
— can ship the first version in hours rather than days. No CRD design, no schema
review, no Helm chart update. Create a labeled ConfigMap and apply the Katalog.

**Schema stability is uncertain.** If the operator's input model is still being
discovered — users are adding fields as they find new requirements — the
ConfigMap model allows the schema to evolve organically without migration work.
When the schema stabilizes, it can be formalized into a CRD with the full
OpenAPI schema as documentation of what the operator expects.

**Deletion protection is paramount.** For operators managing resources that
must not be accidentally destroyed, the ConfigMap model provides structural
deletion safety that the CRD model cannot provide without additional webhook
infrastructure.

---

## 7. When the CRD model is appropriate

The ConfigMap model is not universal.

**Admission-time schema validation is required.** A `kubectl apply` with a
missing required field returns 200 OK when using the ConfigMap model. Validation
happens at reconcile time, not admission time. For operators exposed to users
who should receive immediate feedback on invalid inputs — via `kubectl apply`
returning an error — the CRD model with admission webhooks is appropriate.

**API discovery is required.** `kubectl explain pipeline.spec` only works
for CRDs. Teams that build tooling around the operator's API — schemas in IDEs,
auto-completion in CI systems — need a CRD. The ConfigMap model provides no
machine-readable API schema.

**External consumption.** If other teams or external systems will create input
resources and need a stable, versioned API contract, a CRD is the correct
primitive. The Kubernetes API server's versioning guarantees apply only to CRDs.

---

## 8. Conclusion

The ConfigMap data-driven operator model is not a workaround. It is a
first-class input model in Orkestra, supported by the same template rendering,
provider dispatch, cross-operator IPC, status propagation, and control center
observability as CRD-based operators. It trades admission-time schema validation
and API discoverability for zero installation overhead, permanent stability,
schema flexibility, and absolute protection against cascade deletion.

For a significant category of internal operators — platform automation, pipeline
runners, compliance enforcers, resource provisioners built for known internal
users — the ConfigMap model is not merely acceptable. It is the better choice.
The absence of a CRD is not a limitation. It is the feature.
