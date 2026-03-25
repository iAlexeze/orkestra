# How Dependencies Work

This document explains how Orkestra manages dependencies between CRDs — automatically, without any code.

---

## **The Problem**

When you have multiple CRDs that depend on each other, you need them to start in the right order.

Imagine a web application that needs a database before it can run:

- **Database CRD** must be ready before the **Application CRD** starts
- If the database CRD is missing, the application should wait
- When the database appears, the application should start automatically

Traditional operators don't handle this. You have to write code to check if dependencies exist, wait for them, and retry when they appear.

Orkestra does this for you.

---

## **Declaring Dependencies**

In your Katalog, you declare dependencies with `dependsOn`:

```yaml
crds:
  - name: database
    apiTypes:
      group: database.myorg.io
      version: v1alpha1
      kind: Database
      plural: databases
    # ... other config

  - name: application
    apiTypes:
      group: app.myorg.io
      version: v1alpha1
      kind: Application
      plural: applications
    dependsOn:
      - database      # ← Application depends on Database
    # ... other config
```

That's it. Orkestra handles the rest.

---

## **How It Works**

### **1. Orkestra Builds a Graph**

When Orkestra starts, it reads all your CRDs and their dependencies. It builds a graph that shows which CRD depends on which.

```
database ──┐
           │
           ▼
     application
```

### **2. Orkestra Computes the Order**

From the graph, Orkestra computes the order to start CRDs:

- CRDs with no dependencies start first
- CRDs that depend on others start only after their dependencies are ready

In the example:
1. `database` starts first (no dependencies)
2. `application` starts second (waits for `database`)

### **3. Orkestra Creates Ready Channels**

For each CRD, Orkestra creates a "ready channel" — think of it as a closed door.

- If the CRD exists in the cluster, the door opens immediately
- If the CRD is missing, the door stays closed

### **4. Dependents Wait**

When Orkestra reaches a CRD that depends on others, it waits for those doors to open.

```
application: waiting for database door to open...
```

### 5. **When a Missing CRD Appears**

If a CRD is missing at startup, Orkestra doesn't stop. It keeps checking in the background.

When the missing CRD finally appears:

1. Orkestra detects it
2. It starts the CRD's workers
3. It opens the ready door
4. All waiting dependents automatically start

---

## **Example in Action**

### Scenario: Database Missing at Startup

You start Orkestra, but the `database` CRD isn't installed yet. Only the `application` CRD exists.

```
1. Orkestra sees database is missing
2. Orkestra sees application depends on database
3. application starts waiting for database
4. Orkestra keeps checking for database in the background
```

Your cluster has a `database` CR installed (the resource), but the `database` CRD (the definition) is missing.

Later, you install the `database` CRD:

```bash
kubectl apply -f database-crd.yaml
```

Orkestra detects the CRD:

```
5. Orkestra notices database CRD appears
6. Orkestra starts database workers
7. Orkestra opens the ready door for database
8. application sees the door open and starts its workers
9. application reconciles any existing Application CRs
```

The system self‑heals. No restart needed.

---

## What If a CRD Is Deleted Later?

If you delete a CRD while Orkestra is running:

- The CRD's informer keeps running (it logs errors, but it's harmless)
- The workers stop (they can't run without the CRD)
- Dependents that were already started continue running (they're already active)
- If the CRD is recreated later, the informer recovers automatically

You don't need to restart Orkestra.

---

## **Shutdown Order**

When Orkestra shuts down, it stops CRDs in the **reverse** order they started.

```
Startup:   database → application
Shutdown:  application → database
```

This ensures dependents stop before their dependencies. No broken references.

---

## **Visualizing Dependencies**

You can see the dependency graph with:

```bash
ork template --katalog my-katalog.yaml --graph
```

Output:
```
Dependency Graph:
database
application
  └─ database
```

This shows that `application` depends on `database`.

---

## **What About Circular Dependencies?**

Orkestra detects circular dependencies and refuses to start:

```yaml
crds:
  - name: a
    dependsOn: [b]
  - name: b
    dependsOn: [a]
```

Error:
```
dependency cycle detected involving a → b → a
```

Fix your dependencies before running.

---

## **Under the Hood**

Orkestra uses a simple but powerful mechanism:

- **Graph** — built from your Katalog at startup
- **Ready channels** — one per CRD, closed when the CRD is ready
- **Retry loop** — checks for missing CRDs in the background
- **Topological sort** — computes startup order
- **Reverse order** — for clean shutdown

No code. Just declaration.

---

## **Summary**

| What | How Orkestra Handles It |
|------|------------------------|
| **Declare dependencies** | `dependsOn: [database]` in your Katalog |
| **Startup order** | Automatic — dependencies first |
| **Missing CRDs** | Wait, retry in background, activate when they appear |
| **CRD deletion** | Workers stop, informer recovers when recreated |
| **Shutdown order** | Reverse of startup |
| **Circular dependencies** | Detected and rejected |

**You declare the dependencies. Orkestra handles the order.** 🎼