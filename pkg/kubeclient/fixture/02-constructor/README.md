# Args via Constructor — BlockchainNode

The constructor owns its full reconcile loop — including the time check and the
HTTP call. The Katalog declares *what* to check and *when*; the constructor decides
*how*:

```yaml
constructor:
  args:
    flagUrl: '{{ .spec.serviceUrl }}/flags/{{ .metadata.name }}/v2Enabled'
    businessHoursStart: "09:00"
    businessHoursEnd: "18:00"
```

```go
flagUrl := kube.Args().String("flagUrl")
start   := kube.Args().String("businessHoursStart")
end     := kube.Args().String("businessHoursEnd")

featureEnabled := r.inBusinessHours(start, end) && r.checkFlag(ctx, flagUrl)
```

The constructor checks the clock itself and makes the HTTP call itself — but the
window and the URL pattern come from the Katalog, not from the binary. Change the
business-hours window or the flag endpoint in YAML; the binary stays the same.

This is the difference from hooks: the runtime evaluates the note and makes the
external call for hooks. The constructor owns those steps — the Katalog hands it
the configuration to make its own decisions.

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
ork simulate --dev-server
```

## Step 3 — Run

```bash
ork run --dev-server

kubectl get deployment 02-constructor-my-node \
  -o jsonpath='{.metadata.annotations.feature\.demo/v2-enabled}' && echo

kubectl get blockchainnode 02-constructor-my-node \
  -o jsonpath='{.status.featureEnabled}' && echo

kubectl get blockchainnode 02-constructor-my-node \
  -o jsonpath='{.status.inBusinessHours}' && echo
```

> Outside business hours the constructor skips the flag call and the annotation
> is `"false"`. Inside hours with the dev server running it is `"true"`.

## E2E

```bash
make docker push IMAGE_REPO=yourregistry/blockchainnode-operator IMAGE_TAG=latest

ork e2e --dev-server \
  --set runtime.image.repository=yourregistry/blockchainnode-operator \
  --set runtime.image.tag=latest
```

## Cleanup

```bash
./cleanup.sh
```
