# External

`external:` makes HTTP calls before any resource group runs. Results land in `.external.<name>.*` and are available to every resource template, `when:` condition, and status field for the rest of that reconcile cycle. A single external call can gate a Deployment, embed a live config into a ConfigMap, sign an image, or supply the auth token for the next call in the sequence.

`external:` is a field on `HookTemplates` — the same struct used by `onCreate` and `onReconcile`. Declare it under whichever lifecycle hook owns the resources that consume it:

- **`onReconcile:`** — the most common placement. The call runs every reconcile alongside drift-correction. Resources under `onReconcile` are already re-applied on every cycle; `reconcile: true` on individual resources is redundant here.
- **`onCreate:`** — use when the external call gates resources that need both creation and drift correction. Declare `reconcile: true` on those resources so Orkestra updates them (not just creates them) on every reconcile.

**Why it exists.** Operators often need to coordinate with the world outside the cluster — an upstream health check, a signing service, a feature flag API, an external auth provider. Without `external:`, you would write Go hooks for each. With `external:`, you declare the call and reference the result in templates. The operator handles the HTTP, the retry, the timeout, and the error surfacing.

---

## The key design decision

Every external call is either a **hard prerequisite** or **optional enrichment**. This is `continueOnError`.

`continueOnError: false` (the default) halts the reconcile when the call fails and writes `Ready=False` to the CR condition. Use this when the call is an infrastructure requirement — there is no meaningful state without it.

`continueOnError: true` lets the reconcile continue when the call fails, with `.error` set and the result available in status fields. Use this when the call result carries meaning even on failure — a rejection, a degraded flag value, a cached config.

The distinction matters: a 403 from a signing service is a policy decision, not an infrastructure failure. The operator should keep reconciling, surface the rejection in status, and enforce it through a `when:` gate on the Deployment — not halt with a raw error.

---

## Where external sits in the pipeline

```text
informer cache → normalize → mutation → validation
    → cross-CRD reads          (.cross.* available in url: and body:)
    → external calls           ← you are here
    → resource groups
    → enrich → status fields
```

External runs after `cross:` context is injected, so `.cross.*` is accessible in `url:` and `body:` fields. Every call's result is available to every resource group and status field that follows.

---

## Pages

- [Patterns](01-patterns.md) — health gate, config injection, image signing with rejection tracking, chaining, feature flag rollout
- [Best practices](03-best-practices.md) — when to gate calls, 4xx vs 5xx, driving resource attributes, tokens, timeouts, naming
- [Reference](02-reference.md) — full field table, wire format, result context, constraints
