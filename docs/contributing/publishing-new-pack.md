# Publishing a New Example Pack – The Right Way (Don't Repeat My 3 CI Builds)

I just made a costly mistake: missing a few steps when adding a new pack cost me **three extra CI builds**. Each missed step → failed pipeline → wasted time and resources. Here's the complete, correct checklist so you don't repeat it.

---

## Two scenarios

### 1. Adding an example to an existing pack

If you're adding a new example subdirectory to an existing pack (e.g., add `05-new-example` under `beginner/`), you only need to:

1. Create the example directory with the required files (`katalog.yaml`, `crd.yaml`, `cr.yaml`, `README.md`, etc.)
2. Optionally add an E2E workflow for the example (if it exercises a new feature)

**No code changes** – the pack is already embedded.

---

### 2. Adding a completely new pack

If you're creating a brand‑new pack (e.g., `maintenance`), you must update **four places** (plus optional tests). Missing any of these breaks CI.

#### Step 1 – Create the pack directory

```
examples/new-pack/
  example-1/
  example-2/
  ...
```

Each example must contain the standard files (`katalog.yaml`, `crd.yaml`, `cr.yaml`, `README.md`, optionally `cleanup.sh`, etc.)

---

#### Step 2 – Update [examples/embed.go](../../examples/embed.go)

Add the new pack to the `//go:embed` line:

```go
//go:embed beginner intermediate advanced security use-cases new-pack
var FS embed.FS
```

> **Why:** The CLI uses embedded filesystem to serve examples for `ork init`.

---

#### Step 3 – Update [cmd/cli/init_helper.go](../../cmd/cli/init_helper.go)

Add the pack name and its path to the `packPaths` map:

```go
var packPaths = map[string]string{
    "beginner":     "beginner",
    "intermediate": "intermediate",
    "advanced":     "advanced",
    "security":     "security",
    "use-cases":    "use-cases",
    "new-pack":     "new-pack",   // <--- add this
}
```

> **Why:** This maps the CLI argument `--pack new-pack` to the correct embedded directory.

---

#### Step 4 – Update [.github/workflows/package-examples.yml](../../.github/workflows/package-examples.yml)

In the **"Package Example Packs"** step, add a `tar` command for the new pack:

```yaml
# New pack
tar -czf "dist/examples_new-pack_${TAG}.tar.gz" \
  -C examples/new-pack .
```

Place it after the existing packs (order doesn't matter).

> **Why:** CI packages examples as release artifacts for users to download.

---

#### Step 5 – Update [.github/workflows/sign-and-release.yml](../../.github/workflows/sign-and-release.yml)

In the **"Create GitHub Release"** step, add the new artifact:

```yaml
dist/examples_new-pack_${{ github.ref_name }}.tar.gz
```

Add it to the list of files to upload.

> **Why:** So the tarball appears in the GitHub release.

---

#### Step 6 (Optional but recommended) – Add E2E tests

Create a new E2E workflow for the pack/example (e.g., `.github/workflows/e2e-new-pack.yml`) following the pattern of existing E2E tests (e.g., `e2e-beginner-pack-01.yml`). This validates that the pack works correctly in a live cluster.

---

## The Expensive Mistake I Made

I added a new pack but forgot to:
1. Update `embed.go` → first build failure
2. Update `packPaths` → second failure (CLI couldn't find the pack)
3. Update `package-examples.yml` → third failure (missing tarball in release)

**Three CI runs wasted.** Don't be me.

---

## Checklist for adding a new pack

- [ ] Create `examples/<pack>/`
- [ ] Add example subdirectories with all required files
- [ ] Update `examples/embed.go` (add to `//go:embed` line)
- [ ] Update `cmd/cli/init_helper.go` (add to `packPaths` map)
- [ ] Update `.github/workflows/package-examples.yml` (add `tar` command)
- [ ] Update `.github/workflows/sign-and-release.yml` (add artifact)
- [ ] (Optional) Add E2E workflow for the new pack/example
- [ ] Run `make generate` (if needed) and commit all changes
- [ ] Push and watch CI pass ✅

---

This document will be added to `docs/contributing/publishing-new-pack.md` to help future contributors (and my future self). 🚀