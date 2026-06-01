# Declarative End-to-End Testing

`ork e2e` verifies an operator against a real Kubernetes cluster. One YAML file describes what to apply, which cluster to use, and what the expected state is at each checkpoint. No test framework. No Go. No mocks.

---

## Why it exists

`ork simulate` verifies template logic without a cluster — fast, sub-second, catches expression errors and mis-typed field references. That is the inner loop.

E2E is the outer gate. It runs the same reconciler, but against a real API server with real admission webhooks, real RBAC, real pod scheduling, and real image pulls. The things simulate cannot catch — ordering bugs between CRDs, finalizer races, webhook rejections, actual Deployment readiness — are exactly what E2E catches.

The two tools are complementary, not alternatives. Simulate first. E2E before you push.

---

## Every learning example ships with a runnable E2E

All examples in [Learning-to-orkestrate](../../getting-started/01-learning-to-orkestrate/) include an `e2e.yaml`. You do not need to write one from scratch to see the full cycle:

```bash
ork init --pack beginner
cd beginner/01-hello-website
ork e2e
```

That command creates a kind cluster, applies the operator, applies the CR, waits for the Deployment and Service to be ready, deletes the CR, and verifies cleanup. The whole cycle takes about 90 seconds on a cold machine.

---

## The shape of a test

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E

metadata:
  name: hello-website-e2e
  description: Deploy a website, verify it comes up, verify cleanup on delete.

spec:
  katalog: ./katalog.yaml
  crd:     ./crd.yaml
  cr:      ./cr.yaml

  expect:
    - name: Deployment and Service ready
      after: cr-applied
      timeout: 60s
      resources:
        - kind: Deployment
          namespace: default
          ready: true
        - kind: Service
          namespace: default

    - name: Cleanup verified
      after: cr-deleted
      timeout: 30s
      resources:
        - kind: Deployment
          name: hello-website
          namespace: default
          count: 0
```

Four inputs drive every test: the katalog under test, the CRD to install, the CR to apply, and a list of checkpoints (`expect`) that must pass in order.

---

## From one test to a suite

As operators grow, a single `e2e.yaml` per Katalog becomes the norm. When a Komposer combines several Katalogs, a suite file at the root imports all of them:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: E2E
metadata:
  name: full-stack-app-suite

imports:
  - ./01-multi-region/e2e.yaml
  - ./03-cross-crd/e2e.yaml
  - ./04-once-secret/e2e.yaml
  - ./05-anyof/e2e.yaml
```

One `ork e2e` runs the whole suite — one cluster, four tests, sequential. The suite file has no test of its own; it exists only to compose.

**Try it:**
```bash
ork init --pack use-cases/full-stack-app
ork e2e -f e2e.yaml
```

---

## Pages

- [How it works](01-how-it-works.md) — the full lifecycle: cluster provisioning, CRD apply, Orkestra install, expectations, cleanup
- [Writing a test](02-writing-a-test.md) — quick-reference for spec fields with annotated examples
- [Suites and imports](03-suite-and-imports.md) — composing multiple E2E files, cluster strategy, pure aggregators
- [Best practices](04-best-practices.md) — focused tests, naming, `count: 0`, CI integration

---

## See also

- [ork simulate](../simulate/index.md) — the fast inner loop; use before E2E
- [E2E schema reference](../../reference/schema/04-e2e/index.md) — every field, every option
- [Writing Your First E2E](../../getting-started/06-writing-your-first-e2e.md) — step-by-step walkthrough
- [ork e2e CLI reference](../../reference/cli/08-e2e.md)
