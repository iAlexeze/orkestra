# Per-Target Args — BlockchainAppWithTargets

The same hook binary can behave differently on different surfaces. The platform
team declares two targets — `v2-enabled` and `v2-disabled` — each with its own
`operatorBox` and its own `args`. The hook reads `kube.Args()` and never knows
which surface it came from.

```yaml
serve:
  target:
    v2-enabled:
      primary: true
      operatorBox:
        preReconcile:
          enqueueGate:           # blocks outside business hours on this surface
            when:
              - field: '{{ inBusinessHours }}'
                equals: "true"
        reconciler:
          hooks:
            args:
              featureEnabled: "true"    # forced — no HTTP call needed
              inBusinessHours: '{{ inBusinessHours }}'

    v2-disabled:
      operatorBox:
        reconciler:
          hooks:
            args:
              featureEnabled: "false"   # forced off — no gate
              inBusinessHours: '{{ inBusinessHours }}'
```

The hook code is identical to `01-hooks`. The Katalog determines what each
surface means; the caller just picks a target:

**Requirement:** `ork` CLI — install from [orkestra-install](https://github.com/orkspace/orkestra#getting-started)

---

## Step 1 — Generate the registry

```bash
make registry
```

## Step 2 — Build

```bash
make clean && make build
ork validate katalog.yaml
ork simulate -f simulate-v2-enabled.yaml
ork simulate -f simulate-v2-disabled.yaml
```

## Step 3 — Run

```bash
ork run --dev-server
```

In another terminal, start the gateway and get the token:

```bash
export ORK_PORT=8888
ork gate run
```

Apply via the `v2-enabled` surface (feature on, business-hours gate active):

```bash
export TOKEN=$(kubectl get secret ork-dev-token -n default -o jsonpath='{.data.token}' | base64 -d)
```

```bash
ork serve apply -f intent/intent-v2-enabled.yaml --token $TOKEN --api http://localhost:8888
```

Check the result:

```bash
kubectl get deployment 03-hooks-targets-my-chain \
  -o jsonpath='{.metadata.annotations.feature\.demo/v2-enabled}' && echo

kubectl get blockchainappwithtargets 03-hooks-targets-my-chain \
  -o jsonpath='{.status.featureEnabled}' && echo

kubectl get blockchainappwithtargets 03-hooks-targets-my-chain \
  -o jsonpath='{.status.inBusinessHours}' && echo
```

Switch to `v2-disabled` (feature off, no gate):

```bash
ork serve apply -f intent/intent-v2-disabled.yaml --token $TOKEN --api http://localhost:8888
```

> Switching targets cleans up the previous surface's resources automatically.
> `keepPreviousSurface: true` on the target entry skips the cleanup when you
> want both surfaces running simultaneously.

## E2E

```bash
make docker push IMAGE_REPO=yourregistry/blockchainappwithtargets-operator IMAGE_TAG=latest

ork e2e --dev-server \
  --set runtime.image.repository=yourregistry/blockchainappwithtargets-operator \
  --set runtime.image.tag=latest
```

## Cleanup

```bash
chmod +x ./cleanup.sh && ./cleanup.sh
```
