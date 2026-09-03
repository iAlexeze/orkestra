# Beginner Pack

The foundation. Every concept here appears in every more advanced example. Work through these in order.

```bash
ork init --pack beginner
cd beginner/01-hello-website
ork run
```

| Example | What it teaches |
|---|---|
| `01-hello-website` | Your first operator. CRD declaration, Katalog template expressions, owner references, status fields. The mental model everything else builds on. |
| `02-with-serviceaccount` | Three resources from one CR. ServiceAccount wiring, pod identity, reading live cluster state into status via Notes. |
| `03-secret-copy` | Built-in Kubernetes resource management. A Secret distribution operator: copies a Secret from a platform namespace into every team namespace. `fromSecret`, `toNamespaces`. |
| `03b-configmap-copy` | Same pattern as 03 applied to ConfigMaps. Statusless resource distribution. Good companion to 03. |
