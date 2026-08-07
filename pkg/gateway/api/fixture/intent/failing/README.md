# Failing Intent Files

Intent files that exercise the validation and permission chain in `ork serve play`.
Each one is designed to fail at a specific stage so you can see the error surface.

```bash
KATALOG=pkg/gateway/api/fixture/katalog.yaml
```

---

## Stage 1 failures — target resolution

### Unknown target

```bash
ork serve play -f $KATALOG --token control-center \
  -i pkg/gateway/api/fixture/intent/failing/unknown-target.yaml
```

`target: doesnotexist` — not declared in the katalog. Fails at stage 1 with the
list of available targets.

---

## Stage 2 failures — token check

### `ci-pipeline` attempting create

`ci-pipeline` has `get` and `list` at the CRD level — `create` is not in its
permissions.

```bash
ork serve play -f $KATALOG --token ci-pipeline \
  -i pkg/gateway/api/fixture/intent/failing/ci-pipeline-create.yaml
```

### `control-center` on a read-only alias

`preview` restricts `control-center` to `get` and `list`. Create is denied at
the alias token level.

```bash
ork serve play -f $KATALOG --token control-center \
  -i pkg/gateway/api/fixture/intent/failing/preview-create.yaml
```

### `ci-pipeline` on a restricted alias

`ci-pipeline` is not listed in the `internal` alias token map at all — it is
denied entirely regardless of operation, even though it has CRD-level access on
the primary target.

```bash
ork serve play -f $KATALOG --token ci-pipeline \
  -i pkg/gateway/api/fixture/intent/failing/ci-pipeline-on-alias.yaml
```

---

## Stage 3 failures — CR construction

### Missing name

`name` is omitted. The fixture does not declare `serve.name`, so the caller
must supply it. The chain runs to completion — but the CR body printed at
stage 4 shows no `metadata.name`, and the summary shows `PlatformResource/ in
team-payments`. That is what would be sent to SSA; the real gateway rejects it
there with "metadata.name is required" before the CR lands.

```bash
ork serve play -f $KATALOG --token control-center \
  -i pkg/gateway/api/fixture/intent/failing/missing-name.yaml
```

---

## Relationship to `ork simulate`

The gateway is an intent runner. The runtime is a CR runner.

The gateway collects intentions — flat fields from a caller — and translates them
into a valid Kubernetes object. The runtime takes that object and reconciles it
into cluster resources. Neither needs a cluster to do its job locally.

- `ork serve play` runs the **gateway's** half: intent in, CR out.
- `ork simulate` runs the **runtime's** half: CR in, child resources out.

Together they cover the full delivery loop without a cluster:

```
intent file  →  ork serve play  →  CR  →  ork simulate  →  child resources
```
