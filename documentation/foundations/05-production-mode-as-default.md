# Production Mode as Default

Most systems distinguish between a local development mode and a production mode. Development mode relaxes things — faster feedback, looser validation, skipped steps. Production mode adds them back. The gap between the two is where most surprises live.

Orkestra does not have two modes. Every run is a production run.

---

## `ork run` is a production-grade deployment

`ork run` is not a convenience wrapper for local development. When you run it locally, you are running the full developer CLI — every command available. In production, the runtime binary has all other commands stripped out at compile time. But `ork run` itself is identical: the same reconciler, the same validation pipeline, the same behavior. What production removes is the surface you do not need there. What it runs is exactly what you ran locally.

`ork run --dev` creates a real kind cluster — real Kubernetes API server, real etcd, real admission webhooks. The `--dev` flag provisions the infrastructure you do not already have. It does not relax anything about how Orkestra behaves inside it.

---

## The producer pipeline has no shortcuts

Every Orkestra pattern goes through the same pipeline before it can be distributed:

```text
ork validate   →   ork simulate   →   ork e2e   →   ork push
    ↓                  ↓                ↓                  ↓
schema valid?     assertions pass?   cluster test?    gates passed?
                  no cluster        real cluster      → artifact ships
```

`ork simulate` runs the reconciler loop against an in-memory state machine. No cluster. No infrastructure. Sub-second. It asserts that what the Katalog claims to create is what the reconciler actually creates.

`ork e2e` runs the same assertions against a real Kubernetes cluster. A CR is applied. Resources are checked. Conditions are verified. The operator behaves exactly as it would in production — because it is production.

`ork push` runs both gates automatically before publishing. Simulate fails → push is blocked. E2E fails → push is blocked. The gates can be overridden — `--no-simulate`, `--no-e2e`, `--force` — but the override is visible. A skipped gate is recorded in the OCI annotations. Every consumer who inspects the artifact will see what was not run. The proof travels with the artifact because the gates are the default, and bypassing them leaves a permanent mark.

---

## The proof travels with the artifact

When a pattern is pushed to the registry, Orkestra writes the gate results as OCI annotations on the artifact:

```yaml
io.orkestra.simulate.status:  passed
io.orkestra.simulate.count:   4
io.orkestra.e2e.status:       passed
```

These annotations are immutable — they record what the author proved before they shipped. A consumer who pulls the pattern can read them with `ork inspect` before writing a single line of YAML:

```bash
ork inspect ghcr.io/myorg/katalogs/postgres@v1.0.0
# Simulate:  ✓ Passed — 4 assertions
# E2E:       ✓ Passed
```

A pattern showing `simulate: skipped` or `e2e: failed` signals to every consumer that the author bypassed a gate. This is not hidden. It is surfaced in every inspection command.

Consumers can go further. They can pull the pattern and re-run the author's assertions in their own cluster:

```bash
ork pull ghcr.io/myorg/katalogs/postgres@v1.0.0
ork simulate -f ~/.orkestra/cache/postgres-v1.0.0/simulate.yaml
ork e2e -f ~/.orkestra/cache/postgres-v1.0.0/e2e.yaml
```

The same simulate.yaml and e2e.yaml the author ran are embedded in the artifact. The consumer is not trusting the annotation — they are reproducing the proof. This is the supply chain verification model: behavior you can confirm yourself, before you let it manage your cluster.

---

## Supply chain security

A pattern in the Orkestra registry is not just a YAML file. It is a versioned, immutable artifact with embedded behavioral proof. Each of these properties has a security implication:

**Versioned** — consumers pin to an exact version. A new push to `latest` does not automatically flow to anyone. Breaking changes cannot silently appear in a production operator.

**Embedded proof** — the simulate and e2e assertions are in the artifact. Not in a separate wiki. Not in a README that may have been edited after the push. In the artifact, written at push time by the tool that verified them.

**Reproducible** — any consumer can re-run the proof in their environment. The annotations describe what passed; the embedded files let you confirm it.

**Visible signal** — `ork patterns` shows E2E status for every pattern. `ork inspect` shows simulate assertions. A pattern that skipped its gates is visible at a glance, before anyone imports it.

Together these properties mean that importing a pattern from the Orkestra registry is not an act of trust. It is an act of verification. You inspect the proof, you re-run it if you choose, and you know exactly what you are deploying before you deploy it.

---

## No divergence between environments

The only thing that differs between a local `ork run` and a production Helm deployment is configuration — the Katalog you give it, the environment variables set, the resources available. The runtime behavior is identical.

This means:
- A bug caught by `ork simulate` locally is a bug that would have appeared in production.
- A pattern that passes `ork e2e` against a local kind cluster behaves the same way in a production EKS cluster.
- The RBAC bundle generated by `ork generate bundle` is the same bundle in every environment.
- The validation rules enforced at admission time in staging are the same rules in production.

There is no "we'll harden it before it goes to production." Production is always already the standard. The only question is which cluster you are running it against.

---

## Every decision is a production decision

This principle extends beyond the runtime to how Orkestra is designed.

The production binary excludes developer commands at compile time — not because production has stricter flags, but because the production binary was never built with those commands. [Secure by Design](04-secure-by-design.md) follows from production-as-default: if every run is a production run, then every binary is a production binary.

`ork validate --full` shows you permissions before any cluster interaction — not because you might deploy to production soon, but because the review step belongs before deployment, always, not just before the final deployment.

The bundle you generate and review is the same bundle in every environment. The deliberate restart that upgrades the runtime is the same operation locally and in production. [Configuration is deliberate](03-no-autosync.md) follows from the same premise: if every run is a production run, then every configuration change is a production change.

---

## What this means for teams

A pattern that passes `ork simulate` and `ork e2e` locally is a pattern that is ready for production. Not "ready to be hardened." Ready.

A PR that includes a failing `ork simulate` is a PR that contains a broken operator. Not "broken in a way we'll fix before production." Broken.

An artifact in the registry with `e2e: passed` is an artifact that a stranger can pull, re-run the proof, and deploy with confidence — because production was the standard the author worked to when they shipped it.

---

## Where this appears across the documentation

- **[Registry: Publish with gates](../orkestra-registry/02-katalogs.md)** — `ork push` gate behavior, `--no-simulate`, `--no-e2e` flags and their annotation consequences
- **[Simulate](../concepts/simulate/)** — the in-memory reconcile loop and assertion model
- **[E2E](../concepts/e2e/)** — the declarative cluster test format
- **[Supply chain verification](../orkestra-registry/02-katalogs.md#supply-chain-verification)** — re-running author proofs as a consumer
- **[Configuration is Deliberate](03-no-autosync.md)** — why upgrades require an explicit restart
- **[Secure by Design](04-secure-by-design.md)** — why the production binary is narrower than the developer CLI
