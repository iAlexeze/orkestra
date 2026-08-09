# Lifecycle in Orkestra

OLM exists because operators were binaries. When the unit of distribution is a binary, lifecycle management becomes a separate problem — you need a separate system to package, version, upgrade, and deprecate it. OLM is that system for the Kubernetes operator world.

Orkestra removes the problem at the root. The operator is not a binary — it is a pattern. Patterns are data. Data's lifecycle is not an external concern to manage. It is a built-in property of the artifact.

**Lifecycle follows production.** The same model that bakes tests into the artifact at publish time ([simulate](../simulate/) gates + [e2e](../e2e/) gates, proof in OCI annotations) is the model that governs upgrade, deprecation, and deletion. There is no separate lifecycle system to install. You write patterns, and the lifecycle comes with them.

This document covers the lifecycle of one pattern — from first write to deprecation. The whitepaper covers the binary-vs-data argument in depth: [Declarative Operators Whitepaper](/publications/01-declarative-operators-whitepaper).

---

## The full arc

| Stage | How it works |
|-------|-------------|
| **Create** | A Katalog is YAML — no build pipeline, no image, no binary |
| **Validate** | `ork validate` — offline, schema and security posture |
| **Test** | `ork simulate` (sub-second, no cluster) + `ork e2e` (real cluster) |
| **Distribute** | `ork push` — gates run, proof baked into OCI annotations, artifact ships |
| **Inspect** | `ork inspect` — proof is visible before pulling, before importing |
| **Consume** | `ork pull` — artifact arrives with its proof |
| **Upgrade** | Version bump in the Komposer — same push pipeline, same gates |
| **Deprecate** | `metadata.deprecation` — author sees it at `ork push`; consumers at `ork inspect`, `ork pull`, `ork validate`; `ork run`/`ork gate` enforce `accept` |
| **Delete** | `security.deletionProtection` — webhook blocks `kubectl delete` on CRs, CRDs, and Orkestra infrastructure |

---

## Creation

A Katalog declares a CRD and its complete operator behavior. There is no build step.

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: platform
spec:
  crds:
    database:
      apiTypes:
      ...
    app:
      apiTypes:
      ...
      dependsOn:
        database: healthy   # app does not reconcile until database is healthy
```

Complex Katalogs can import Motifs — reusable resource blueprints for security posture, probes, retry policy, and RBAC. Motifs are themselves versioned OCI artifacts.

---

## Validation and testing

`ork validate` runs entirely offline. It checks schema, security posture, and structural correctness. No cluster required.

`ork simulate` runs the real reconciler against an in-memory cluster. It asserts which resources are created in which reconcile cycle. Sub-second, no Docker, runs in CI.

`ork e2e` spins up a kind cluster, deploys the operator, and verifies behavior against real scheduling and webhooks. This is the outer gate before publishing.

Both are declarative — they are YAML files describing expectations, not test scripts. That matters for the lifecycle: the tests ship with the pattern, they are reviewable, and they cannot be silently skipped.

---

## Distribution

```bash
ork push database:v1.0.0 ./database/
```

`ork push` runs both gates automatically. If either fails, the push is blocked. When they pass, the results are baked into OCI annotations attached to the artifact:

| Annotation | What it records |
|-----------|-----------------|
| `io.orkestra.simulate.status` | Passed / no assertions / skipped |
| `io.orkestra.simulate.assertions` | Count of `expect:` blocks |
| `io.orkestra.e2e.status` | Verified against a real cluster |
| `io.orkestra.e2e.assertions` | Total expectations |
| `io.orkestra.katalog.typed` | Whether a custom runtime is required |
| `io.orkestra.deprecated` | Lifecycle status |
| `io.orkestra.deprecated.message` | Deprecation message |
| `io.orkestra.katalog.deprecated.migrated_to` | Migration target |
| `io.orkestra.katalog.deprecated.timeline_from` | Deprecation window open date |
| `io.orkestra.katalog.deprecated.timeline_to` | End-of-life date |

The artifact is self-describing. A consumer reads its proof before pulling.

---

## Consumption

```bash
ork inspect database:v1.0.0
```

`ork inspect` shows the simulation status, e2e status, assertion counts, deprecation status, and the list of files inside the artifact — before pulling, before importing.

```bash
ork pull database:v1.0.0
```

`ork pull` downloads the artifact, caches it, and recursively pulls any referenced Motifs. The artifact arrives with its proof intact.

---

## Upgrade

Patterns are versioned with OCI tags. `database:v1.0.0` and `database:v1.1.0` are distinct artifacts with distinct proofs.

To upgrade, the Komposer references the new version:

```yaml
imports:
  registry:
    - url: database:v1.1.0
```

The same push pipeline runs. The new artifact carries its own simulation and e2e proof. Upgrade is not a ceremony — it is the same production model, applied again. See [Foundations: Configuration is Deliberate](../../foundations/03-no-autosync.md) for why upgrade is always an explicit action, never automatic.

---

## Deprecation

```yaml
metadata:
  name: database
  deprecation:
    message: "Use v2.0.0 — improved connection pooling and status reporting"
    migratedTo: database:v2.0.0
    timeline:
      from: "2026-09-01"   # deprecation window opens — warning with countdown
      to:   "2027-03-01"   # end of life
```

The deprecation is part of the artifact. It is surfaced at every touchpoint — and the author sees it first: `ork push` shows the exact warning consumers will see, immediately after the katalog is validated and before the artifact is uploaded. After that, `ork validate`, `ork inspect`, and `ork pull` all surface the same state-aware warning. The state is computed from today vs the timeline:

| Condition | Shown as |
|-----------|----------|
| No `timeline`, or `today < from` | ⚠ Deprecated |
| `from ≤ today < to` | ⚠ Deprecated · N days until EOL |
| `today ≥ to` | ✗ END OF LIFE |

The pattern remains in the registry for compatibility — it is not deleted — but it is clearly marked at every touchpoint. `ork validate` enforces that `message` is present and that `from` is strictly before `to`.

**Runtime enforcement** — `ork run` and `ork gate` refuse to start a deprecated or EOL Katalog unless the operator has explicitly acknowledged it in the file:

```yaml
deprecation:
  accept:
    beforeEol: true   # allows startup during the deprecation warning window
    eol: true         # additionally required after the end-of-life date
```

This makes the decision to run a deprecated pattern a visible, reviewable record in the Katalog itself — not a flag in a deploy script. `eol: true` without `beforeEol: true` is not accepted; both must be set to run past the EOL date.

The separation is deliberate: `ork validate` is a pre-flight tool and shows warnings without blocking. The enforcement gate lives at runtime startup — after validation passes, before the operator begins reconciling. A PR adding `eol: true` is a traceable, reviewable decision rather than a buried CLI flag.

See the [deprecation schema reference](../../reference/schema/02-katalog/00-metadata/deprecation.md) for the full field list, enforcement table, and OCI annotation mapping.

---

## Deletion protection

Deletion protection attaches a label to every managed CR and CRD, and registers a validating webhook that blocks any `kubectl delete` on labeled resources. When enabled, it protects both your CRs and Orkestra's own infrastructure — the Deployment, Service, webhook configurations, and supporting resources all carry the same label and are blocked by the same webhook.

```yaml
security:
  deletionProtection:
    enabled: true
```

Per-CRD overrides let you opt individual CRDs out of CR-level or CRD-level protection independently. See [Deletion Protection](../../security/04-deletion-protection.md) for the full configuration, strict mode, and gateway-only behavior.

---

## Why this matters

OLM solved the right problem for its era. When operators were binaries, lifecycle management had to be external — and OLM built exactly the system that required: its own controllers, its own CRDs, its own installation lifecycle. The overhead was the necessary cost of the binary constraint.

Orkestra makes the question moot. The operator is not a binary — it is a Katalog. There are no new CRDs, no new controllers, no lifecycle stack to operate. [Your CRD is enough](/blog/your-crd-is-enough). The lifecycle — versioning, testing, distribution, deprecation, deletion — is expressed in the same artifact language as the operator itself.

The proof is in the annotation. Every `ork push` attaches what was verified, to what level, and when. Every `ork inspect` surfaces it. The lifecycle is not a process running somewhere. It is a record traveling with the artifact.
