# Live Delivery

`ork serve play` runs the gateway chain offline — no cluster, no gateway, no network. `ork serve apply` is the live flip: it sends the same intent file to a real gateway and gets back a real response.

The two commands share the same input format and the same detection logic. The difference is where the chain runs — in-process from your terminal, or in the gateway on the cluster.

---

## The apply call

```bash
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"
```

The intent file is a flat YAML or JSON document — the same shape play accepts:

```yaml
target: apifixture
name: my-payment-service
team: platform
environment: staging
workloadType: app
repoURL: https://github.com/myorg/payments
```

The gateway handles everything else: which CRD, which namespace, what the CR looks like, what annotations to stamp, what the response payload carries back.

The caller needs one credential — the bearer token. No kubeconfig. No cluster role. No `kubectl`.

---

## What the gateway does on apply

The same pipeline `ork serve play` simulates locally:

1. Target resolution — identifies the CRD and alias from the `target` field
2. Token check — validates the bearer token's permissions for this CRD and operation
3. CR construction — builds the full CR from the flat fields
4. Provenance stamping — writes `serve-target`, `serve-alias`, `serve-source` onto the CR
5. Admission — the admission webhook fires if enabled; validation and mutation rules run
6. CR delivery — the CR is applied to the cluster via server-side apply

The response on success:

```text
  serve apply  intent.yaml

  ✓ PlatformResource  team-payments/my-payment-service
  poll: https://gateway.myorg.io/api/v1/resources/platformresource/team-payments/my-payment-service

accepted
```

The `poll` URL is where to GET the CR once the runtime has reconciled it and written status back.

---

## Full CR mode

If the file has `apiVersion` and `kind` instead of `target`, the gateway detects full CR mode and applies it directly — no field routing, no namespace template. Admission still runs.

```bash
ork serve apply -f cr.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"
```

Both modes are valid. Use target mode for the normal developer workflow; full CR mode for CI pipelines that assemble objects programmatically or for applying a CR an operator elsewhere built.

---

## Dry run

`--dry-run` runs the full admission pipeline at the gateway — token check, field validation, webhook rules — without writing the CR to etcd.

```bash
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --dry-run
```

Use it to validate an intent against a live gateway before committing, or to check that a token has the right permissions for an operation.

---

## The GitOps pattern

```text
intent.yaml committed to Git
    ↓
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"
    (live — POST /api/v1/apply)
    ↓
Gateway validates, stamps provenance, applies CR
    ↓
Runtime reconciles
```

The gateway validates on every apply — token permissions, field constraints, admission rules. A rejected apply exits non-zero and the pipeline stops. There is no separate offline validation step because the consumer does not have the Katalog — only the gateway does.

If you want to verify an intent before committing it, use `--dry-run`:

```bash
ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --dry-run
```

The gateway runs the full admission pipeline and returns whether the intent would be accepted — without writing anything to etcd.

Rollback is `git revert` + re-run `ork serve apply` on the reverted intent file.

---

## In CI

```yaml
- name: Apply intent
  run: |
    ork serve apply -f ./orkestra/intent.yaml \
      --api https://gateway.staging.myorg.io \
      --token ${{ secrets.ORK_TOKEN }}
```

With `orkestra-action`, when `serve` is added as an input:

```yaml
- uses: orkspace/orkestra-action@v1
  with:
    serve: ./orkestra/intent.yaml
    serve-api: https://gateway.staging.myorg.io
    serve-token: ${{ secrets.ORK_TOKEN }}
```

---

- [Local Intent Testing](06-local-intent-testing.md) — the offline half: play, the intent file format, the chain stages

- [CLI reference — ork serve apply](../../reference/cli/13-serve.md#ork-serve-apply)

- [The Gateway as a Delivery Layer](05-gateway-as-delivery-layer.md)
