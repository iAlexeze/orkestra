# Generated files

`ork migrate -o ./my-operator` writes five files. All have `TODO(ork migrate):` markers where the tool could not infer values from the source.

## katalog.yaml

The constructor Katalog stub. Fields you must fill in:

```yaml
metadata:
  name: webapp-reconciler    # derived from receiver type
  author: myorg              # TODO: set your org

spec:
  crds:
    webapp-reconciler:
      apiTypes:
        group: TODO          # TODO: your CRD group (e.g. apps.myorg.io)
        version: v1alpha1
        kind: TODO           # TODO: your CRD kind (e.g. WebApp)
        plural: TODO
        object: TODO
        objectList: TODOList
        location: github.com/myorg/my-operator/api/v1alpha1  # adjust package path

      operatorBox:
        default: false       # constructor owns the full loop

        constructor:
          location: github.com/myorg/my-operator/controller  # adjust
          function: NewWebAppReconciler                       # derived from receiver type
          resources:
            - kind: TODO     # TODO: list the resource kinds you manage
```

## simulate.yaml

Simulation stub. Fill in the resource kinds your operator creates in cycle 1:

```yaml
spec:
  expect:
    ops:
      - cycle: 1
        verb: create
        resource: TODO  # e.g. deployments
```

Run `ork simulate` after filling this in — no cluster required.

## e2e.yaml

End-to-end test stub. Fill in CRD kind, CR name, and assertions:

```yaml
spec:
  expect:
    - name: Resources created
      after: cr-applied
      timeout: 90s
      resources:
        - kind: TODO   # e.g. Deployment
          name: TODO   # e.g. my-webapp
          namespace: default
          ready: true
```

Run `ork e2e` to execute against a real cluster (kind is created automatically).

## go.mod

Module file with Orkestra pinned to the CLI version that ran the migration:

```
module github.com/myorg/my-operator

go 1.22

require (
    github.com/orkspace/orkestra v0.8.1
    k8s.io/api v0.29.3
    k8s.io/apimachinery v0.29.3
    k8s.io/client-go v0.29.3
)
```

Run `go mod tidy` to resolve indirect dependencies.

## The rewritten reconciler

Same filename as the input. See [01-output.md](01-output.md) for what changed.
