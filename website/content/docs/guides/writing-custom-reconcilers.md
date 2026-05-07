---
title: "Writing Custom Reconcilers"
weight: 6
description: "The `GenericReconciler` handles most use cases with templates and hooks. But sometimes you need complete control over th..."
---

The `GenericReconciler` handles most use cases with templates and hooks. But sometimes you need complete control over the reconciliation loop. That's when you write a **custom reconciler**.

---

## When to Use a Custom Reconciler

| Feature | GenericReconciler | Custom Reconciler |
|---------|-------------------|-------------------|
| Create Deployments | ✅ | ✅ |
| Create Services | ✅ | ✅ |
| Drift correction | ✅ | You implement |
| Finalizers | ✅ | You implement |
| Events | ✅ | You implement |
| Metrics | ✅ | You implement |
| External API calls | ✅ (via hooks) | ✅ (direct) |
| Complex state machines | ❌ | ✅ |
| Custom status updates | ❌ | ✅ |
| Performance tuning | ❌ | ✅ |

Use a custom reconciler when:

- You need a complex state machine (e.g., provisioning → ready → degraded)
- You need to update status frequently
- You need to orchestrate multiple external resources
- You need to implement custom retry logic

---

## The Reconciler Interface

A custom reconciler must implement one method:

```go
type Reconciler interface {
    Reconcile(ctx context.Context, key string) error
}
```

That's it. Orkestra handles everything else: informers, queues, workers, metrics, and health tracking.

---

## Scaffolding a Custom Reconciler

```bash
# Scaffold a typed project
ork init my-operator --typed
cd my-operator
```

Create your reconciler in `pkg/reconciler/website.go`:

```go
package reconciler

import (
    "context"
    "fmt"

    "github.com/orkspace/orkestra/domain"
    "github.com/orkspace/orkestra/pkg/event"
    "github.com/orkspace/orkestra/pkg/kubeclient"
    "github.com/orkspace/orkestra/pkg/logger"
    websitev1 "github.com/myorg/apis/website/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/tools/cache"
)

type WebsiteReconciler struct {
    informer cache.SharedIndexInformer
    kube     *kubeclient.Kubeclient
    event    *event.Event
}

func NewWebsiteReconciler(
    informer cache.SharedIndexInformer,
    kube *kubeclient.Kubeclient,
    ev *event.Event,
) *WebsiteReconciler {
    return &WebsiteReconciler{
        informer: informer,
        kube:     kube,
        event:    ev,
    }
}

func (r *WebsiteReconciler) Reconcile(ctx context.Context, key string) error {
    // Parse the key (namespace/name)
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        return fmt.Errorf("invalid key: %w", err)
    }

    // Get the object from the informer cache
    obj, exists, err := r.informer.GetIndexer().GetByKey(key)
    if err != nil {
        return fmt.Errorf("failed to get object: %w", err)
    }

    if !exists {
        // Object was deleted — clean up external resources
        return r.cleanup(ctx, namespace, name)
    }

    website := obj.(*websitev1.Website)

    // Check if the object is being deleted
    if website.DeletionTimestamp != nil {
        return r.handleDeletion(ctx, website)
    }

    // Ensure finalizers
    if err := r.ensureFinalizers(ctx, website); err != nil {
        return err
    }

    // Reconcile the desired state
    return r.reconcile(ctx, website)
}
```

---

## Implementing Reconcile Logic

Here's a complete implementation that creates a Deployment:

```go
func (r *WebsiteReconciler) reconcile(ctx context.Context, website *websitev1.Website) error {
    logger.FromContext(ctx).Info().
        Str("website", website.Name).
        Msg("reconciling website")

    // Create or update Deployment
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      website.Name,
            Namespace: website.Namespace,
            OwnerReferences: []metav1.OwnerReference{
                *metav1.NewControllerRef(website, websitev1.SchemeGroupVersion.WithKind("Website")),
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &website.Spec.Replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": website.Name,
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app": website.Name,
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "app",
                            Image: website.Spec.Image,
                            Ports: []corev1.ContainerPort{
                                {ContainerPort: website.Spec.Port},
                            },
                        },
                    },
                },
            },
        },
    }

    _, err := r.kube.Clientset().AppsV1().Deployments(website.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
    if apierrors.IsAlreadyExists(err) {
        // Update if needed
        _, err = r.kube.Clientset().AppsV1().Deployments(website.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
        if err != nil {
            return err
        }
    } else if err != nil {
        return err
    }

    // Update status
    website.Status.Phase = "Ready"
    _, err = r.kube.DynamicClient().Resource(websitev1.GroupVersionResource).Namespace(website.Namespace).UpdateStatus(ctx, website, metav1.UpdateOptions{})
    if err != nil {
        return err
    }

    // Emit event
    r.event.Recorder().Eventf(website, corev1.EventTypeNormal, "Reconciled",
        "Successfully reconciled website %s", website.Name)

    return nil
}
```

---

## Managing Finalizers

Finalizers protect your CR from being deleted until cleanup completes:

```go
func (r *WebsiteReconciler) ensureFinalizers(ctx context.Context, website *websitev1.Website) error {
    finalizer := "finalizer.demo.orkestra.io/website"

    for _, f := range website.Finalizers {
        if f == finalizer {
            return nil // Already has it
        }
    }

    // Add finalizer
    website.Finalizers = append(website.Finalizers, finalizer)
    _, err := r.kube.DynamicClient().Resource(websitev1.GroupVersionResource).Namespace(website.Namespace).Update(ctx, website, metav1.UpdateOptions{})
    return err
}
```

---

## Handling Deletion

When a CR is deleted, clean up external resources before removing finalizers:

```go
func (r *WebsiteReconciler) handleDeletion(ctx context.Context, website *websitev1.Website) error {
    logger.FromContext(ctx).Info().
        Str("website", website.Name).
        Msg("handling deletion")

    // Clean up external resources
    // e.g., delete cloud load balancer, deregister domain, etc.

    // Remove finalizer to allow deletion to proceed
    finalizer := "finalizer.demo.orkestra.io/website"
    newFinalizers := []string{}
    for _, f := range website.Finalizers {
        if f != finalizer {
            newFinalizers = append(newFinalizers, f)
        }
    }
    website.Finalizers = newFinalizers

    _, err := r.kube.DynamicClient().Resource(websitev1.GroupVersionResource).Namespace(website.Namespace).Update(ctx, website, metav1.UpdateOptions{})
    return err
}
```

---

## Registering a Custom Reconciler

In your Katalog, set `reconciler.default: false` and provide the constructor:

```yaml
crds:
  - name: website
    apiTypes:
      group: demo.orkestra.io
      version: v1alpha1
      kind: Website
      plural: websites
      location: github.com/myorg/apis/website/v1alpha1
    operatorBox:
      default: false
      constructor:
        location: github.com/myorg/my-operator/pkg/reconciler
        function: NewWebsiteReconciler
```

Then generate the runtime registry:

```bash
ork generate registry --file katalogs/website.yaml
go mod tidy
```

---

## Next Steps

Now that you know how to write custom reconcilers, learn how to test your operators.

👉 [Testing Operators →](./testing-operators.md)


---
