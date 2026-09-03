# Serve API

The Serve API is the Gateway's intent delivery surface. It accepts flat, human-readable fields from a caller and produces a fully validated, provenance-stamped Kubernetes object applied to the cluster — without the caller needing any knowledge of CRDs, API groups, namespaces, or Kubernetes object shape.

The caller declares what they want. The Gateway handles the rest.

---

## Intent delivery chain

Every create or update request passes through six stages in sequence:

**1. Target resolution** — The caller names a target (or alias). The Gateway resolves it to the specific CRD declared in the Katalog. Aliases let you expose the same CRD under multiple names with independent token scoping.

**2. Token check** — The named token is verified against the target and operation. Tokens declare which targets they can reach, which operations they can perform, and which namespaces they can operate in. A request with an invalid or insufficiently scoped token is denied before any CR is constructed.

**3. CR construction** — The flat intent fields are mapped to CR fields using the `serve.fields` declarations in the Katalog. Required fields are enforced. Type coercions and enum constraints are applied. The result is a fully shaped Kubernetes object.

**4. Provenance** — The Gateway stamps `orkestra.orkspace.io/serve-target` and `orkestra.orkspace.io/serve-alias` annotations on the CR. These record the delivery surface that created the object and survive through the full object lifecycle.

**5. Admission validation** — Validation and mutation rules are evaluated against the constructed CR. Deny-action violations return an error immediately — the CR is never written. Mutation rules apply their defaults and overrides. Warn-action violations are included in the response.

**6. Apply and respond** — The CR is applied to the cluster via server-side apply. The response is shaped by `serve.config.response` — either the full CR, a payload subset, or a mix of both.

---

## Tokens

Tokens are the authorization primitive. Each token is declared in the Katalog:

```yaml
serve:
  tokens:
    - name: platform-team
      targets:
        - name: myapp
          operations: [create, update, delete, get, list]
    - name: readonly
      targets:
        - name: myapp
          operations: [get, list]
```

Tokens scope access to targets, operations, and optionally namespaces. A token with namespace restrictions can only read or write CRs in those namespaces. The Gateway enforces this at the token check stage — no namespace bypass is possible.

---

## Aliases

Aliases expose the same CRD under a different name with a different token set:

```yaml
serve:
  aliases:
    - name: myapp-preview
      target: myapp
      tokens:
        - name: preview-ci
          targets:
            - name: myapp-preview
              operations: [create, update]
```

A caller using `myapp-preview` sees only the fields and operations that alias permits. The underlying CRD, its validation rules, and its reconciler are the same. Provenance annotations record which alias was used, so you can always trace what surface created a given CR.

---

## Field declarations

`serve.fields` maps the caller's flat keys to CRD spec paths:

```yaml
serve:
  fields:
    - name: workloadType
      path: spec.workloadType
      required: true
      type: enum
      values: [app, job, cronjob]

    - name: repoURL
      path: spec.source.repoURL
      required: false
```

Required fields that are absent generate an immediate deny. Enum fields generate an `in` membership check. Fields not declared in `serve.fields` are not accepted — the schema the caller sees is exactly what the operator author declared.

---

## Response shaping

The response the caller receives is controlled by `serve.config.response`:

```yaml
serve:
  config:
    response:
      default: false          # omit the full CR from the response
      payload:
        url: "{{ .spec.source.repoURL }}"
        environment: "{{ .spec.environment }}"
```

`default: false` omits the raw CR. `payload` evaluates template expressions against the constructed CR and returns the results. A caller building a CI/CD integration or a platform UI receives exactly the fields they need — nothing more.

---

## Read operations

For `get`, `list`, and `delete` the Gateway performs a token check and target resolution, then proxies the operation to the cluster. The same response shaping applies to `get` and `list` results — a caller using a read token sees only the payload fields declared for their alias.

---

## Operations

| Operation | What the Gateway does |
|-----------|----------------------|
| `create` | Constructs CR, runs admission, applies via SSA |
| `update` | Same as create — SSA is idempotent |
| `get` | Token check, then cluster GET with response shaping |
| `list` | Token check, then cluster LIST with response shaping applied per item |
| `delete` | Token check, then cluster DELETE |
