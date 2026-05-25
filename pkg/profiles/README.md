# pkg/profiles

`profiles` is the single source of truth for all named presets in Orkestra. Resource, security, probe, and autoscaler profiles each expand into fully-formed types at katalog load time — the runtime never sees a profile name.

Both `pkg/katalog` (validation and expansion) and `pkg/orkestra-registry` (runtime resolution) import this package. It imports only `pkg/types` and `pkg/utils`, keeping the dependency graph clean.

## Developer documentation

| I want to… | Go to |
|-----------|-------|
| See every profile name and what it expands to | [docs/01-profiles.md](docs/01-profiles.md) |
| Understand how profiles fit in the katalog load and runtime flow | [docs/02-internals.md](docs/02-internals.md) |
| Add a new profile name or a new profile kind | [docs/03-adding-a-profile.md](docs/03-adding-a-profile.md) |
