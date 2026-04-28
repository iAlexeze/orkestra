# Contributing to the Orkestra E2E Suite

Thank you for your interest in contributing.  
The E2E suite is designed to be simple, fast, and welcoming — a great place for first‑time contributors.

There are two main ways to contribute:

---

## 1. Use an existing example pack

If you want to add a new E2E workflow using an existing pack:

1. Explore the example packs under `orkestra/examples/`
2. Pick a pack and example subdirectory
3. Create a new workflow in `.github/workflows/`
4. Follow the pattern used in the beginner E2E tests:
   - scaffold operator with `ork init`
   - apply CRD + bundle
   - install Orkestra via Helm
   - apply CR
   - verify reconciliation
   - verify cleanup

This is the easiest way to get started.

---

## 2. Create a new example pack

If you want to contribute a new operator scenario:

1. Add a new pack or example under `orkestra/examples/`
2. Include:
   - `crd.yaml`
   - `cr.yaml`
   - `katalog.yaml`
   - `README.md`
   - any supporting files (e.g., setup.yaml)
3. Add a matching E2E workflow following the same pattern as the beginner tests

The rule is simple:

```
pack → e2e workflow
```

This keeps the examples and the tests growing together as Orkestra evolves.

---

## Why this pattern matters

By pairing each example pack with an E2E workflow:

- examples stay **versioned** with Orkestra  
- tests stay **relevant** as the platform grows  
- contributors can add new scenarios without touching the core  
- the community benefits from real, runnable operator examples  
- the E2E suite becomes a living showcase of Orkestra’s capabilities  

This is more than testing — it’s a shared library of operator patterns.

---

## Need help?

Open an issue or join the discussion.  
We’re excited to see what you build.
