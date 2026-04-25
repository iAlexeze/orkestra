### Good first issue — Add a new example pack E2E workflow

**Title suggestion:** `Good first issue: add E2E workflow for <pack-name> example`

---

**What to do**

Add a new GitHub Actions workflow that runs the existing E2E pattern for a chosen example pack under `orkestra/examples/`. The pattern is simple:

**pack → e2e workflow**

---

**Why this helps**

- Expands the runnable example library  
- Keeps examples versioned with Orkestra  
- Provides an easy, meaningful first contribution for new maintainers

---

**Acceptance criteria**

- A new workflow file added to `.github/workflows/` (name: `e2e-<pack-name>.yml`)  
- Workflow uses the Orkestra Action to scaffold the operator and generate a bundle  
- Workflow installs Orkestra via Helm, applies CRD + CR, verifies reconciliation, and verifies cleanup  
- Workflow is self-contained and references the example under `orkestra/examples/<pack-path>`  
- Add a one‑line description at the top of the workflow explaining the example

---

**Step by step**

1. Pick an example under `orkestra/examples/<pack>/<example-subdir>`.  
2. Copy the beginner E2E workflow as a template.  
3. Update `example-subdir` and any `setup.yaml` paths to match the example.  
4. Ensure the workflow applies the CRD, bundle, and CR, waits for reconciliation, and deletes the CR to verify garbage collection.  
5. Commit the workflow to a branch and open a PR with the label **good first issue**.

---

**Helpful resources**

- E2E README: `orkestra/e2e/README.md`  
- CONTRIBUTING: `orkestra/e2e/CONTRIBUTING.md`  
- Workflow template (copy this into `.github/workflows/e2e-<pack-name>.yml`):

```yaml
name: Orkestra E2E - <pack-name>

on: workflow_dispatch

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Create kind cluster
        uses: helm/kind-action@v1
        with:
          cluster_name: ork-e2e
      - name: Setup operator and generate bundle
        id: ork
        uses: orkspace/orkestra-action@v1
        with:
          init: true
          pack: <pack>
          example-subdir: <example-subdir>
          validate: true
          generate-bundle: true
      # follow the beginner pattern: apply CRD, apply bundle, install helm, apply CR, verify, cleanup
```

---

**Notes for maintainers**

- Keep the workflow minimal and self-contained so it runs in a kind cluster.  
- If the example needs namespaces or source resources, include a `setup.yaml` under the example and apply it in the workflow.  
- Mark the PR with **good first issue** and request a quick review.
