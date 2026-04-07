---
title: "Writing Hooks"
weight: 42
---


# Writing Hooks

Orkestra's declarative templates cover 80% of use cases. For the remaining 20%, you can write **hooks** — Go functions that run during reconciliation.

Hooks give you full access to the CR, the Kubernetes client, and the context. You can:

- Call external APIs
- Implement complex validation
- Perform cross‑resource checks
- Integrate with external systems

---

## When to Use Hooks

Use hooks when you need logic that can't be expressed in templates:

| Use Case | Template | Hook |
|----------|----------|------|
| Create a Deployment | ✅ Yes | |
| Create a Service | ✅ Yes | |
| Copy a Secret | ✅ Yes | |
| Call an external API | ❌ No | ✅ Yes |
| Validate against a database | ❌ No | ✅ Yes |
| Cross‑resource checks | ❌ No | ✅ Yes |
| Custom business logic | ❌ No | ✅ Yes |

---

## Hook Structure

A hook is a Go function that returns a `ReconcileHooks` struct. You can implement one, two, or all three methods:

```go
type ReconcileHooks[T Object] struct {
    OnReconcile func(ctx context.Context, obj T) error  // Create/Update
    OnDelete    func(ctx context.Context, obj T) error  // Deletion cleanup
    OnNotFound  func(ctx context.Context, key string) error // Object missing
}
```

All methods are optional. Implement only what you need.

---

## Scaffolding a Project with Hooks

```bash
# Scaffold a typed project (hooks require compiled types)
ork init my-operator --typed
cd my-operator
```

This creates:
```
my-operator/
  cmd/
    orkestra/
      main.go
  pkg/
    runtime/        # generated files
    hooks/          # your hooks go here
  katalogs/
  go.mod
```

---

## Writing Your First Hook

Let's write a hook for a `Website` CR that calls an external API to validate the domain name.

### Step 1: Create the Hook File

Create `pkg/hooks/website.go`:

```go
package hooks

import (
    "context"
    "fmt"
    "net/http"

    "github.com/ialexeze/orkestra/domain"
    websitev1 "github.com/myorg/apis/website/v1alpha1"
)

func WebsiteHooks() domain.ReconcileHooks[domain.Object] {
    return domain.ReconcileHooks[domain.Object]{
        OnReconcile: func(ctx context.Context, obj domain.Object) error {
            // Type‑assert to the concrete type
            website, ok := obj.(*websitev1.Website)
            if !ok {
                return fmt.Errorf("expected *websitev1.Website, got %T", obj)
            }

            // Call external API
            resp, err := http.Get("https://api.dnsvalidator.com/check?domain=" + website.Spec.Domain)
            if err != nil {
                return fmt.Errorf("domain validation failed: %w", err)
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("domain %s is not valid", website.Spec.Domain)
            }

            return nil
        },
        OnDelete: func(ctx context.Context, obj domain.Object) error {
            // Cleanup external resources when the CR is deleted
            website := obj.(*websitev1.Website)
            // Call API to deregister the domain
            return nil
        },
    }
}
```

### Step 2: Declare the Hook in Your Katalog

```yaml
crds:
  website:
    apiTypes:
      group: demo.orkestra.io
      version: v1alpha1
      kind: Website
      plural: websites
      location: github.com/myorg/apis/website/v1alpha1
    reconciler:
      default: true
      hooks:
        location: github.com/myorg/my-operator/pkg/hooks
        function: WebsiteHooks
```

### Step 3: Generate Runtime Wiring

```bash
ork generate runtime --katalog katalogs/website.yaml
go mod tidy
```

### Step 4: Run Orkestra

```bash
go run ./cmd/orkestra/ run --katalog katalogs/website.yaml
```

Now every `Website` CR will be validated against the external API before reconciliation.

---

## Accessing Kubernetes Resources in Hooks

Hooks have access to the Kubernetes client through the context:

```go
func WebsiteHooks() domain.ReconcileHooks[domain.Object] {
    return domain.ReconcileHooks[domain.Object]{
        OnReconcile: func(ctx context.Context, obj domain.Object) error {
            kube, ok := kubeclient.FromContext(ctx)
            if !ok {
                return fmt.Errorf("kubeclient not found in context")
            }

            website := obj.(*websitev1.Website)

            // List all Secrets in the namespace
            secrets, err := kube.Clientset().CoreV1().Secrets(website.Namespace).List(ctx, metav1.ListOptions{})
            if err != nil {
                return err
            }

            // Use the list...
            return nil
        },
    }
}
```

---

## Accessing the Template Resolver

You can also use the template resolver to evaluate expressions:

```go
func WebsiteHooks() domain.ReconcileHooks[domain.Object] {
    return domain.ReconcileHooks[domain.Object]{
        OnReconcile: func(ctx context.Context, obj domain.Object) error {
            resolver, err := orktmpl.NewResolver(ctx, obj)
            if err != nil {
                return err
            }

            // Resolve a template expression
            image, err := resolver.Resolve("{{ .spec.image }}")
            if err != nil {
                return err
            }

            // Use the resolved value...
            return nil
        },
    }
}
```

---

## Hook Execution Order

When both templates and hooks are defined:

1. **Finalizers are added**
2. **Hooks run** (if defined)
3. **Templates run** (if hooks succeed)

This allows hooks to validate or modify the object before templates create resources.

---

## Debugging Hooks

Use structured logging to debug hooks:

```go
func WebsiteHooks() domain.ReconcileHooks[domain.Object] {
    return domain.ReconcileHooks[domain.Object]{
        OnReconcile: func(ctx context.Context, obj domain.Object) error {
            logger.FromContext(ctx).Info().
                Str("hook", "WebsiteHooks").
                Str("name", obj.GetName()).
                Msg("running OnReconcile hook")

            // Your logic here...
            return nil
        },
    }
}
```

---

## Next Steps

Hooks give you unlimited flexibility. When you need complete control over reconciliation, you can write a **custom reconciler**.

👉 [Writing Custom Reconcilers →](./writing-custom-reconcilers.md)


---

