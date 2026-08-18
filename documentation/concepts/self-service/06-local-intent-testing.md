# Local Intent Testing

The gateway is an intent runner. The runtime is a CR runner.

The gateway collects intentions — flat fields from a caller — and translates them into a valid Kubernetes object. The runtime takes that object and reconciles it into cluster resources. Neither needs a cluster to do its job locally. `ork serve play` runs the gateway's half; `ork simulate` runs the runtime's half.

---

The gateway apply chain is six stages: target resolution, token check, CR construction, provenance stamping, admission validation and mutation, response payload evaluation. In production, all six run in the gateway process when a POST lands at `/api/v1/apply`. Before any of that is running — before a cluster is provisioned, before a delivery surface is wired up, before a GitOps webhook is configured — you can run the same chain locally against your Katalog.

`ork serve play` is that chain, in-process, from a file.

---

## What an intent file is

An intent file is a flat YAML or JSON document with the same shape as a target-mode POST body. It contains a `target` and whatever fields that target expects:

```yaml
target: apifixture
name: my-payment-service
workloadType: app
team: platform
environment: staging
repoURL: https://github.com/myorg/payments
productionApproval: JIRA-1234
```

That is the same payload a CI pipeline would send, or a Control Center form would assemble, or a GitOps webhook would construct from a merged config file. `ork serve play` reads it and walks it through the gateway chain from the inside.

---

## What the chain does locally

```bash
ork serve play --token control-center
```

Five stages, printed as they run:

**1. Target resolution** — the target name is looked up in the Katalog. Aliases resolve the same way they do in the gateway: `preview`, `internal`, or the primary target name all work. The CRD and any alias are surfaced.

**2. Token check** — the named token is checked against the resolved CRD and alias. The same `TokenAllowedFor` logic the gateway uses: alias-specific token map first, CRD-level fallback, then the operation check. A `preview` alias that restricts `control-center` to `get`/`list` will deny `--operation create` here, exactly as it does in the gateway.

**3. CR construction** — `serve.fields` are routed into `spec`, `metadata.labels`, or `metadata.annotations` according to the Katalog declaration. `serve.name` and `serve.namespace` expressions are resolved against the submitted fields. The built CR is printed.

**4. Provenance stamping** — `orkestra.orkspace.io/serve-target` and `serve-alias` are applied to the built CR, the same annotations the gateway stamps before SSA.

**5. Response payload** — `serve.config.response.payload` expressions are evaluated against the built CR. The result is printed — what a caller would receive in the `payload` block of the apply response.

If any stage fails, the trace stops there. A token check failure tells you which permission was missing and why. A CR construction failure tells you which expression could not be resolved. There is no ambiguity about where the intent broke.

---

## The intent-play loop

The pattern is: write an intent file, play it, adjust, repeat.

```bash
# Start with the target schema so you know what fields to provide
ork serve schema --target apifixture

# Write your intent as JSON
cat > intent.json <<EOF
{
  "target": "apifixture",
  "name": "payments-service",
  "workloadType": "app",
  "team": "platform",
  "environment": "staging",
  "repoURL": "https://github.com/myorg/payments",
  "productionApproval": "JIRA-1234"
}
EOF

# Play it
ork serve play --token control-center

# Check what a read-only surface sees
ork serve play --token control-center --target preview

# Check what a CI token can do
ork serve play --token ci-pipeline
```

No cluster. No gateway running. No apply happened. The intent file stays as the source of truth — commit it alongside the Katalog if you want, or keep it local as a test fixture.

---

## Extending play into simulate

Play stops at stage 5 — it builds and stamps the CR but does not reconcile it. `--simulate` hands the built CR directly to `ork simulate` so you can see the child resources too.

```bash
# Op-print mode: see what the reconciler produces from this intent
ork serve play --token control-center --simulate

# Assert mode: declare what it must produce
ork serve play --token control-center --simulate simulate.yaml
```

`--simulate simulate.yaml` points to a `simulate.yaml` spec. The katalog, cycles, `skipExternal`, and `expect:` block all come from the spec — only the CR is substituted (play's built CR replaces `spec.cr`). A failing `expect:` block fails the command with the same assert output as `ork simulate`.

This covers the full local delivery loop:

```text
intent file  →  ork serve play  →  CR  →  ork simulate  →  child resources
```

Both halves run locally, with no cluster required at either end. `--simulate` is only valid for `create` and `update` — read operations do not produce a CR.

---

## Why this matters for delivery

When a GitOps webhook fires, or a CI pipeline POSTs to `/api/v1/apply`, what arrives at the gateway is structurally identical to what `ork serve play` reads. The difference is that the gateway then does SSA against a cluster. Everything before that — the five chain stages — is the same.

Playing the intent locally lets you validate that chain before the delivery surface exists. A field that fails `serve.name` resolution fails locally with the same error it would fail in the gateway. A token that is denied on the `internal` alias is denied locally by the same token check.

This means the intent file is not just a convenience for testing. It is a preview of exactly what will happen when the real delivery path runs. The two should produce identical chain output — the only difference is that play stops at stage 6 and the gateway continues to SSA.

---

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--intent` | `-i` | `intent.yaml` or `intent.json` | Intent file to play |
| `--token` | `-t` | *(required)* | Token name to check |
| `--target` | | | Override the `target` field in the file |
| `--operation` | `-o` | `create` | Operation to simulate |
| `--namespace` | `-N` | | Namespace (for `get`/`list`/`delete`) |
| `--name` | `-n` | | Resource name (for `get`/`delete`) |
| `--simulate` | | | Hand the built CR to `ork simulate`; pass a path to use a simulate spec in assert mode |

When `--intent` is omitted, `ork serve play` looks for `intent.yaml` first, then `intent.json`, in the current directory.

---

## Where to go next

- [CLI reference — ork serve play](../../reference/cli/13-serve.md#ork-serve-play)
- [Gateway as a Delivery Layer](05-gateway-as-delivery-layer.md) — what the chain does in production
- [Target Mode](02-target-mode.md) — the flat intent format and how it maps to a CR
- [Token Scoping](03-token-scoping.md) — what the token check enforces
- [Aliases and Intent Provenance](04-aliases-and-provenance.md) — how aliases and provenance annotations affect the chain
