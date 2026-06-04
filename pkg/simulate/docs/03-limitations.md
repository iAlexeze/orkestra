# 03 — What simulate does not cover

The fake cluster starts empty and has no external connectivity. Some operator features require a real cluster.

## Current gaps

| Feature | Status | Notes |
|---------|--------|-------|
| `external:` HTTP calls | Active by default — hits real network | Pass `--skip-external` to stub with empty 200 responses |
| `cross:` CRD observation | Active when peer CRs are in the CR file | Separate sibling CRDs' CRs with `---`; each is seeded into a fake informer |
| Go hooks (typed reconcile hooks) | Active — runs from compiled custom binary | Build your operator binary with `make registry && make build`; hooks fire against the fake cluster |
| Custom constructors | Active — runs from compiled custom binary | Same binary requirement as hooks |
| Multi-doc CR files | Active — each CRD matched to its CR by `kind` | CRDs with no matching CR are skipped with a note |
| Cross-namespace reads (secrets, configmaps) | Not active | `ork e2e` |
| Git hooks | Not active | `ork e2e` |
| Provider blocks (AWS, MongoDB) | Not active | `ork e2e` |
| Admission webhook validation/mutation | Not registered | `ork e2e` |
| Watch events / real informer updates | Not executed — indexer is static | `ork e2e` |
| Status written back to indexer | Not persisted — state machine only advances one step | `ork e2e` for multi-phase flows |
| Real pod scheduling and readiness | Deployments are marked ready immediately | `ork e2e` |

For anything in the "`ork e2e`" column, provision a real kind cluster and run against the actual operator runtime.

→ Next: [04-internals.md](04-internals.md)
