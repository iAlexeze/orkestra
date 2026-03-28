# Basic Reconciliation

Now that you’ve written your first Katalog and Komposer, it’s time to see how Orkestra actually **reconciles** a Custom Resource (CR).  
This guide walks through the simplest possible reconciliation flow:

- A CRD  
- A CR  
- A minimal Katalog  
- A Komposer that loads it  
- Orkestra reconciling the CR into real Kubernetes resources  

!!! note
    This is the *first* time we introduce reconciliation.  
    Everything here is intentionally simple and fully declarative.

---

## Step 1 — Define a Simple CRD

Create a file called `myapp-crd.yaml`:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: myapps.demo.myorg.io
spec:
  group: demo.myorg.io
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
                port:
                  type: integer
  scope: Namespaced
  names:
    plural: myapps
    singular: myapp
    kind: MyApp
```

Apply it:

```
kubectl apply -f myapp-crd.yaml
```

!!! tip
    CRDs define the *shape* of your API.  
    Katalogs define the *behavior* of your API.

---

## Step 2 — Create a Simple CR

Create a file called `myapp.yaml`:

```yaml
apiVersion: demo.myorg.io/v1alpha1
kind: MyApp
metadata:
  name: demo
spec:
  image: nginx:1.27
  replicas: 2
  port: 80
```

Apply it:

```
kubectl apply -f myapp.yaml
```

At this point, nothing happens yet — Orkestra hasn’t been started.

!!! note
    Orkestra only reconciles CRs when the runtime is running and a Komposer has loaded your Katalog.

---

## Step 3 — Write a Minimal Katalog

Create `katalog.yaml`:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: myapp-katalog
spec:
  crds:
    - name: myapp
      apiTypes:
        group: demo.myorg.io
        version: v1alpha1
        kind: MyApp
        plural: myapps
      reconciler:
        default: true
        onCreate:
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              replicas: "{{ .spec.replicas }}"
              port: "{{ .spec.port }}"
              namespace: "{{ .metadata.namespace }}"
              reconcile: true
```

This tells Orkestra:

- Watch for `MyApp` CRs  
- Create a Deployment using values from the CR  
- Keep the Deployment in sync with the CR  

!!! tip
    This is the smallest meaningful operator you can build with Orkestra.

---

## Step 4 — Write a Komposer That Loads the Katalog

Create `komposer.yaml`:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Komposer
metadata:
  name: myapp-komposer

sources:
  files:
    - ./katalog.yaml
```

!!! note
    Komposers load katalogs.  
    Katalogs define reconciliation behavior.

---

## Step 5 — Start Orkestra

Run:

```
ork run --katalog katalog.yaml
```

Or, if using the Komposer:

```
ork run --katalog komposer.yaml
```

!!! note
    Both katalogs and komposers are passed using the `--katalog` flag.  
    This is by design, as a Komposer is simply a declarative bundle of katalogs.


Orkestra will:

1. Load your Katalog  
2. Register the CRD  
3. Start informers  
4. Watch for `MyApp` CRs  
5. Reconcile them into Deployments  

!!! caution
    If Orkestra cannot find your CRD, it will not start reconciliation.  
    Always apply the CRD before running the runtime.

---

## Step 6 — Observe Reconciliation

Check the Deployment:

```
kubectl get deployments
```

You should see:

```
NAME    READY   UP-TO-DATE   AVAILABLE   AGE
demo    2/2     2            2           5s
```

Check the health endpoint:

```
curl localhost:8080/katalog/myapp/health | jq
```

You’ll see:

- last reconcile time  
- number of reconciles  
- any errors  
- worker activity  

!!! tip
    The health endpoint is your best friend during development.  
    It shows exactly what Orkestra is doing.

---

## Step 7 — Update the CR

Edit the CR:

```yaml
spec:
  replicas: 4
```

Apply it:

```
kubectl apply -f myapp.yaml
```

Orkestra will:

- Detect the change  
- Reconcile the CR  
- Update the Deployment to 4 replicas  

Check:

```
kubectl get deploy demo
```

!!! note
    You did not write any code.  
    Orkestra handled the entire reconciliation loop for you.

---

## Step 8 — Delete the CR

```
kubectl delete -f myapp.yaml
```

Orkestra will:

- Remove the Deployment  
- Clean up resources  
- Finalize the CR  

!!! tip
    Deletion behavior can be customized later using `onDelete` blocks.

---

## Summary

You have now seen the full reconciliation lifecycle:

1. Define a CRD  
2. Create a CR  
3. Write a Katalog  
4. Load it with a Komposer  
5. Run Orkestra  
6. Watch reconciliation happen  
7. Update the CR and see drift correction  
8. Delete the CR and watch cleanup  

This is the foundation for everything else you will build with Orkestra.

---

## Next Steps

Continue with:

**Example Workflows ([Beginner](../examples/index.md))**  
Learn how to build multi‑resource operators, add drift correction, use dependencies, and structure real‑world katalogs.
