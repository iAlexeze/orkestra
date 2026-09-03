# Security Pack

Admission, deletion protection, and namespace isolation — each runnable independently.

```bash
ork init --pack security
```

| Example | What it teaches |
|---|---|
| `admission` | Validation and mutation at admission time. `deny`, `warn`, `default`, `override` rules. No webhook server — Orkestra handles the webhook endpoint when the operator is deployed. |
| `deletion-protection` | Preventing accidental deletion of CRs carrying live state. Finalizer management, deletion condition evaluation, and safe teardown sequencing. |
| `namespace-protection` | Restricting which namespaces an operator will act in. Scoped operator identity — the operator ignores CRs outside its permitted namespaces. |
