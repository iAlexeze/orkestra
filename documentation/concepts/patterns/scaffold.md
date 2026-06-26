# Scaffolding a Pattern

`ork create pattern` generates the full file set needed to build, test, and publish an Orkestra pattern. It surfaces the same katalog scaffolding as `ork generate katalog` and adds the testing layer — `simulate.yaml` and `e2e.yaml` — so the operator suite is complete from the start.

```bash
ork create pattern
ork create pattern -o ./my-operator/
ork create pattern --add-hook -o ./my-operator/
```

---

## What gets generated

```text
my-operator/
  katalog.yaml    — operator declaration
  simulate.yaml   — in-memory test (ork simulate)
  e2e.yaml        — real-cluster test (ork e2e)
  README.md       — actionable steps from edit to release
```

In typed mode (`--add-hook`, `--add-constructor`, `--typed`):

```text
  values.yaml     — runtime image (set before ork e2e)
  Makefile        — registry, build, build-runtime, docker, push, release
  Dockerfile      — production container image (distroless, runtime binary only)
```

---

## Dynamic vs typed

The typed flags are forwarded directly to katalog generation — they behave identically to `ork generate katalog`.

| Flag | Katalog mode | Extra files |
|------|-------------|-------------|
| *(none)* | Dynamic — declarative templates only | — |
| `--add-hook` | Typed — commented `hooks` declaration | `values.yaml`, `Makefile`, `Dockerfile` |
| `--add-constructor` | Typed — commented `constructor` declaration | `values.yaml`, `Makefile`, `Dockerfile` |
| `--typed` | Typed — both sections commented, pick one | `values.yaml`, `Makefile`, `Dockerfile` |

---

## From scaffold to running operator

### Dynamic mode

```bash
# 1. Generate
ork create pattern -o ./my-operator/
cd my-operator/

# 2. Edit katalog.yaml — fill in spec.crds, declare resources
# 3. Validate
ork validate

# 4. Simulate — no cluster needed
ork simulate

# 5. Run locally
ork run --dev

# 6. E2E test
ork e2e

# 7. Observe
ork control
```

### Typed mode

```bash
# 1. Generate
ork create pattern --add-hook -o ./my-operator/
cd my-operator/

# 2. Edit katalog.yaml — fill in apiTypes
# 3. Generate type registry
make registry

# 4. Write your hook function

# 5. Release the image
make release IMAGE=ghcr.io/myorg/my-operator:v0.1.0

# 6. Set the image in values.yaml
#    runtime.image.repository: ghcr.io/myorg/my-operator
#    runtime.image.tag: v0.1.0

# 7. Validate
ork validate

# 8. Simulate
ork simulate

# 9. Run locally
ork run --dev

# 10. E2E test — passes values.yaml to the Orkestra Helm chart
ork e2e

# 11. Push
ork push
```

---

## The testing layer

`simulate.yaml` and `e2e.yaml` are starters — they reference `./katalog.yaml` and `./cr.yaml` but have no assertions yet.

Add assertions before running, or generate them from a live run:

```bash
ork simulate init
```

See [Simulate](../simulate/index.md) and [E2E](../e2e/index.md) for the full test authoring guide.

---

## Publishing

Once your tests pass, push to the registry:

```bash
ork push
```

In typed mode, release the image first, then update `values.yaml` so `ork e2e` can pass it to the Orkestra Helm chart, then push the katalog:

```bash
make release IMAGE=ghcr.io/myorg/my-operator:v0.1.0
# edit values.yaml: set runtime.image.repository and runtime.image.tag
ork e2e
ork push
```

→ See [`ork create pattern` CLI reference](../../reference/cli/create.md#ork-create-pattern)
