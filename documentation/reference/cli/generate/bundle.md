# ork generate bundle

Generate a complete Orkestra installation bundle containing RBAC and a ConfigMap embedding your Katalog.

```bash
ork generate bundle --file <file> [flags]
```

The bundle is self‑contained and ready to apply directly to a Kubernetes cluster.  
It includes the minimal RBAC required for your CRDs and a ConfigMap containing your Katalog.

---

## Flags

| Flag | Description |
|------|-------------|
| `-k, --file <file>` | One or more Katalog files (required) |
| `-o, --output <file>` | Write output to file (default: stdout) |
| `-n, --namespace <name>` | Namespace for generated resources (default: `orkestra-system`) |
| `--dry-run` | Print output without writing files |

---

## Usage

Generate a bundle:

```bash
ork generate bundle --file katalog.yaml
```

Write to a file:

```bash
ork generate bundle --file katalog.yaml -o bundle.yaml
```

Custom namespace:

```bash
ork generate bundle --file katalog.yaml --namespace production
```

Multiple Katalogs:

```bash
ork generate bundle --file a.yaml --file b.yaml
```

---

## Behavior

- Merges one or more Katalog files.
- Validates the merged Katalog.
- Generates:
  - ServiceAccounts (runtime + control center)
  - ClusterRole with minimal required permissions
  - ClusterRoleBinding
  - ConfigMap embedding the Katalog
- Produces a single YAML document stream:
  - RBAC
  - `---`
  - ConfigMap

---

## Notes

- This is the recommended way to deploy Orkestra in production.
- The output is deterministic and GitOps‑friendly.
- The bundle can be applied directly:

```bash
kubectl apply -f bundle.yaml
```
