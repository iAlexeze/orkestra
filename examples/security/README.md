# Security Pack

Protect your cluster from accidental deletions, bad input, and policy violations — enforced at admission time, before resources reach etcd.

```bash
ork init --pack security
```

---

## Examples

| Example | What it teaches |
|---------|-----------------|
| [Admission Webhooks](admission/README.md) | `ValidatingWebhookConfiguration` + `MutatingWebhookConfiguration` on your CRDs. Bad CRs rejected at apply time. Defaults filled in before validation runs. |
| [Deletion Protection](deletion-protection/README.md) | Block deletion of critical CRDs and CRs via admission webhook. Contrast with unprotected resources that can be deleted freely. |
| [Namespace Protection](namespace-protection/README.md) | Restrict what namespaces operators can act on. Allowed namespaces pass; restricted ones are rejected at admission time. |

---

## Running an example

```bash
ork init --pack security
cd security/admission
ork run
```

No cluster? Add `--dev` to create a temporary kind cluster.

---

## E2E

Every example ships with a runnable `e2e.yaml`. Run a single example end-to-end:

```bash
cd admission && ork e2e
```

Or run the full security suite — all three examples in one kind cluster:

```bash
ork e2e -f e2e.yaml
```

The suite spins up a cluster, runs each example in sequence, and tears everything down. Total runtime: ~5 minutes.
