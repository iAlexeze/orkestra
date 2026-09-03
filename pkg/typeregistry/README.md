# pkg/typeregistry

This package contains a single generated file — `zz_generated_typeregistry.go`. Do not edit it manually.

Regenerate it whenever the Katalog changes:

```sh
ork generate registry -f katalog.yaml
```

The file wires Go types and hook/constructor functions into the Orkestra runtime's type registries (`ObjectRegistry`, `ListRegistry`, `HookRegistry`, `ReconcilerRegistry`). Without it, typed CRDs cannot be decoded and custom reconcile logic is not called.

For a full explanation of what gets generated and why, see [pkg/tools/generate/docs/03-type-registry.md](../tools/generate/docs/03-type-registry.md).
