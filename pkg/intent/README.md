# pkg/intent

`intent` is the experimental layer where the runtime meets the intent model.

The gateway proved that operators can receive intent — a flat, human-vocabulary payload with no `apiVersion`, no `kind`, no `spec` — and translate it into a Kubernetes CR without callers ever seeing the manifest. `intent` explores what the runtime can do with that, once the gateway hands a CR off for reconciliation.

The questions driving this layer:

- **How far can intent travel?** The gateway stamps provenance; can the runtime route, dispatch, and gate on it without losing the original context?
- **What does per-target mean for the runtime?** The gateway resolves a target surface from the caller's vocabulary; what does the reconciler do differently for each surface?
- **Can the manifest stay an implementation detail end-to-end?** From intent → CR → reconciler, without the caller or the operator code needing to understand Kubernetes structure?

This is not a finished answer — it is an active investigation. Code here reflects what has been learned so far. APIs may change.

## Open questions

**Where does intent live between delivery and replay?**

Intent can arrive from anywhere — a CI pipeline, a Slack command, a browser form, a cron job. Unlike GitOps, where the manifest lives in a Git repository and can be replayed by re-applying the commit, intent has no durable store today. The CR carries provenance annotations (target, alias, source identity), but the CR is already the translation result — the original flat payload is gone.

One direction: an intent registry backed by OCI. The gateway is the sole writer — it records the raw intent payload alongside the provenance it stamps on the CR, as an OCI artifact with matching annotations. The content address (SHA of the payload) gives natural deduplication. `ork serve replay --registry` reads from the store and re-runs each intent through the *current* gateway — so replay re-derives the CR from today's field translations, not the schema that existed when the intent was first delivered. Schema evolution becomes transparent to replay.

This is qualitatively different from replaying a manifest from Git. Git replay re-applies what the cluster received then. Intent replay re-derives what the cluster should receive now.

`pkg/registry` already uses OCI for operator pattern distribution. The annotation model it uses maps directly onto what intent provenance needs.

| Sub-package | Responsibility |
|-------------|----------------|
| [target/](target/README.md) | Target resolution, intent-to-CR translation, per-target reconciler dispatch |
