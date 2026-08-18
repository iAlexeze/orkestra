# 15 — Lifecycle Compatibility

Declare which Kubernetes and Orkestra versions a pattern has been verified against. These constraints are advisory at `ork validate` time (syntax is checked) and enforced at apply time against the running cluster version and the baked-in Orkestra version.

## Files

| File | Purpose |
|------|---------|
| `katalog.yaml` | Stable Katalog with `lifecycle.compatibility` declaring version ranges |
| `crd.yaml` | WebApp CRD |

---

## The compatibility block

```yaml
lifecycle:
  maturity: stable
  compatibility:
    kubernetes: ">=1.28"        # verified against Kubernetes 1.28 and above
    orkestra: ">=0.7.0"         # verified against Orkestra 0.7.0 and above
```

Both fields accept [Masterminds/semver](https://github.com/Masterminds/semver) range syntax — the same library Helm uses:

```
>=1.28          at least 1.28
>=1.28, <1.33   between 1.28 and 1.33 exclusive
^1.28           >=1.28, <2.0
~1.28           >=1.28, <1.29
```

---

## Validate — syntax check

`ork validate` checks that the range string is valid semver syntax. It does not connect to a cluster or check the live version.

```bash
ork validate
# ✓ webapp  valid
```

An invalid range fails immediately:

```yaml
compatibility:
  kubernetes: "!!!invalid"
# ✗ lifecycle.compatibility.kubernetes: "!!!invalid" is not a valid semver range
```

---

## Apply-time enforcement

When `ork run` or `ork serve apply` loads the Katalog, it compares:

- `compatibility.kubernetes` against the live cluster server version
- `compatibility.orkestra` against the version baked into the `ork` binary at build time

A mismatch produces a clear error before any reconcile loop starts:

```
✗ compatibility.kubernetes: cluster is 1.27.3 — pattern requires >=1.28
```

No flag is needed for the Orkestra version check — the build version is always available locally.

---

## Next step

→ [16-komposer-accept](../16-komposer-accept/README.md) — a Komposer that acknowledges the lifecycle state of all its imported patterns
