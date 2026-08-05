# 05 — Auth

## v1: Static bearer tokens

The Apply API uses `Authorization: Bearer <token>` on every request. Each token entry in the Katalog can reference a Kubernetes Secret or provide a direct value:

```yaml
gateway:
  applyAPI:
    auth:
      tokens:
        - name: ci-pipeline
          secretRef:
            name: ork-apply-token
            key: token
            # namespace: orkestra-system  ← defaults to Orkestra's own namespace

        - name: terraform-runner
          secretRef:
            name: ork-tf-token
            key: token

        - name: ci-injected
          token: "${ORK_CI_TOKEN}"   # env var expansion — no literal values accepted
```

`name` is for logging and audit. `secretRef` is the standard pattern — the gateway reads or self-bootstraps the Secret at startup using its ServiceAccount. `token` accepts `${ENV_VAR}` references expanded at startup; set the variable via `extraEnv` in the Helm values for both the gateway and the control center deployments (see `charts/orkestra/values.yaml`). Literal values are not accepted.

The auth middleware matches the `Bearer` value against all configured tokens. No token — 401. No match — 401. Match — the request proceeds with the token name attached to the context for logging.

### secretRef vs token

`secretRef` reads the token from a Kubernetes Secret at startup — the value never appears in the Katalog YAML, is rotatable independently, and is RBAC-scoped to the gateway's ServiceAccount. This is the recommended pattern.

`token: "${ENV_VAR}"` expands an environment variable at startup. Use this when you manage the token externally (e.g. injected from a secrets manager at deploy time). Set the variable in `gateway.extraEnv` and `controlCenter.extraEnv` in your Helm values:

```yaml
gateway:
  extraEnv:
    - name: ORK_CI_TOKEN
      valueFrom:
        secretKeyRef:
          name: my-token-secret
          key: token

controlCenter:
  extraEnv:
    - name: ORK_CI_TOKEN
      valueFrom:
        secretKeyRef:
          name: my-token-secret
          key: token
```

The value must always be an `${ENV_VAR}` reference — Orkestra does not accept literal sensitive values in YAML.

### Self-bootstrap and rotation

When a `secretRef` is configured, the gateway runs the same `once: true` + `rotateAfter` flow that `pkg/runtime/runners` uses for operator-managed secrets — via the same functions (`secretExists`, `secretNeedsRotation`, `deleteSecretForRotation`, `generationAnnotations`).

```yaml
gateway:
  applyAPI:
    auth:
      tokens:
        - name: control-center
          secretRef:
            name: ork-apply-token
            key: token
            rotateAfter: 90d   # optional — omit for no automatic rotation
```

Startup flow:

```
gateway starts
  └─ for each secretRef token:
       ├─ Secret missing → create with uuidv4 + generated-at annotation
       ├─ Secret exists, rotateAfter set → check generated-at annotation
       │    ├─ expired  → delete → recreate with new uuidv4
       │    └─ current  → read token, proceed
       └─ Secret exists, no rotateAfter → read token, proceed
```

The `generated-at` and `rotate-after` annotations written onto the Secret are the same ones `pkg/runtime/runners` uses — the rotation clock works identically to operator-managed credentials.

To pin a specific token: create the Secret before the gateway starts. The gateway finds it, annotates it (starting the rotation clock from now), and proceeds. To force immediate rotation: delete the Secret and restart.

### RBAC — automatic

When any `secretRef` entry is declared under `applyAPI.auth.tokens`, `GenerateGatewayRBACRules()` in `pkg/katalog/generate_rbac.go` automatically adds `get` and `create` on `secrets` to the gateway's generated ClusterRole. `get` is needed to read existing tokens; `create` is needed for self-bootstrap when the Secret does not yet exist.

```go
// Apply API — secretRef token resolution and self-bootstrap
if k.HasApplyAPISecretRefs() {
    rules = append(rules, rbacv1.PolicyRule{
        APIGroups: []string{""},
        Resources: []string{"secrets"},
        Verbs:     []string{"get", "create"},
    })
}
```

If the gateway already has `secrets` access via `NeedsCertificates()` (TLS cert management), the rules are additive — Kubernetes merges them and the gateway gets the union of verbs.

### Operator-provisioned tokens

An operator can create the Apply API Secret as part of reconciliation. A team onboarding operator, for example, creates a namespace and a Secret containing the Apply API token in one reconcile cycle:

```yaml
onReconcile:
  namespaces:
    - name: "team-{{ .spec.teamName }}"
  secrets:
    - name: "{{ .metadata.name }}-apply-token"
      namespace: orkestra-system
      once: true
      data:
        token: "{{ uuidv4 }}"
```

`once: true` — the token is generated on first reconcile and never overwritten. The new team's CI reads the Secret from the cluster and uses the `token` key as the bearer token. Access provisioning is operator-managed — no manual token creation.

### Scoping — `idp.allowedTokens`

Two independent layers, both enforced on every request:

- **`allowedNamespaces`/`restrictedNamespaces` on the CRD entry** — topology, the same for every caller. If a CRD should only ever exist in `team-payments`, set `allowedNamespaces: [team-payments]` — every token, every caller, sees that boundary.
- **`idp.allowedTokens`, per CRD** — identity. Which operations (`get`/`list`/`create`/`update`/`delete`/`*`), on which endpoint class (`resources`/`schema`/`global`), in which namespaces, *for this specific token*. Two tokens against the same CRD can get different answers — a `ci-pipeline` token that can create in `staging` but not touch `production`; a `security-audit` token that's read-only everywhere.

```yaml
idp:
  allowedTokens:
    ci-pipeline:
      namespaces: [team-payments-staging]
      permissions:
        resources: [create, update, get, list]   # no delete
```

Checked in `checkIDPPermission` before the apply/read/list/delete handler runs — a denied request never reaches SSA. Returns `403` with a message naming exactly which check failed (unknown token, wrong namespace, missing operation), not a silent drop or generic `401`. `ork validate` checks token names against `gateway.applyAPI.auth.tokens`, rejects invalid operations and duplicate entries, and rejects a token namespace the CRD itself doesn't allow.

A CRD with no `idp.allowedTokens` block places no restriction here — any gateway-level token can call any endpoint the Apply API exposes for it, bounded only by the CRD-level namespace rules above.

→ [Token Scoping](../../../../documentation/concepts/idp/03-token-scoping.md) — full model, worked multi-token example

→ [IDP token permissions](../../../../documentation/security/08-idp-permissions.md) — the security write-up

## Adding a new auth mode

Auth is handled in `auth.go`. The middleware interface takes a request and returns the caller identity or an error. Adding a new mode means implementing the interface and wiring it in the Apply server constructor.

Two auth modes are documented as follow-on contributions. See [contributing-controlcenter.md](../../../documentation/contributing/contributing-controlcenter.md#idp-mode--follow-on-improvements) for the design intent:

### OIDC

The gateway validates the `Authorization: Bearer` token against an OIDC issuer. The CC can use the user's existing OIDC session as the Apply API credential — no separate token needed. The caller identity becomes the OIDC subject claim.

### Service account token review

For in-cluster callers (CI running inside the cluster), the gateway validates Kubernetes service account tokens via the `TokenReview` API. No static token configuration required for in-cluster use cases.

→ Back: [04-resources.md](04-resources.md)
