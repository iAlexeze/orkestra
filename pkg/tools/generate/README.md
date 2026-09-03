# pkg/tools/generate

`generate` transforms a Katalog into deployable Kubernetes artifacts and Go code. Each sub-command reads one or more `katalog.yaml` files, merges them, and writes a specific type of output.

```sh
ork generate registry  -f katalog.yaml          # Go TypeRegistry code
ork generate rbac      -f katalog.yaml          # RBAC ClusterRoles + ServiceAccounts
ork generate configmap -f katalog.yaml          # ConfigMap embedding the Katalog
ork generate bundle    -f katalog.yaml          # RBAC + ConfigMap in one file
ork generate dashboards -f katalog.yaml         # Grafana dashboard JSON (starting point)
ork generate katalog                            # Starter katalog.yaml scaffold
ork generate crd       -f katalog.yaml          # CRD + sample CR manifests
ork generate all       -f katalog.yaml          # registry + dashboards in one pass
```

All file-writing commands accept `--dry-run` to print output to stdout instead.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| Understand all generators and what each produces | [docs/01-generators.md](docs/01-generators.md) |
| Understand RBAC, ConfigMap, and Bundle generation | [docs/02-bundle-rbac.md](docs/02-bundle-rbac.md) |
| Understand TypeRegistry code generation | [docs/03-type-registry.md](docs/03-type-registry.md) |
| Understand CRD and sample CR generation | [docs/04-crd-generation.md](docs/04-crd-generation.md) |
