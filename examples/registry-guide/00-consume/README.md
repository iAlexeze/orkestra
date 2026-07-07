# 00 — Consume a registry pattern

The fastest path from zero to a running Postgres instance. No katalog to write. No CRD to author. No Go code.

The Postgres operator already exists in the Orkestra registry. This step pulls it, deploys it, and hands you a running StatefulSet managed by a reconciler you did not write.

---

## Before you deploy — inspect and pull

**1. Inspect the quality signals and file list**

```bash
ork inspect postgres:v1.0.0
```

You will see:

```text
postgres:v1.0.0
  Kind:        Katalog
  Simulate:    ✓ Verified · 5 assertions · 1.4s · tested 1d ago
  E2E:         ✓ Verified · 5 assertions · 3m41s · tested 1d ago

  Files:
    katalog.yaml                   1.5 KB
    crd.yaml                       1.1 KB
    cr.yaml                        210 B
    simulate.yaml                  604 B
    e2e.yaml                       1.3 KB

To pull:
  ork pull postgres:v1.0.0
```

The author ran simulate and e2e before publishing. The proof is in the artifact. You can see every file that was pushed before pulling anything.

**2. Read the files you care about before pulling**

You can inspect any file directly from the registry — before committing to a pull:

```bash
ork inspect postgres:v1.0.0 --view katalog.yaml,simulate.yaml
```

This prints the raw content of those two files inline. You see exactly what the pattern does and what guarantees the simulate gate provides — without downloading anything else.

A real example output:

```text
── katalog.yaml ──
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: postgres
  version: v1.0.0
imports:
  - motif: oci://ghcr.io/orkspace/orkestra-registry/motifs/postgres:v1.0.0
...

── simulate.yaml ──
cr: cr.yaml
expect:
  ...
  ops:
    - cycle: 1
      verb: create
      resource: services
      name: my-postgres-postgres-headless
    - cycle: 1
      verb: create
      resource: services
      name: my-postgres-postgres
```

Once you are satisfied with what you see, pull.

**3. Pull to local cache**

```bash
ork pull postgres:v1.0.0
```

The pattern is cached at `~/.orkestra/registry/...` and the file list is printed on download. Nothing is deployed yet.

**4. Validate, template and simulate — verify the pattern offline**

```bash
ork validate
ork template
ork simulate
```

`ork template` resolves the OCI import from cache and shows the full merged Kubernetes resources offline — no cluster needed. `ork simulate` replays the simulate gate the author ran before publishing.

---

## Deploy

**1. Apply the CRD**

The CRD comes from the upstream pattern. Fetch it directly from the artifact and apply:

```bash
ork inspect postgres:v1.0.0 --view crd.yaml > crd.yaml
kubectl apply -f crd.yaml
```

**2. Generate and apply the bundle**

```bash
ork generate bundle -o bundle.yaml
kubectl apply -f bundle.yaml
```

**3. Install Orkestra**

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  --wait --timeout 120s
```

**4. Create a Postgres instance**

Inspect the canonical CR from the pattern to see the available fields, then apply:

```bash
ork inspect postgres:v1.0.0 --view cr.yaml       # use > cr.yaml to save to file
kubectl apply -f cr.yaml
```

**5. Watch it reconcile**

```bash
kubectl get postgres my-postgres
```

The Runtime reconciles the CR and provisions: StatefulSet, headless Service, client Service, and a credentials Secret. The password is auto-generated on first reconcile and rotated every 30 days.

---

## What to look for

```bash
kubectl get secret my-postgres-creds -o jsonpath='{.data.password}' | base64 -d && echo
```

The password was generated once at first reconcile. It will not change until `rotateAfter: 30d` elapses — even if you delete and recreate the pod.

Confirm the StatefulSet is wired to the same secret — not a hardcoded value:

```bash
kubectl get statefulset my-postgres-postgres \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="POSTGRES_PASSWORD")].valueFrom.secretKeyRef.name}' && echo

# my-postgres-creds
```

The secret name matches. The pod receives the password as `POSTGRES_PASSWORD` injected from that secret — the reconciler never writes the credential into the StatefulSet spec directly.

```bash
kubectl get statefulset my-postgres-postgres -n default
# NAME          READY   AGE
# my-postgres-postgres   1/1     30s
```

```bash
ork proxy
# open http://localhost:8081 to see the live reconcile state
```

---

## Another way to run it

If you don't want to go through inspect → pull → deploy manually, `ork run` does it in one command:

```bash
ork run postgres:v1.0.0 --dev --apply-cr
```

`--dev` creates a local kind cluster if you don't have one. `--apply-cr` applies the CRD and a sample CR so the operator starts with something to reconcile. No files to download, no kubectl, no Helm.

```bash
# Already have a cluster? Skip --dev
ork run postgres:v1.0.0 --apply-cr
```

`ork run` pulls from the registry on first use and caches locally — subsequent runs start immediately.

---
## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh
```

---

## Next step

→ [01-motifs](../01-motifs/README.md) — write your own reusable Motif
