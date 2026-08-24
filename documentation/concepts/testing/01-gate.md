# Gate and Local Gateway

Two tools for testing the admission and delivery layers locally — before a cluster is involved.

---

## `ork gate` — admission rules in isolation

`ork gate` evaluates your Katalog's `validation.rules` and `mutation.rules` against a CR file. The same evaluation engine the admission webhook runs in production, in-process, from a file.

```bash
ork gate -f katalog.yaml --cr cr.yaml
```

Both flags default to their standard filenames, so in a typical Katalog directory:

```bash
ork gate
```

### Output

**Pass:**

```text
▶  ork gate
  cr:      cr.yaml
  katalog: katalog.yaml

◆  PlatformResource  (apifixture)
  ✓ 9/9 rules passed
```

**Deny:**

```text
◆  PlatformResource  (apifixture)
  ✗ spec.workloadType  spec.workloadType must be one of: app, cert, monitoring

  ✗ 8/9 rules passed · ✗ 1 denial(s)

admission denied
```

**Warn:**

```text
◆  PlatformResource  (apifixture)
  ⚠ spec.productionApproval  production deploys should include a productionApproval ticket

  ✓ 8/9 rules passed · ⚠ 1 warning(s)
```

Denials exit non-zero. Warnings exit zero — they are advisory only.

`when:` conditions, `or:` groups, and field path expressions all run identically to the real webhook. A rule guarded by `when: workloadType=cert` does not fire for an `app` CR, just as it wouldn't in the cluster.

---

### Limitations

Two operators are skipped in local mode:

| Operator | Why skipped | Real behaviour |
|----------|-------------|----------------|
| `unique` | Requires a live informer cache | Checked against all existing CRs in the cluster |
| `external` | Requires a real HTTP endpoint | Calls a service before admission |

Both are flagged clearly in the output.

---

### When to use `ork gate` vs `ork serve play`

`ork serve play` runs the full delivery chain — it builds a CR from an intent file and evaluates admission as stage 5. It is the right tool when you own the intent (a CI pipeline, a Control Center form, a webhook).

`ork gate` is the right tool when:

- The CR comes from somewhere else — `kubectl apply`, a GitOps controller, a script.
- You want to check admission rules without constructing an intent.
- You are reproducing a webhook denial and need to isolate which rule fired.

Multi-document CR files are supported. Each document is matched to a CRD by kind.

---

## `ork gate run` — full local gateway

`ork gate run` starts the Gateway process locally — HTTP only, no TLS, no deployed pods. The Gateway API and webhook intake handlers run exactly as they do in production; admission and conversion webhooks are disabled because TLS is not available in local mode.

```bash
ork gate run -f katalog.yaml
```

```text
⎈  ork gate run
  local mode — admission and conversion webhooks are disabled (TLS not available)
  gateway API: http://localhost:8080
```

Once running, you can call `ork serve apply` against it:

```bash
# Terminal 1
ork gate run -f katalog.yaml

# Terminal 2
ork serve apply -f intent.yaml -t my-token
ork serve apply -f intent.yaml -t my-token --verbose   # see the raw response
```

This is the closest local equivalent to a full production delivery round-trip — the same JSON payload, the same token check, the same CR returned in the response — without a Helm deployment or a cluster.

### Difference from production

| | `ork gate` (production binary) | `ork gate run` (local) |
|-|-------------------------------|------------------------|
| Transport | HTTPS + TLS | HTTP only |
| Admission webhooks | Yes | No |
| Conversion webhooks | Yes | No |
| Gateway API + intake | Yes | Yes |

---

## Where to go next

- [Testing in Orkestra](index.md) — full testing overview
- [Local intent testing](../self-service/06-local-intent-testing.md) — `ork serve play` in depth
- [`ork gate` CLI reference](../../reference/cli/14-gate.md)
- [`validation.rules` schema](../../reference/schema/02-katalog/07-validation.md)
