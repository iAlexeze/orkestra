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
apiVersion: orkestra.sh/v1
kind: Katalog
metadata:
  name: website-operator
  namespace: default
spec:
  crd:
    group: apps.example.com
    kind: Website
    version: v1alpha1
    scope: Namespaced
    description: "Manages a simple web application deployment"
  resources:
    - kind: Deployment
      name: "{{ .Name }}"
      namespace: "{{ .Namespace }}"
      template: ./templates/deployment.yaml
    - kind: Service
      name: "{{ .Name }}-svc"
      namespace: "{{ .Namespace }}"
      template: ./templates/service.yaml
  status:
    conditions:
      - type: Ready
        description: "Website is fully reconciled"
      - type: Degraded
        description: "Website reconciliation failed"
  observe:
    events: true
    controlCenter: true
```

## 3. Create resource templates

Create `templates/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "{{ .Name }}"
  namespace: "{{ .Namespace }}"
  labels:
    app: "{{ .Name }}"
    managed-by: orkestra
spec:
  replicas: {{ .Spec.replicas | default 1 }}
  selector:
    matchLabels:
      app: "{{ .Name }}"
  template:
    metadata:
      labels:
        app: "{{ .Name }}"
    spec:
      containers:
        - name: website
          image: "{{ .Spec.image }}"
          ports:
            - containerPort: 80
```

Create `templates/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: "{{ .Name }}-svc"
  namespace: "{{ .Namespace }}"
spec:
  selector:
    app: "{{ .Name }}"
  ports:
    - port: 80
      targetPort: 80
  type: ClusterIP
```

## 4. Start the operator

```bash
ork run --katalog ./katalog.yaml
```

Orkestra will:
1. Generate and register the `websites.apps.example.com` CRD
2. Start the reconciliation controller
3. Open Control Center at [http://localhost:8090](http://localhost:8090)

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
Open [http://localhost:8090](http://localhost:8090) to see your `Website` resource in the Control Center UI with live status, events, and child resource details.
{{< /callout >}}

## Next steps

- Learn about [Katalog basics](/docs/basics/) — schemas, validation, and lifecycle hooks
- Explore [state machines](/docs/concepts/) — model complex lifecycles declaratively
- Try [Komposer](/docs/guides/) — compose from Helm charts and OCI registries
