# Patterns

A **pattern** is any artifact published to the Orkestra Registry. There are three kinds.

---

## Motif

A Motif is a **reusable resource primitive**. It declares named inputs and contributes resources — Deployments, StatefulSets, Services, PVCs, Secrets — to any CRD that imports it. A Motif has no CRD of its own. It is a composable building block.

```text
01-motifs/web-service/
  motif.yaml      ← required
  README.md       ← optional, shown in ork patterns
```

A team writes a web-service Motif once — Deployment + ClusterIP Service + optional Ingress — and every team that needs a stateless web service imports it. When the probe configuration changes, the Motif version bumps. Katalogs that import it upgrade on their own schedule.

Push to: `ORK_MOTIFS_REGISTRY`

Reference: [motifs](../orkestra-registry/01-motifs.md)

---

## Katalog

A Katalog is a **complete operator declaration**. It binds a CRD to reconcile logic — either inline or by importing Motifs — and adds admission rules, status fields, and gates. A Katalog published to the registry is a full operator someone else can pull and run without writing Go.

```text
02-katalog-api/
  katalog.yaml    ← required
  crd.yaml        ← CRD manifest consumers apply before the operator starts
  cr.yaml         ← sample CR for testing
  simulate.yaml   ← assertions — the behavioral proof
  e2e.yaml        ← cluster verification gate
  README.md
```

`katalog.yaml` is the only required file. Everything else strengthens the artifact's quality signal. An artifact with `simulate.yaml` and `e2e.yaml` carries a proof. One without carries a promise.

Push to: `ORK_REGISTRY`

Reference: [katalogs](../orkestra-registry/02-katalogs.md)

---

## Komposer

A Komposer is the **composition plane**. It imports multiple patterns — from the registry, from local files, or from Helm — and declares how to run them together as one Orkestra runtime. Komposers are what platform teams ship to production.

```text
05-komposer/
  komposer.yaml   ← required
  cr-valid.yaml   ← example CRs exercising the composed operators
  cr-denied.yaml
```

A Komposer published to the registry can be imported by another Komposer with `useKomposer: true`. This lets teams share full platform compositions, not just individual operators.

Push to: `ORK_REGISTRY` (same as katalogs)

Reference: [komposers](../orkestra-registry/03-komposers.md)

---

## How they relate

```text
Motif               ← imported by Katalog at spec.crds[*].imports
  └── Katalog       ← imported by Komposer at imports.registry
        └── Komposer  ← deployed as one Orkestra runtime binary
```

A Motif contributes resources. A Katalog owns a CRD and its behavior. A Komposer composes multiple Katalogs into a platform. Each layer is versioned and published independently.

---

## Pattern directory rules

Every pattern directory pushed to the registry must contain exactly one primary file (`motif.yaml` for Motifs, `katalog.yaml` for Katalogs and Komposers). All other files are optional but contribute to the quality signal visible in `ork inspect`.

Local motif imports (`../01-motifs/web-service/motif.yaml`) are valid for local development — `ork simulate` and `ork template` resolve them. They are not valid for publishing: `ork push` blocks when local imports are found and tells you what to replace them with.

---

## Try it

```bash
ork init --pack registry-guide
cd 01-motifs/web-service
ork validate
```

→ Next: [Consuming Patterns](02-consuming.md)
