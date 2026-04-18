# How Orkestra Works — End to End

This is the complete picture. From a Katalog YAML file to a CR being applied
to the cluster and resources appearing — every step, every komponent, every
decision point.

---

## The Katalog is the operator

A Katalog is a YAML file that declares everything an operator does:

```yaml
apiVersion: orkestra.konductor.io/v1Alpha
kind: Katalog
metadata:
  name: platform
spec:
  crds:
    application:
      apiTypes:
        group: platform.io
        version: v1alpha1
        kind: Application
        plural: applications
      workers: 10
      resync: 30s
      dependsOn:
        database: healthy    # workers don't start until database CRD is healthy

      validation:
        rules:
          - field: spec.image
            operator: exists
            action: deny

      mutation:
        rules:
          - field: spec.replicas
            default: 2

      operatorBox:
        default: true   # use Orkestra's GenericReconciler, no Go required
        cross:
          - crd: database
            selector:
              name: "{{ .metadata.name }}"
            as: db
        onCreate:
          external:
            - name: registry-check
              url: "{{ .spec.registryUrl }}/v2/{{ .spec.image }}/manifests/latest"
              expectedStatus: 200
              continueOnError: false
          deployments:
            - name: "{{ .metadata.name }}"
              image: "{{ .spec.image }}"
              when:
                - field: external.registry-check.status
                  equals: "200"
                - field: cross.db.status.phase
                  equals: "Ready"
          secrets:
            - name: "{{ .metadata.name }}-credentials"
              once: true
              data:
                password: "{{ randomAlphanumeric 32 }}"
        status:
          fields:
            - path: phase
              value: "Ready"
              when:
                - field: children.deployment.status.readyReplicas
                  greaterThan: "0"
```

The Katalog replaces: Go types, controller-gen, the reconciler, the RBAC,
the Deployment YAML, the Secret management — all of it.

---

## Startup sequence

When the operator binary starts, `konstructOrkestra` assembles the runtime.
Nothing connects to the cluster yet. Everything is closures and pointers.

```
main()
  └── konstructOrkestra(kfg, merger, ctx)
        │
        ├── 1. katalog.NewKatalog(merger, kfg)
        │         Parses YAML → validates CRD entries → builds dependency graph
        │         kat.Enabled() → map[string]CRDEntry (name → entry)
        │
        ├── 2. katalog.NewSchemeRegistry(kat)
        │         Registers Go types with the runtime scheme
        │         Dynamic CRDs (unstructured: true) skip this
        │
        ├── 3. kubeclient.NewKubeclient(kfg, scheme)
        │         REST config from in-cluster or kubeconfig
        │         Dynamic client for watch/list without Go types
        │         Typed clientset for core resources (Secrets, Events)
        │         Started immediately (informer factory needs REST config)
        │
        ├── 4a. ClientProvider
        │         Associates each CRD with a REST client constructor
        │         Constructor is deferred — called on first informer use
        │
        ├── 4b. SharedInformerFactory
        │         One SharedIndexInformer per CRD
        │         Not started yet — Start() called later by orkestra
        │
        ├── 4c. ProviderRegistry (loadProviders)
        │         Registers aws:, mongodb: providers
        │         Non-fatal — missing credentials log warning, operator starts
        │         Must be before factory loop — closures capture the reference
        │
        ├── 4d. ktrlRegistry (ResourceKatalog) + per-CRD factory closures
        │         For each CRD in kat.Enabled():
        │           - create queue entry in queueRegistry
        │           - create SharedIndexInformer (typed or dynamic)
        │           - build reconciler factory closure capturing:
        │               crdInfo, inf, ev, kube, hooks, newObj,
        │               providerRegistry, ktrlRegistry
        │           - ktrlRegistry.Register(gvk, crd, inf, factory)
        │
        ├── 5a. CRDHealth map
        │         One *CRDHealth per CRD — shared by reconciler and HTTP routes
        │
        ├── 5b. HTTP routes
        │         /katalog/{crd}/health, /katalog/{crd}, /katalog/{crd}/cr, ...
        │         Registered on hs.mux before Start() binds the port
        │
        ├── 6. DependencyKordinator
        │         Holds the dependency graph
        │         Does not start workers yet — waits for orkestra.Start()
        │
        └── 7+8. Orkestra supervisor
                  Registers komponents in start order
                  Returns — main.go calls orkestra.Start(ctx)
```

---

## Start sequence

`orkestra.Start(ctx)` calls `komponent.Start()` in registration order:

```
1. HealthServer.Start()      → binds port, /readyz returns 200
2. Kubeclient.Start()        → (already started, no-op)
3. Event.Start()             → connects recorder to clientset
4. QueueRegistry.Start()     → initialises per-CRD queues
5. DefaultWorkqueue.Start()  → initialises shared fallback queue
6. InformerFactory.Start()   → starts all SharedIndexInformers
   │                            each informer does:
   │                              LIST all existing CRs (initial sync)
   │                              WATCH for changes
   │                            closes "synced" channel when cache is populated
   │
7. DependencyKordinator.Start()
      reads dependency graph → topological order
      for each CRD in topo order:
        wait until dependsOn CRDs meet their condition
        wait for this CRD's informer cache to sync
        for i := 0; i < crd.Workers; i++:
          go startWorker(ctx, gvk, factory())
```

---

## The reconcile loop

Once workers are running, each worker processes items from its queue:

```
worker goroutine (one per worker slot):
  loop:
    key, shutdown = queue.GetWithContext(ctx)
    if shutdown: return

    if key == "__drain_sentinel__": continue  // deactivation drain

    err = reconciler.Reconcile(ctx, key)

    if err != nil:
      queue.AddRateLimited(key)   // retry with backoff
      health.RecordFailure(gvk)
    else:
      queue.Done(key)
      health.RecordSuccess(gvk)
```

Inside `GenericReconciler.Reconcile(ctx, key)`:

```
1. Split key → namespace/name
2. informer.GetIndexer().GetByKey(key)
   └── In-memory hash map lookup. Zero API calls.
       If not found: call hooks.OnNotFound, return nil (item deleted)

3. obj.GetDeletionTimestamp() != nil?
   └── Yes: handleDeletion
         ├── hooks.OnDelete (Go hook) or runTemplateOnDelete (declarative)
         │     └── onDelete jobs, provider cleanup, ordered deletion
         └── removeFinalizers → PATCH obj

4. ensureFinalizers → PATCH if needed
   ensureManagedLabel → PATCH if needed
   ensureManagedAnnotations → PATCH if needed

5. reconcileImpl
   ├── mutation (apply defaults) if MutateFirst
   ├── validation (warn: log+event, deny: return error)
   ├── mutation (apply defaults) if not MutateFirst
   │
   ├── hooks.OnReconcile (Go hook)?
   │     └── Yes: call it, skip declarative path
   │
   └── runTemplateReconcile (declarative path)
         │
         ├── Step 1: NewResolver(ctx, obj)
         │     data = {spec: {...}, status: {...}, metadata: {...}}
         │     r.data is the template context
         │
         ├── Step 2: readCross(rc.Cross, resolver)
         │     for each cross: declaration:
         │       katalogRegistry.GetInformerByName(decl.Kind)
         │         → informer.GetIndexer().GetByKey(ns/name)
         │         → ReadCrossFromInformer → {found, spec, status, labels}
         │       OR fetchCrossViaHTTP (cross-binary/cluster fallback)
         │     resolver = resolver.WithCross(crossData)
         │     .cross.database.status.phase now available in all expressions
         ├── Step 3: runGit(rc.OnReconcile.Git, resolver)
         │     If a git: block is declared in the Katalog:
         │       - resolve repo, branch, and path templates
         │       - clone or fetch the repository into the working directory
         │       - compute the current commit hash
         │       - detect whether the commit changed since the last reconcile
         │       - record metrics (orkestra_git_operations_total, duration, errors)
         │       resolver = resolver.WithGit({
         │         commit: "<hash>",
         │         changed: "true|false",
         │         path: "<workingDir>",
         │         error: "<msg>",
         │         called: "true",
         │       })
         │     .git.commit, .git.changed, .git.path, .git.error now available
         │     All subsequent template expressions and when: conditions can use:
         │       {{ .git.commit }}, {{ .git.changed }}, {{ .git.path }}
         ├── Step 4: runDocker(rc.OnReconcile.Docker, resolver)
         │     If a docker: block is declared in the Katalog:
         │       - resolve image, workingDirectory, and dockerfile templates
         │       - perform docker build
         │       - perform docker push (if push: true)
         │       - record metrics (orkestra_docker_operations_total, duration, errors)
         │       resolver = resolver.WithDocker({
         │         image: "<registry/repo:tag>",
         │         buildSucceeded: "true|false",
         │         error: "<msg>",
         │         called: "true",
         │       })
         │     .docker.image, .docker.buildSucceeded, .docker.error now available
         │     All subsequent template expressions and when: conditions can use:
         │       {{ .docker.image }}, {{ .docker.buildSucceeded }}
         │
         ├── Step 3: runExternal(rc.OnReconcile.External, resolver)
         │     for each external: call (sequential):
         │       evaluate when: conditions (EvaluateWhen)
         │       if skipped: inject {called: "false"}
         │       else: http.Do(req) with timeout
         │              inject {status, body, error, called: "true"}
         │       resolver = resolver.WithExternal(results)
         │     .external.registry-check.status now available
         │
         ├── Step 4: expandForEach* for each resource type
         │     if forEach declared: resolve list field → N copies with .item
         │     if no forEach: pass through unchanged (fast path)
         │
         ├── Step 5: runResourceGroup(OnCreate, update=false)
         │     for each resource type:
         │       runDeployments, runServices, runSecrets, runConfigMaps,
         │       runServiceAccounts, runJobs, runCronJobs
         │     Each resource function:
         │       EvaluateWhen(data, src.Conditions, src.AnyOf)
         │         → AND conditions (when:) AND OR conditions (anyOf:)
         │       if not passed: skip (or DeleteIfOwned if was created before)
         │       resolve templates: resolver.Resolve(src.Image) etc.
         │       if Secret with once: true: check existence first, skip if exists
         │       Create/Update resource in cluster
         │
         ├── Step 6: runResourceGroup(OnReconcile, update=true)
         │     Same as Step 5 but update=true → drift correction
         │
         └── Step 7: runProviders(rc.ProviderBlocks, resolver)
               for each provider block:
                 registry.Get(blockName)
                 resolveProviderBlock → evaluate all field templates
                 filterProviderDeclarations → evaluate when: conditions
                 provider.Reconcile(ctx, req)
                 // aws: → S3/RDS/Route53 via AWS SDK v2
                 // mongodb: → database/user/collection via mongo driver

6. patchStatusWithChildren(ctx, obj, reconcileErr)
   ├── ReadChildren
   │     for each known GVR (deployment, service, job, ...):
   │       kube.DynamicClient().Resource(gvr).Namespace(ns).List(
   │         ctx, metav1.ListOptions{
   │           LabelSelector: "orkestra-owner="+obj.GetName(),
   │           ResourceVersion: "0",   // watch cache, not etcd
   │           Limit: 1,
   │         })
   │     all 7 GVR queries run in PARALLEL with 3-second deadline
   │     → children map: {deployment: {name, namespace, kind, status, ready}}
   │
   ├── resolver = resolver.WithChildren(children)
   │     .children.deployment.status.readyReplicas now available
   │
   ├── resolveStatusFields(resolver.Data(), rc.Status.Fields)
   │     for each status field:
   │       EvaluateWhen(data, field.When, field.AnyOf)
   │       if passed: resolver.Resolve(field.Value)
   │       setNestedStatus(result, field.Path, value)
   │     last-writer-wins: terminal states declared last win
   │
   ├── set Ready condition:
   │     True if reconcileErr == nil
   │     False if reconcileErr != nil (message = error)
   │
   └── kube.PatchStatus(ctx, obj, statusPatch)
         HTTP PATCH /status
         Only this call touches etcd for status
```

---

## Watch events → reconcile triggers

The informer factory routes API server events to workqueues:

```
API server watch event (Added/Modified/Deleted for any CR)
    │
    ▼
informer.handleEvent(obj)
    │
    ├── namespace/name → queue key
    ├── find queue for this GVK from queueRegistry
    └── queue.AddRateLimited(key)
            │
            └── worker.GetWithContext() unblocks
                    └── reconciler.Reconcile(ctx, key)
```

The queue deduplicates: if a CR is modified 10 times in 100ms, only one
reconcile fires. The reconciler reads the current state — it does not see
the events, only the current object. This is level-triggered reconciliation.

Resync fires every `resync` duration (default 15s). It re-enqueues all CRs
even if nothing changed. This is how drift is detected and corrected.

---

## The Control Center

The Control Center reads from the same informer caches the reconciler uses.
This means the UI is always consistent with what the reconciler sees — no
extra API calls, no caching layer, no eventual consistency lag beyond the
informer's own lag (typically < 1 second).

```
Browser: GET /controlcenter/katalog/pipeline/cr/default/build-and-test
    │
    ▼
controlcenter.handleCRDetail(w, r)
    │
    ├── url path → crdName="pipeline", namespace="default", name="build-and-test"
    ├── http.Get(orkestra:8080/katalog/pipeline/cr/default/build-and-test)
    │         ── cr_handlers.go: BuildCRDetailAndEventsHandler
    │               ├── informer.GetIndexer().GetByKey("default/build-and-test")
    │               │         ← in-memory, zero API calls, <1ms
    │               │
    │               ├── readChildrenForEndpoint (parallel, 3s deadline, RV="0")
    │               │         ← 7 GVR queries, watch cache not etcd
    │               │
    │               └── fetchCREvents (field selector, RV="0")
    │
    └── template.Execute(cr_detail.html, viewData)
              Status fields, Conditions table, Child resource cards, Events table
              auto-refresh every 10 seconds via <meta http-equiv="refresh">
```

---

## Cross-CRD observation — how it's zero API calls

When Application's reconciler runs and needs to know if its Database CR is Ready:

```
Application operatorBox: readCross(decls=[{kind: "database", as: "db"}], ...)
    │
    ├── katalogRegistry.GetInformerByName("database")
    │         ← ktrlRegistry lookup by lowercase name
    │         returns the *same* SharedIndexInformer the database reconciler uses
    │         This informer is always running, cache always populated
    │
    ├── inf.GetIndexer().GetByKey("default/my-app")
    │         ← in-memory hash map lookup
    │         returns the full unstructured Database CR object
    │         zero API calls
    │
    └── ReadCrossFromInformer → {found: "true", status: {phase: "Ready", endpoint: "..."}}
          resolver = resolver.WithCross({db: {found, status, spec, labels}})

Template engine: "{{ .cross.db.status.phase }}"  → "Ready"
when: condition: cross.db.status.phase equals "Ready"  → true
Deployment is created.
```

The Application worker and the Database worker are separate goroutines reading
from separate queues, but they share the same in-memory informer cache. No
coordination needed. No locks. No API calls. One binary, one cache per CRD,
zero coupling.

---

## Secret generation with once: — why it's correct

```
First reconcile:
  runSecrets: src.Once = true, update = false
    resolver.Resolve(src.Name) → "my-app-credentials"
    secretExists(ctx, kube, "default", "my-app-credentials")
      → kube.Clientset().CoreV1().Secrets("default").Get(ctx, name, RV="0")
      → 404 Not Found → false
    resolver.ResolveSecretTemplate(src)
      → note.Map()["randomAlphanumeric"](32) → "k7Xm3pQs9vR2nTwY8cL1..."
      → resolved.Data = {password: "k7Xm3pQs9vR2nTwY8cL1..."}
    orksecrets.Create(ctx, kube, owner, spec)
      → Secret created in cluster

Every subsequent reconcile:
  runSecrets: src.Once = true, update = false
    secretExists → 200 OK → true
    continue  ← randomAlphanumeric is NEVER called again
    Password unchanged. Application not broken.
```

---

## forEach expansion — how N deployments appear

```yaml
Katalog declaration:
  deployments:
    - name: "{{ .metadata.name }}-{{ .item }}"
      image: "{{ .spec.image }}"
      forEach:
        field: spec.regions
        as: item
```

CR spec.regions: ["us-east-1", "eu-west-1", "ap-southeast-1"]

```go
expandForEachDeployments(resolver, srcs):
  src.ForEach != nil
  items = resolveListField(resolver.Data(), "spec.regions")
        = ["us-east-1", "eu-west-1", "ap-southeast-1"]

  for i, item in items:
    itemResolver = resolver.WithItem(item, "item", i)
    expanded.Name = itemResolver.Resolve("{{ .metadata.name }}-{{ .item }}")
                  = "my-app-us-east-1"
    append to result
```

runDeployments receives 3 DeploymentTemplateSources:
```json
  {Name: "my-app-us-east-1", Image: "nginx:latest"}
  {Name: "my-app-eu-west-1", Image: "nginx:latest"}
  {Name: "my-app-ap-southeast-1", Image: "nginx:latest"}
```

3 Deployments created. All owned by the CR. All labelled orkestra-owner=my-app.
