# Normalize Examples

Four focused examples showing what `normalize` does and why it matters. Each runs independently with `ork run`. No Helm. No webhooks.

| Example | What it teaches |
|---|---|
| [01 — String Cleanup](01-string-cleanup/README.md) | `toLower`, `trimSpace`, domain stripping — accept any casing, produce one canonical value |
| [02 — Image Normalization](02-image-normalization/README.md) | Ensure registry prefix and explicit tag regardless of what the user wrote |
| [03 — Defaults Without a Webhook](03-defaults-without-webhook/README.md) | `default` inside normalize = mutation without deploying Orkestra Gateway |
| [04 — Full WebService](04-webservice/README.md) | All patterns combined — image, environment, domain, secrets, configmap, forEach backends |

Start with 01. Each example builds on the same concept. 04 is the full picture.

---

**Further reading:** [Normalize concept doc](https://orkestra.sh/docs/concepts/operatorbox/normalize)

---

## E2E

Run the full suite — all four normalize examples in one command:

```bash
ork e2e -f e2e.yaml
```

This runs [e2e.yaml](./e2e.yaml), which imports each sub-example's `e2e.yaml` and runs them sequentially in the same cluster. To run a single example:

```bash
cd 04-webservice && ork e2e
```
