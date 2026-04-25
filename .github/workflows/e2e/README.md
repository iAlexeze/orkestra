# Orkestra E2E Suite

This directory contains end‑to‑end tests that demonstrate Orkestra’s capabilities across real operator scenarios.  
Each test uses the official **Orkestra GitHub Action** to:

- scaffold an operator using `ork init`
- load an example pack
- generate a bundle
- install the Orkestra runtime via Helm
- apply CRDs and CRs
- verify reconciliation and cleanup

These E2E tests use the same example packs featured in  
**Learning to Orkestrate**, the guided path for understanding Orkestra operators.

## How it works

Each workflow corresponds to a single example pack:

```
pack → e2e workflow
```

For example:

- `beginner/01-hello-website` → creates a Deployment  
- `beginner/02-website-with-service` → Deployment + Service  
- `beginner/03-secret-copy` → multi‑namespace secret propagation  
- `beginner/03b-configmap-copy` → ConfigMap distribution  

The workflows run in GitHub Actions using a kind cluster and complete in under ~90 seconds.

## Example snippet

```yaml
- name: Setup operator
  uses: orkspace/orkestra-action@v1
  with:
    init: true
    pack: beginner
    example-subdir: 02-website-with-service
    validate: true
    generate-bundle: true
```

This scaffolds the operator, loads the example pack, validates and prepares all artifacts needed for the test.

## Why this exists

These tests serve two purposes:

1. **Showcase Orkestra’s capabilities** in real scenarios  
2. **Provide a welcoming place for contributors** to add new packs and new E2E workflows

Each new example pack can be paired with a matching E2E workflow, keeping the examples and the platform growing together.