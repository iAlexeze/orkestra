# validation

Declarative validation rules evaluated at admission time (via webhook) and at reconcile time.
Declared on a CRDEntry or inside a Motif's `admission` block.

```yaml
validation:
  rules:
    - field: spec.replicas
      operator: lte
      value: "10"
      valueType: int
      message: "replicas must not exceed 10"

    - field: spec.engine
      equals: postgres
      message: "only postgres engine is supported"
      action: deny

    - field: spec.image
      prefix: "myregistry.example.com/"
      message: "image must come from the internal registry"
      action: warn

    - field: spec.productionApproval
      operator: exists
      message: "production deploys require an approval ticket"
      action: deny
      when:
        - field: spec.environment
          equals: production

    - field: spec.domain
      operator: exists
      message: "spec.domain is required for cert workloads"
      action: deny
      when:
        - field: spec.workloadType
          equals: cert
      anyOf:
        - field: spec.workloadType
          equals: cert
        - field: spec.workloadType
          equals: monitoring
```

## `validation.include`

Pull rules from an external file to keep the Katalog compact. The file contains a `rules:` key with the same list structure as inline rules.

```yaml
validation:
  include: ./admission/apprequest.yaml   # relative to the katalog file
  rules:
    - field: spec.tier                   # appended after included rules
      operator: in
      value: "free,pro,enterprise"
      action: deny
```

`admission/apprequest.yaml`:

```yaml
rules:
  - field: spec.team
    operator: exists
    message: "spec.team is required"
    action: deny
  - field: spec.environment
    operator: in
    value: "staging,production"
    message: "spec.environment must be staging or production"
    action: deny
```

Included rules come first. Inline `rules:` append after. The `include:` path is resolved relative to the katalog file's directory — the katalog can be run from any working directory. The field is cleared from the runtime bundle after expansion.

## `validation.rules`

Each rule describes one check. Rules are evaluated in order.

| Field | Required | Description |
|-------|----------|-------------|
| `field` | yes | Dot-notation path in the CR, or a Go template expression (see below). |
| `message` | yes | Error or warning message. Supports Go templates resolved against the CR and notes FuncMap. |
| `action` | no | `deny` (default) — reject; `warn` — allow but log a warning |
| `operator` + `value` | yes* | Explicit comparison (see operators). `value` supports Go templates. |
| `valueType` | no | `string` (default), `int`, `float`, `bool` |
| `when` | no | All conditions must pass for this rule to be evaluated (AND). Empty means unconditional. Conditions support Go template expressions via `EvaluateWhen`. |
| `anyOf` | no | At least one condition must pass for this rule to be evaluated (OR). When both `when` and `anyOf` are declared, both blocks must pass. |
| `link` | no | The `serve.fields`/`serve labels/annotations` key this rule concerns, when `field:` isn't already a plain, self-describing path — see [Linking a rule to its form field](#linking-a-rule-to-its-form-field-link) below. |

*Use either an operator+value pair or a shorthand field.

`when` and `anyOf` use the same `Condition` type as resource templates — see [06-when-conditions.md](06-when-conditions.md) for the full operator reference.

### Required fields are enforced automatically

`required: true` on an `serve.fields` or `serve labels/annotations` entry isn't just a form hint — at katalog load time, every `required: true` entry gets an implicit `exists` rule synthesized automatically, with `message:` already matching the field's `label:`. This is enforced at the API server for every client — the Control Center form, `curl`, a CI pipeline, a custom UI, `kubectl apply` — not only the one that happens to render a required-field asterisk. You don't hand-write a `validation.rules` entry for plain "this field must be present" checks — marking the field required is enough:

```yaml
serve:
  fields:
    targetRevision:
      label: "Branch / Tag"
      required: true
# → synthesizes: { field: spec.targetRevision, operator: exists,
#                  message: "Branch / Tag is required", action: deny }
```

`serve.labels`/`.annotations` entries synthesize the same way, through `getLabel`/`getAnnotation` rather than a raw dot-path — required annotation keys with dots (the Kubernetes-recommended `prefix/name` shape) are handled correctly without you needing to know that dot-path resolution would otherwise misparse them.

### Enum fields are validated automatically

`type: enum` on an `serve.fields` or `serve labels/annotations` entry also synthesizes a rule — an `in` check against the declared `enum:` list, with the same label-matching message. Membership is checked only when the field has a value: an enum field that isn't also `required: true` can still be omitted entirely, it just can't be set to something outside the list.

```yaml
serve:
  fields:
    workloadType:
      label: "Workload Type"
      type: enum
      enum: [app, cert, monitoring, infra]
      required: true
# → synthesizes: { field: spec.workloadType, operator: in,
#                  value: "app,cert,monitoring,infra",
#                  message: "Workload Type must be one of: app, cert, monitoring, infra",
#                  action: deny }
```

`serve.fields` entries backed by a spec field don't infer `enum:` from the CRD's OpenAPI schema for this purpose — declare it explicitly to opt a field into this synthesis, the same way `serve labels/annotations` already requires `type`/`enum` since those have no CRD schema to infer from.

### Serve-aware messages for hand-written rules

Auto-synthesis covers plain existence and enum membership. For anything more specific — a range, a prefix, a cross-field comparison — you still write the `validation.rules` entry by hand, and there the same principle applies: write `message:` using the field's `label:` instead of its raw `spec.*` path. A developer submitting through the Control Center form saw the label, not the YAML path — an error that echoes the path back is a translation the developer has to do themselves.

```yaml
serve:
  fields:
    image:
      label: "Container Image"

validation:
  rules:
    # before — leaks the internal field name
    - field: spec.image
      prefix: "myorg/"
      message: "spec.image must start with myorg/"

    # after — matches what the developer actually saw on the form
    - field: spec.image
      prefix: "myorg/"
      message: "Container Image must start with myorg/"
```

This applies whether the rule fires from a form submission, `kubectl apply`, or a CI pipeline — the message is the same either way, so keep it in the vocabulary of the person reading it, not the API shape.

### Linking a rule to its form field (`link:`)

`message:` fixes what the developer *reads*. `link:` fixes what the Control Center (or any Gateway API client) *highlights* — which requires knowing which rendered field a violation concerns, not just a human-readable sentence about it.

For a plain `field: spec.image`, that's free: the violation reports `"spec.image"`, and a client strips the `spec.` prefix to get `"image"`, the same key the field was rendered under. But `serve labels/annotations` entries (labels and annotations) can't be a plain dot-path — they resolve through `getLabel`/`getAnnotation` template expressions instead, because e.g. an annotation key following the Kubernetes `prefix/name` shape contains dots that a dot-path resolver would misparse as extra segments. A hand-written rule on a *spec* field can end up in the same situation, wrapping it in something like `isValidGitRepository` instead of comparing it directly. Either way, the violation reports the raw expression — `{{ getLabel . "team" }}` or `{{ isValidGitRepository .spec.repoURL }}` — which isn't a field name a client can match against anything it rendered.

`link:` closes that gap — it's the plain key, reported instead of `field:` in the violation:

```yaml
serve:
  labels:
    team:
      label: "Team"
      required: true

validation:
  rules:
    - field: '{{ isDNS1123Subdomain (getLabel . "team") }}'
      link: team
      equals: "true"
      message: "team must be a valid DNS subdomain"
      action: deny
```

Without `link: team`, this violation reports `field: '{{ isDNS1123Subdomain (getLabel . "team") }}'` — nothing a form can highlight. With it, the violation reports `field: "team"`, matching the rendered field directly.

`link:` is validated at katalog-load time: it must name a key declared in `serve.fields`, `serve.labels`, or `serve.annotations` on the same CRD — a typo or a stale reference to a renamed/removed field is a load-time error, not a silently-broken highlight discovered later. Pointing it at a spec field whose `field:` is already exactly `spec.<name>` is also an error — at that point `field:` is already a clean key on its own, so the link is redundant.

Required/enum rules synthesized from `serve labels/annotations` set `link:` automatically — you only write it by hand for custom rules.

One thing this unlocks: multiple rules can target the same field. Before `link:`, there was pressure to cram every check for one field into a single expression, because that expression doubled as the only thing identifying which field it concerned. With `link:` decoupling "which check" from "which field," each check can be its own rule with its own message:

```yaml
validation:
  rules:
    - field: '{{ isDNS1123Subdomain (getLabel . "team") }}'
      link: team
      equals: "true"
      message: "team must be a valid DNS subdomain"
      action: deny

    - field: '{{ isReservedTeamName (getLabel . "team") }}'
      link: team
      equals: "false"
      message: "team cannot be a reserved platform namespace prefix"
      action: deny
```

Both rules highlight the same field, each with a message specific to what actually failed.

### Template expressions in rules

`field:`, comparison values (`equals:`, `prefix:`, `min:`, `value:`, …), and `message:` are all resolved as Go templates before evaluation. The full CR fields and notes FuncMap are available.

When `field:` is a template expression, the resolved value is used in the comparison directly — not looked up as a path in the CR. The original expression is preserved in violation messages.

```yaml
notes:
  functions:
    - name: inBusinessHours
      expression: '{{ and weekday (timeInWindow "09:00" "18:00") }}'
    - name: allowedRegistry
      expression: "myorg/"

validation:
  rules:
    - field: "{{ inBusinessHours }}"
      equals: "true"
      action: deny
      message: "deployments are only allowed during business hours"

    - field: spec.image
      prefix: "{{ allowedRegistry }}"
      action: deny
      message: "image must come from {{ allowedRegistry }}"
```

## `validation.external`

External HTTP calls can be declared directly under `validation:`. They fire before any rule is evaluated, and their results are available in `field:` expressions as `.external.<name>.*`.

```yaml
validation:
  external:
    - name: healthCheck
      url: "{{ .spec.healthCheckUrl }}/health"
      expectedStatus: 200
      continueOnError: true
      fires:
        reconcile: false   # admission-only — skip during reconcile resyncs

  rules:
    - field: "{{ .external.healthCheck.status }}"
      equals: "200"
      action: deny
      message: "health check failed — CR rejected"
```

`fires.reconcile: false` means the call only runs at `kubectl apply` time. When omitted (default), the call also runs on every reconcile — the result is available the same way as `onReconcile.external` calls.

See [13-external.md](13-external.md) for the full field reference.

## Operators

`validation.rules` uses the exact same operator set and shorthand fields as `when:`/`anyOf:` — both are backed by the same `Condition`/`ConditionOperator` evaluation code (`pkg/types/validation_eval.go` and `pkg/types/when.go` share one operator table so the two can't drift apart). See [when/anyOf conditions § Operators](06-when-conditions.md#operators) for the full list.

One operator worth calling out: `unique` — field value must be unique across all existing instances of this CRD. It works the same way in `validation.rules` and in `when:`/`anyOf:` (e.g. gating a template source or mutation rule on whether a field is still available), and it's enforced at both reconcile and admission time — just via two different checks with different guarantees:

- **Reconcile time** — the reconciler lists other instances via a live call against the API server. This is the authoritative check: immune to cache staleness, always correct.
- **Admission time** — the gateway asks the runtime's own `/katalog/{crd}/cr?field=` endpoint (served from its informer cache, not a live API call) whether any other instance already has this value. This is a fast, best-effort early rejection, not a second source of truth — if the runtime's cache is a moment stale, a duplicate can still slip through admission, but it's always caught on the very next reconcile regardless. Nothing about the reconcile-time guarantee depends on admission catching it first.

Testable with `ork simulate` by adding a second document of the same kind to the CR file — see the `cr` field in [Simulate schema](../05-simulate/index.md). `ork simulate` only exercises the reconcile-time path (admission webhooks aren't registered in the fake cluster); the admission-time path needs a real cluster — see `ork e2e`.

## `action`

| Value | Effect |
|-------|--------|
| `deny` (default) | Webhook returns a rejection; reconcile fails with an error. |
| `warn` | Webhook allows the operation; a warning is logged. |

## When validation runs

- **At admission**: if `security.webhooks.admission.enabled: true` and the CRD's `webhooks.validation: true`.
- **At reconcile**: always — even without a webhook, rules are checked during each cycle.

---
