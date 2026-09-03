# Orkestra — Public Deployment

Three Orkestra runtimes running in the same cluster, each in its own namespace, each managing a different operator. The Orkestra Control Center aggregates all three — live at **[cc.orkestra.sh](https://cc.orkestra.sh)**.

This is what "aggregated runtimes" looks like in practice: three independent operators, one view.

---

## What's running

| Namespace | Operator | CRDs | What it demonstrates |
|-----------|----------|------|----------------------|
| `orkestra-system-01` | hello-website | 1 | Deployment + Service from a `Website` CR |
| `orkestra-system-02` | website-with-serviceaccount | 1 | Same, plus a dedicated ServiceAccount per instance |
| `orkestra-system-03` | secret-distribution | 1 | Copies a Secret across namespaces from a cluster-scoped CR |
| `orkestra-system-04` | app-platform | 5 | ReplicaSets (no Deployment wrapper), auto-generated API keys with 90-day rotation, ConfigMap distribution |
| `orkestra-system-05` | data-platform | 10 | Full data lifecycle: ingestion, processing, delivery, scheduling, observability — 30d/60d credential rotation |
| `orkestra-system-06` | network-suite | 7 | Traffic routing, ReplicaSet backends, self-signed TLS generation, 180d rate limit key rotation |

The Control Center runs in `orkestra-system-01` and is configured to watch all six runtimes.

---

## Deploy

```bash
# Add the Helm repo if you haven't already
helm repo add orkestra https://orkspace.github.io/orkestra
helm repo update

# Deploy all three
make all

# Or one at a time
make cluster-01
make cluster-02
make cluster-03
```

---

## How it works

Each cluster directory contains:

- `katalog.yaml` — the operator declaration (no `crdFile` / `crFiles` / `setup` — those are dev-path fields; CRDs and CRs are applied separately here)
- `crd.yaml` — the CustomResourceDefinition
- `cr.yaml` — a sample custom resource to reconcile
- `setup.yaml` — pre-requisite resources (cluster-03 only)

The deploy script for each instance:
1. Creates the namespace
2. Applies the CRD
3. Generates the RBAC bundle from the Katalog (`ork generate bundle`) and applies it
4. Installs Orkestra via Helm
5. Applies the CR — reconciliation starts immediately

---

## Namespacing

`cluster-01` and `cluster-02` both manage `Website` CRDs but are partitioned by `allowedNamespaces` so they never step on each other:

- `cluster-01` watches `demo-01` only
- `cluster-02` watches `demo-02` only

`cluster-03` manages `SecretDistribution`, which is cluster-scoped.

`cluster-04`, `cluster-05`, and `cluster-06` each own distinct CRD groups (`apps.demo.orkestra.io`, `data.demo.orkestra.io`, `net.demo.orkestra.io`) and watch their own demo namespace, so `allowedNamespaces` prevents any cross-runtime interference.

---

## Teardown

```bash
make clean
```
