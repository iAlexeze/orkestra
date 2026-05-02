---
title: "Quick Start"
weight: 2
description: "Build your first Kubernetes operator with Orkestra in 5 minutes."
---

In this guide you'll create a `Website` operator that manages `Deployment` and `Service` resources — the canonical "hello world" of Kubernetes operators.

## 1. Create a project directory

```bash
mkdir my-website-operator && cd my-website-operator
```

## 2. Write the Katalog manifest

Create `katalog.yaml`:

```yaml
apiVersion: orkestra.orkspace.io/v1
kind: Katalog
metadata:
  name: hello-website
  author: orkspace
  version: 0.1.0
  description: The simplest possible operator. One CRD, one Deployment.

spec:
  crds:
    website:
      apiTypes:
        group: demo.orkestra.io
        version: v1alpha1
        kind: Website
        plural: websites

      operatorBox:
        default: true
        onCreate:
          deployments:
            - image: "{{ .spec.image }}"
              # name defaults to: <cr-name>-deployment
              # namespace defaults to: <cr-namespace>
              # replicas defaults to: 1
```

## Apply the website CRD

```bash
kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: websites.apps.example.com
spec:
  group: apps.example.com
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                image:
                  type: string
                  replicas:
                  type: integer
                  default: 1
EOF
```

## 4. Start the operator

```bash
ork run --katalog ./katalog.yaml
```

Orkestra will:
1. Generate and register the `websites.apps.example.com` CRD
2. Start the reconciliation controller
3. Open Control Center at [http://localhost:8081](http://localhost:8081)

## 5. Create a Website resource

In a new terminal:

```bash
kubectl apply -f - <<EOF
apiVersion: apps.example.com/v1alpha1
kind: Website
metadata:
  name: my-site
  namespace: default
spec:
  image: nginx:1.25
  replicas: 2
EOF
```

## 6. Verify reconciliation

```bash
kubectl get websites
kubectl get deployment my-site
kubectl get service my-site-svc
```

You should see:

```
NAME      READY   AGE
my-site   True    12s
```

{{< callout type="tip" title="Control Center" >}}
Open [http://localhost:8081](http://localhost:8081) to see your `Website` resource in the Control Center UI with live status, events, and child resource details.
{{< /callout >}}

## Next steps

- Learn about [Katalog basics](/docs/basics/) — schemas, validation, and lifecycle hooks
- Explore [state machines](/docs/concepts/) — model complex lifecycles declaratively
- Try [Komposer](/docs/guides/) — compose from Helm charts and OCI registries
