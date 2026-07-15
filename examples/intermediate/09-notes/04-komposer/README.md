# 04 — Komposer Note Override

A Komposer composes multiple Katalogs into one runtime and can override any note inline.

## What this teaches

Notes flow through the import chain: Motif → Katalog → Komposer. At each level you can add or replace a note:

- **Motif** ([`../03-motifs/motifs/motif.yaml`](../03-motifs/motifs/motif.yaml)) — declares `fullImage` and `appLabel` (shared across the team)
- **Katalog** ([`../03-motifs/katalog.yaml`](../03-motifs/katalog.yaml)) — declares `serviceHost` inline (cluster-local hostname)
- **Komposer** (`komposer.yaml`) — re-declares `serviceHost` with a production cluster domain

The Komposer's `notes:` block wins. The override is inline — **Komposers cannot use `spec.imports`**; that path is reserved for Katalogs.

## Inspect the override

```bash
ork validate --notes
```

`serviceHost` will show the production domain from this Komposer:

```
Notes (3)
  serviceHost   {{ .metadata.name }}.{{ .metadata.namespace }}.svc.prod-cluster.example.com
  fullImage     {{ .spec.image }}:{{ .spec.tag | default "latest" }}
  appLabel      {{ .metadata.namespace }}-{{ .metadata.name }}
```

## Run

```bash
ork run
```

Verify the override is live — `status.host` uses the production domain, not `cluster.local`:

```bash
kubectl get workload my-service -o jsonpath='{.status.host}' && echo
# my-service.default.svc.prod-cluster.example.com
```

## Open the Control Center

```bash
ork control
```

Open [http://localhost:8081](http://localhost:8081) (username: `orkestra`, password: `orkestra`).

The Resources tab shows `status.host` using the production domain from the Komposer override.

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```
