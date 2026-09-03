# Publishing Katalogs

A Katalog published to the registry is a complete operator. The consumer imports it and gets a full CRD + reconciler — simulate assertions, E2E proof, and all. This page covers the publish flow from local development to a verified artifact.

---

## The katalog directory

```text
02-katalog-api/
  katalog.yaml    ← required
  crd.yaml        ← CRD manifest
  cr.yaml         ← sample CR for testing
  simulate.yaml   ← reconciler assertions (gates push)
  e2e.yaml        ← cluster verification (gates push)
  README.md       ← shown in ork patterns and ork inspect
```

`katalog.yaml` is the only required file. Adding `simulate.yaml` means the artifact carries a proof. Adding `e2e.yaml` means it was verified against a real cluster. Consumers see both before importing.

---

## Local development: use file paths for motif imports

While developing locally, reference Motifs by relative file path:

```yaml
# katalog.yaml — local development
spec:
  crds:
    webapp:
      imports:
        - motif: ../01-motifs/web-service/motif.yaml
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
```

This lets `ork simulate`, `ork template`, and `ork validate` resolve the Motif without needing it in the registry. **Before pushing the Katalog**, replace the path with the OCI ref (see below).

---

## Write and validate

```bash
ork validate -f katalog.yaml
```

Validates the merged Katalog, checks CRD entry types, confirms all Motif inputs declared in `with:` are present in the imported Motif.

Render the expanded Katalog to see what a real reconcile would produce:

```bash
ork template -f katalog.yaml
```

The output shows every resource that would be created by the Deployment, Service, and Ingress — with all template expressions evaluated against the sample CR.

---

## Write simulate assertions

```bash
ork simulate init      # scaffolds simulate.yaml from a live reconcile run
ork simulate           # run and verify
```

Simulate is unit testing for the reconciler — no cluster, no Helm, no Docker. It runs in under a second and catches template errors, wrong resource names, and incorrect cycle ordering before any gate runs.

```text
Simulating webapp/my-webapp

  Cycle 1:
    + deployments/my-webapp
    + services/my-webapp-svc
    ~ status/my-webapp
  Cycle 2:
    ~ status/my-webapp

  ✓ 2/2 assertions passed
  Steady state at cycle 2 in 14ms
```

The simulate assertions are the behavioral contract you publish to consumers. A pattern with `✓ Verified · N assertions` is one you proved works. See [Gate Mechanics](05-gate-mechanics.md) for the full gate story.

---

## Run E2E

```bash
ork e2e
```

Provisions a kind cluster, installs Orkestra, applies the CRD and CR, and verifies all `expect:` conditions. Tears down when done. Takes 2–5 minutes. Use `--cluster` to run against an existing context instead.

---

## Before pushing: replace local Motif imports with OCI refs

```yaml
# katalog.yaml — before publishing
spec:
  crds:
    webapp:
      imports:
        - motif: oci://ghcr.io/myorg/motifs/web-service:v1.0.0
          with:
            image: "{{ .spec.image }}"
            port: "{{ .spec.port }}"
```

`ork push` will block with a clear error if any local file imports remain:

```text
✗ Push blocked: local file imports in katalog.yaml

  spec.crds.webapp.imports[0]: "../01-motifs/web-service/motif.yaml"

  Local imports work for ork simulate and ork template, but cannot
  be resolved by consumers after the katalog is published.

  Before publishing:
    1. Push the motif:  ork push <motif-dir>/
    2. Replace the local path with the OCI ref:
       motif: oci://<your-registry>/motifs/<name>:<version>
```

---

## Push

```bash
export ORK_REGISTRY=ghcr.io/myorg/katalogs

ork push ./
```

With both `simulate.yaml` and `e2e.yaml` present, push runs the full gate sequence:

```text
Pushing webapp-operator:v1.0.0 (Katalog) to ghcr.io/myorg/katalogs...
  ✓ katalog.yaml         valid
  ✓ crd.yaml             valid
  ✓ cr.yaml              (512 B)
  ✓ simulate.yaml        (872 B)
  ✓ e2e.yaml             (1.4 KB)
  ✓ README.md            (3.1 KB)

Running simulate gate (simulate.yaml)...
  ✓ Simulate passed (2 assertions · 14ms)

Running E2E gate (e2e.yaml)...
  ✓ E2E passed (48s)

✓ Pushed: oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
  Digest: sha256:a3f1c8d20e4b7f9c...

To import:
  imports:
    registry:
      - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
```

Push with a version override (useful in CI where the version is set by the pipeline):

```bash
ork push webapp-operator:v1.0.0 ./
```

---

## Verify the artifact

```bash
ork inspect webapp-operator:v1.0.0
```

```text
webapp-operator:v1.0.0
  Kind:        Katalog
  Simulate:    ✓ Verified · 2 assertions · 14ms
  E2E:         ✓ Verified · 48s
  Signed:      ✗ not signed

To import:
  imports:
    registry:
      - oci://ghcr.io/myorg/katalogs/webapp-operator:v1.0.0
```

---

## Sign the artifact

Signing is separate from pushing — it can happen in a different CI job under a
different OIDC identity. Sign with the workflow that should be trusted:

```bash
ork pattern sign webapp-operator:v1.0.0
```

After signing, `ork inspect` shows the subject on the `Signed:` row:

```text
  Signed:      ✓ verified (keyless) · github.com/myorg/webapp/.github/workflows/release.yaml@refs/heads/main
```

To push and sign in one step:

```bash
ork push webapp-operator:v1.0.0 ./ --sign
```

### Declare a signing policy in the katalog

Add a `publish:` block to require signature verification at pull time:

```yaml
publish:
  signing:
    verify: true
    expectedIdentities:
      - github.com/myorg/webapp/.github/workflows/release.yaml@refs/heads/main
```

With `verify: true`, consumers who run `ork pull --verify` will be blocked if the
artifact is unsigned or signed by an unexpected identity.

### Test signing locally

No CI credentials needed. Use `--sign-local` to push to ttl.sh and test the
full sign → verify loop before wiring it into CI:

```bash
ork push webapp-operator:v1.0.0 ./ --sign-local --ttl 24h
# or
ork pattern sign webapp-operator:v1.0.0 --local --dir ./ --ttl 24h
```

The output prints the exact verify and inspect commands with the ttl.sh ref.

→ See [Artifact Signing](../../security/10-artifact-signing.md) for the full story.

---

## Try it

```bash
ork init --pack registry-guide
cd 02-katalog-api

# Local development
ork template
ork simulate
ork e2e

# Update motif import to OCI ref, then push
export ORK_REGISTRY=ghcr.io/myorg/katalogs
ork push ./
ork inspect webapp-operator:v1.0.0
```

→ Next: [Gate Mechanics](05-gate-mechanics.md)
