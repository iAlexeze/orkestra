# When Conditions

Orkestra has two levels of conditional evaluation — one inside the reconciler,
one before it. This pack covers both.

```bash
ork init --pack intermediate/05-when-conditions
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [Conditional Resources](conditional-resources/README.md) | `when:` on individual resources inside the reconciler. Same CRD, different topology per tier — LoadBalancer only for pro and enterprise, monitoring ConfigMap only when enabled. |
| [Conditional Reconciliation](conditional-reconciliation/README.md) | `operatorBox.reconcile:` evaluated by the kordinator before the reconciler is ever called. When conditions fail, the item is discarded — no resources touched, no error recorded, operator stays healthy. |

---

## Running an example

```bash
cd conditional-resources
ork run
```

```bash
cd conditional-reconciliation
ork run
```

---

## Simulate

Conditional reconciliation ships with envtest simulate scenarios — presence
(gate passes) and absence (gate discards). Run them against a real API server:

```bash
ork simulate -f conditional-reconciliation/simulate.yaml --envtest
```

---

## E2E

Each example ships with an `e2e.yaml`. Run one:

```bash
ork e2e -f conditional-resources/e2e.yaml
ork e2e -f conditional-reconciliation/e2e.yaml
```
