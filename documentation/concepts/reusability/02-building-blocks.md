# Building Blocks — Motifs, Katalogs, Komposers

Orkestra's composition model is layered. Each layer is a reusable unit that can be authored independently, versioned, distributed, and composed into a larger whole.

```text
Motif          — a named, reusable fragment of a Katalog
    ↓ imported by
Katalog        — a complete operator declaration
    ↓ imported by
Komposer       — aggregates multiple Katalogs into one runtime
```

---

## Motifs

A Motif is a Katalog fragment packaged for reuse. It can contain anything a Katalog section can: validation rules, mutation rules, hook declarations, resource templates, notes, profiles, external calls.

An operator team publishes a Motif for tenant isolation and RBAC. Every team that provisions namespaces imports it with their own parameters. When the policy changes, the Motif is updated once — no changes in any consumer Katalog.

```yaml
# katalog.yaml
spec:
  crds:
    namespace-provisioner:
      imports:
        - motif: ../motifs/tenant-isolation/motif.yaml
          with:
            namespace: "{{ .spec.targetNamespace }}"
            team: "{{ .spec.team }}"

        - motif: ../motifs/tenant-rbac/motif.yaml
          with:
            team: "{{ .spec.team }}"
            targetNamespace: "{{ .spec.targetNamespace }}"
            owner: "{{ .spec.owner }}"
```

`imports:` is declared per-CRD. Each import names a Motif (local path or OCI reference) and passes `with:` values — static strings or template expressions evaluated per-CR. The consumer declares which version to pull. The Motif author publishes updates independently.

---

## Include — sharing within a unit

`include:` reads a file in the same directory tree and merges it at load time. By the time the runtime starts, every `include:` has been resolved — the runtime sees only the merged result.

Include is available almost everywhere a Katalog can declare structure:

| Location | What include merges |
|----------|---------------------|
| Validation rules | A shared `validation-rules.yaml` across multiple CRDs |
| Mutation rules | Common defaulting logic extracted to a file |
| External call configs | Reusable HTTP call declarations |
| Notes | A function library shared by multiple CRDs in the same Katalog |
| Profiles | Profile sets declared once, included where needed |
| Serve target entries | Token and config declarations for a named surface |
| Conversion webhooks | Shared conversion logic |

A Katalog with dozens of CRDs does not repeat common declarations. Validation rules that apply to every CRD live in one file. Profile sets are declared once. External call patterns are shared.

---

## E2E and Simulate — test composition

The same composition model extends to tests. A Simulate file can import other Simulate files; an E2E suite can import other E2E suites. A platform team can aggregate test coverage across multiple operator packages without duplicating test declarations.

```yaml
# platform-e2e.yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: platform-suite

imports:
  - ./operators/database/e2e.yaml
  - ./operators/cache/e2e.yaml
  - ./operators/network-policy/e2e.yaml
```

Each entry is a bare file path. The aggregated suite runs all imported suites in sequence. Assertions within each imported file remain scoped to their CRDs — the aggregator does not merge or flatten them.

---

## Komposer

A Komposer is the top-level aggregator. It imports multiple Katalogs from local files and merges them into a single runtime.

```yaml
# komposer.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-operators

imports:
  registry:
    - oci://ghcr.io/myorg/patterns/deployment-stack:v1.0.0
  files:
    - ./database-operator/katalog.yaml
    - ./cache-operator/katalog.yaml
    - ./network-policy/katalog.yaml
```

The operator teams maintain and version their Katalogs independently. The platform team composes them in the Komposer.

### Overriding a public pattern

A Katalog declares the schema it was written for — the CRD's API types, the field paths its hooks read. When you import a public Katalog pattern from a registry, those API types may not match your internal CRD.

The Komposer lets you replace the `apiTypes` block — the schema mapping — without touching any of the pattern's logic. Hook behaviour, validation rules, profiles, and gateway declarations are inherited unchanged. Only the API shape is replaced with your own.

This is how a community-published operator pattern becomes an internal operator: import the pattern, declare your CRD's schema in the override, keep everything else.

```yaml
# komposer.yaml
apiVersion: orkestra.orkspace.io/v1
kind: Komposer
metadata:
  name: platform-operators

imports:
  registry:
    - oci://ghcr.io/postgres/patterns/postgres:v1.0.0

# Replace the upstream apiTypes with internal ones
spec:
  crds:
    postgres:
      apiTypes:
        group: myorg.io
        version: v1
        kind: MyOrgDatabase
```

---

## Related topics

- [Orkestra Registry](../../orkestra-registry/index.md) — community Motifs, Katalogs, and Komposers available to import
- [Composition](../composition/index.md) — imports and include explained in depth
- [Schema reference](../../reference/schema/index.md) — Motif, Katalog, Komposer, E2E, and Simulate schemas
