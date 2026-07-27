# Additional Fields

`idp.fields` exposes `spec.*` — but not everything a form should collect belongs in `spec`. A team name, an environment, a feature flag are metadata concepts in Kubernetes: labels and annotations, not workload data the operator computes with. `idp.additionalFields` closes that gap — it exposes label and annotation keys as self-service form fields, written to `metadata.labels`/`metadata.annotations` on apply instead of `spec`.

---

## Why this matters beyond convenience

Once a value lives in `metadata.annotations`, every template expression Orkestra already supports can read it back — `when:` conditions, `value:` resolution, `normalize:`, profile selection — via the `getLabel`/`getAnnotation`/`hasAnnotation`/`hasLabel` notes. That makes annotations a **schema-free configuration channel**: a platform team can add a new decision point — a rollout gate, a tier, a feature flag — without a CRD schema change and without a code change.

This is the same idiom `nginx-ingress-controller` and `cert-manager` already use (`nginx.ingress.kubernetes.io/rewrite-target`, `cert-manager.io/cluster-issuer`) — annotations as the extension point every serious controller reaches for once a field stops being "data the workload needs" and becomes "a knob an operator reads." `idp.additionalFields` is what makes that annotation settable through a form, instead of requiring a hand-edited CR.

```yaml
spec:
  crds:
    application:
      idp:
        enabled: true
        additionalFields:
          labels:
            team:
              label: "Team"
              placeholder: "team-payments"
          annotations:
            canary.myorg.io:
              label: "Enable canary rollout"
              type: boolean

      operatorBox:
        onReconcile:
          deployments:
            - name: "{{ .metadata.name }}"
              replicas: '{{ ternary (eq (getAnnotation . "canary.myorg.io") "true") "1" "3" }}'
```

No new spec field, no CRD schema change — the platform team gets an annotation-driven extension point using notes that already ship.

---

## Choosing labels vs. annotations

The same question Kubernetes itself answers for every native resource:

- **Label** — if you would ever want to *select* on it (filter, group, build a dashboard by it). Team, environment, and tier are almost always labels in real clusters, for exactly this reason.
- **Annotation** — everything else: feature flags, external reference IDs (a Jira ticket, a Terraform run ID), free-form configuration. Not used for selection, not indexed.

Both go through the same `type`/`enum` presentation hints as `idp.fields` — the difference is only the write target.

---

## A gotcha with boolean flags

A checkbox field in the Control Center form always submits an explicit value — unchecked submits `"false"`, not an absent annotation. That means a value-based check (`getAnnotation . "canary.myorg.io" | equals "true"`) is correct for a form-driven boolean field; `hasAnnotation` is not — it treats *any* non-empty value, including the string `"false"`, as present. Reserve `hasAnnotation` for annotations that are genuinely presence-only (set by automation, never toggled through a checkbox).

---

## See also

→ [idp.additionalFields schema reference](../../reference/schema/02-katalog/02-crd-entry.md#idpadditionalfields)
→ [Kubernetes notes — getLabel, getAnnotation, hasAnnotation](../notes/index.md)
→ [use-cases/idp example pack](https://github.com/orkspace/orkestra/tree/main/examples/use-cases/idp) — `01-form` drives its whole flow from `additionalFields` and minimal `spec`
→ [Required fields are enforced automatically](../../reference/schema/02-katalog/07-validation.md#required-fields-are-enforced-automatically) — `required: true` synthesizes server-side enforcement, not just a form hint
→ [IDP-aware messages for hand-written rules](../../reference/schema/02-katalog/07-validation.md#idp-aware-messages-for-hand-written-rules) — write `message:` using the field's `label:`, not its raw `spec.*` path
