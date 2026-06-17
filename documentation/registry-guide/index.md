# Registry Guide

The Orkestra Registry distributes **behavior** — not binaries. A Katalog published at `webapp-operator:v1.0.0` contains the full operator declaration: CRD schema, reconcile templates, status fields, admission rules, simulate assertions, and an E2E proof that it worked before it shipped. Any Orkestra runtime that pulls it gets all of that — versioned, immutable, and ready to run.

This is what changes about operator distribution. You don't ship a controller image and a Helm chart and a README that says "it should work." You ship a artifact that carries its own proof.

---

## The distribution model

Traditional operator distribution looks like this:

```
author writes Go → builds image → pushes to registry → consumer deploys image → writes CRDs manually
```

Orkestra's distribution model:

```
author writes YAML → simulate gate (no cluster) → e2e gate (real cluster) → push to OCI registry
consumer inspects quality signal → pulls artifact → imports into Komposer → deploy
```

The quality signals are annotations baked into the OCI artifact at push time. `ork inspect` reads them before any file is downloaded. A consumer knows whether simulate passed, whether E2E was verified, and when — before importing.

---

## Contents

| Page | What it covers |
|------|----------------|
| [Patterns](01-patterns.md) | What motifs, katalogs, and komposers are — and how they relate |
| [Consuming Patterns](02-consuming.md) | Browse, inspect, pull, and import patterns from the registry |
| [Publishing Motifs](03-publishing-motifs.md) | Write a reusable resource primitive and push it |
| [Publishing Katalogs](04-publishing-katalogs.md) | Write a full operator, run the gates, publish |
| [Gate Mechanics](05-gate-mechanics.md) | How simulate → e2e → push works, --force, draft artifacts, quality signals |
| [Composing Patterns](06-composing.md) | Komposer, ork generate bundle, rollout, supply chain |
| [Upgrading Patterns](07-upgrade.md) | Motif upgrade, katalog follows, version isolation, rollback |
| [API Evolution](08-api-evolution.md) | CRD field restructuring — with webhooks and without |
| [Policy](09-policy.md) | Platform admission, allowedRegistries, supply chain control |
| [Deprecation](10-deprecation.md) | Pattern lifecycle: deprecate, warn consumers, retire |

---

## Try it

The full registry workflow is available as a runnable example pack:

```bash
ork init --pack registry-guide
```

Each section in this guide references the relevant sub-example. Run them in order for the full arc, or jump to any sub-directory for the specific topic.

---

!!! tip Environment variables
    | Variable | Default | Purpose |
    |----------|---------|---------|
    | `ORK_REGISTRY` | `ghcr.io/orkspace/orkestra-registry/patterns/katalogs` | Where `ork push` and `ork pull` send/read katalogs |
    | `ORK_MOTIFS_REGISTRY` | `ghcr.io/orkspace/orkestra-registry/patterns/motifs` | Where `ork push` and `ork pull` send/read motifs |

    Set these before any `ork push` or `ork pull` command. `docker login <registry>` must have been run first — Orkestra reads `~/.docker/config.json` for credentials.

---

## Where to go next

- [Patterns](01-patterns.md) — what motifs, katalogs, and komposers are, and the directory layout rules
- [Consuming Patterns](02-consuming.md) — browse, inspect, pull, and import patterns from the registry
- [Publishing Motifs](03-publishing-motifs.md) — write a reusable resource primitive and push it
- [Publishing Katalogs](04-publishing-katalogs.md) — write a full operator, run the gates, publish
- [Gate Mechanics](05-gate-mechanics.md) — how simulate → e2e → push works, quality annotations, CI usage
- [Composing Patterns](06-composing.md) — Komposer, ork generate bundle, rollout, supply chain
- [Upgrading Patterns](07-upgrade.md) — motif upgrade, katalog follows, version isolation, rollback
- [API Evolution](08-api-evolution.md) — CRD field restructuring with and without webhooks
- [Policy](09-policy.md) — platform admission, allowedRegistries, supply chain control
- [Deprecation](10-deprecation.md) — pattern lifecycle: deprecate, warn consumers, retire
