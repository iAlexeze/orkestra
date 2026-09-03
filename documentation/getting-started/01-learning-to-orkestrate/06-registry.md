# Registry Guide Pack

The distribution pack. Covers the full lifecycle of publishing and consuming Orkestra patterns — from pulling a proven pattern off the public registry to building your own, gating every publish behind simulate and e2e, composing a production platform from registry imports, and automating the entire pipeline with GitHub Actions.

This pack is different from the others: examples run with Helm, not `ork run`. Start by consuming what already exists.

```bash
ork init --pack registry-guide
cd registry-guide/00-consume
```

---

## Before you build anything

| Example | What it teaches |
|---|---|
| `00-consume` | Pull a proven pattern from the public Orkestra registry. Inspect its simulate proof before importing. Re-run the author's assertions in your cluster. Deploy with Helm. The model: behavior you can trust because the proof traveled with it. |

---

## Building and publishing patterns

| Example | What it teaches |
|---|---|
| `01-motifs` | Author three reusable motifs — a web-service, a data-store, and a platform admission policy. Validate and push each to your OCI registry. Version and tag must agree — a mismatch is a hard error. |
| `02-katalog-api` | Build a Katalog from the web-service motif. Run `ork template` to see the full expansion before testing. Gate publish behind `ork simulate` — assertions are the behavioral contract consumers read before importing. |
| `03-katalog-cache` | Simulate gate demo: the simulate gate catches a broken assertion before the pattern ships. Shows `--force` override and what `no-assertion` looks like in `ork inspect`. |
| `04-katalog-platform` | E2E gate demo: simulate passes, e2e fails. An admission rule rejects the test CR — the pattern is blocked from publishing. Shows that simulate and e2e test different things. |

---

## Production deployment and security

| Example | What it teaches |
|---|---|
| `05-komposer` | Compose registry patterns into a production deployment. Supply chain verification: inspect proof and re-run assertions before importing. Admission webhooks, deletion protection, and the clean uninstall sequence. Requires gateway. |
| `06-pattern-zoo` | Seven official Orkestra patterns, one Komposer, twenty lines. Cross-registry composition: official patterns alongside your own. Every imported pattern carries its own proof — the Komposer does not re-test them. |

---

## Lifecycle

| Example | What it teaches |
|---|---|
| `07-upgrade` | Release a new motif version. One katalog upgrades; another stays pinned. Upgrade is deliberate — restart the runtime to pick up the new katalog. `ork plan` previews the reconciler template diff before applying. Rollback is a version bump. |
| `08-bad-actor` | Audit trail inspection. Detect unexpected pushes by comparing digests and diffing against known-good source. Steps to investigate a tampered or unauthorised pattern version and deprecate it. |
| `09-deprecation` | Pattern end-of-life. Declare `deprecation:` in metadata — `ork inspect`, `ork patterns`, and `ork validate` all warn consumers. Old versions are never deleted; consumers migrate on their own timeline. |

---

## Typed operators

| Example | What it teaches |
|---|---|
| `10-hooks-katalog` | Typed Go hooks published as a registry pattern. `ork inspect` shows `Typed: ✓ Has hooks`. Build the runtime image with `make release` before pushing the katalog. |
| `11-typed-komposer` | Mix typed and declarative katalogs in one Komposer. Set the custom runtime image in Helm. CRD startup ordering with `dependsOn`. |

---

## CI automation

| Example | What it teaches |
|---|---|
| `12-ork-action` | The full pipeline in GitHub Actions using `orkspace/orkestra-action`. Declarative and typed operator workflows. `registry-push` validates, simulates, and publishes in one step — no separate gate jobs needed. Version mismatch between git tag and metadata version is a hard CI failure. |
