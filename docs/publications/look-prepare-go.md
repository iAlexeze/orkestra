# Look, Prepare, Go: Kubernetes for Everyone

*Orkestra Project — April 2026*

---

## Abstract

Kubernetes adoption among developers has stalled at the same boundary for a
decade: the gap between a working Dockerfile and a running production
deployment. Developers who can describe their application completely in a
Dockerfile and a `.env` file cannot translate that description into pod specs,
service definitions, ingress configurations, RBAC rules, HPA policies, and
PodDisruptionBudgets without acquiring a body of knowledge orthogonal to
their actual work. The standard responses — Helm charts, platform teams,
managed Kubernetes services — reduce the problem but do not eliminate it. They
move the Kubernetes knowledge requirement rather than removing it.

Orkestra eliminates it. A developer who has a Dockerfile and a `.env` already
has everything Orkestra needs to produce a production-grade Kubernetes
deployment. Three commands — `ork doctor`, `ork doctor init`, `ork deploy` —
convert that description into a running deployment with autoscaling, pod
disruption budgets, ingress routing, secrets management, and live observability.
The developer writes no Kubernetes YAML. The developer learns no Kubernetes
concepts. The deployment is production-grade from the first run.

---

## 1. The wall

The Kubernetes learning curve is not a curve. It is a wall. On one side:
a developer with a Dockerfile, a `.env`, and an application that works locally.
On the other side: a deployment running in a cluster, observable, scalable,
protected from disruption, accessible at a URL.

Between them:

A pod specification. A deployment manifest. A service definition — ClusterIP,
NodePort, or LoadBalancer, each with different routing behavior. An ingress
resource and the separate question of which ingress controller to install and
how. A namespace. A service account. A role and a role binding. Resource
requests and limits. Liveness and readiness probes. An HPA with target
utilization thresholds. A PodDisruptionBudget. Secret management — how to
encode values, how to reference them from pod specs, how to ensure they never
appear in source control. A Helm chart to package all of this so it can be
deployed repeatably.

None of these are intrinsically difficult. Each has documentation. The
difficulty is that there are so many of them, they depend on each other in
non-obvious ways, and none of them are related to what the developer is
actually building. They are infrastructure for the infrastructure.

The standard industry response is to hire someone who knows Kubernetes, or to
use a managed platform that abstracts the cluster. Both are reasonable for
teams with the budget and the requirement. Neither is accessible to an
individual developer, a small team deploying their first production service, or
a developer who wants to use their own cluster for cost, compliance, or
control reasons.

---

## 2. What developers already have

A developer with a production-ready application has, without knowing it,
already described that application completely enough for Kubernetes deployment.

**The Dockerfile** describes the build process, the runtime environment, and
implicitly the operating system dependencies. It is the complete specification
of what the running container needs to be. Kubernetes needs exactly this —
the image built from the Dockerfile is what runs in every pod.

**The `.env` file** describes the application's runtime configuration. Every
variable is either configuration (log level, environment name, connection pool
size) or a secret (database credentials, API keys, signing secrets). This is
exactly the information that Kubernetes Secrets and ConfigMaps hold. The
developer has already classified their variables into these categories by
virtue of knowing which ones they would share and which ones they would not.

**The Git repository** provides the version. The commit SHA is the natural
image tag — it is unique, traceable, and already meaningful to the developer
as the identifier of a specific state of the codebase.

**The port** is almost always in the `.env` as `PORT`, or can be inferred from
the framework. A Go application listens on 8080 by default. A Node.js
application listens on 3000. A Python application on 8000. The application
already knows where it listens.

The developer is not missing any information. They are missing the translation
layer that converts what they have into what Kubernetes expects.

---

## 3. The translation

Orkestra provides the translation in two steps.

**Look** — `ork doctor` examines the project directory and surfaces what it
found: the language, the port, the number of environment variables, whether
the project has a frontend that warrants an Ingress. It shows the developer
what Orkestra will create before creating anything. The developer sees their
own project described in Kubernetes terms without having to know those terms
to understand the description.

**Prepare** — `ork doctor init` generates three files. The first is
`.orkestra/katalog.yaml` — the Orkestra operator declaration that describes
the complete deployment topology. The second is `.orkestra/app.yaml` — a
ConfigMap containing the parameters the developer may want to adjust: port,
replica count, maximum replicas, the public hostname. The third is
`.orkestra/values.yaml` — Helm values for the Orkestra operator installation,
pre-populated with commented stubs for the Control Center ingress and runtime
resources. The Katalog is correct by default; `app.yaml` is the only file the
developer needs to read and possibly edit. It contains no Kubernetes
primitives. It contains only application parameters.

**Go** — `ork deploy` builds the Docker image tagged with the current commit
SHA, pushes it to the specified registry, generates the full deployment bundle
including RBAC and a Kubernetes Secret from the `.env` file's non-config
variables, applies everything to the cluster, and waits for the rollout to
complete. When the rollout succeeds, the deployment is summarized: the image,
the status, the internal service URLs for every project the Komposer knows
about, and a link to the Control Center. If `host` is set in `app.yaml`, the
public URL appears here too. The cluster now contains a Deployment, a Service,
an HPA, a PodDisruptionBudget, a Secret, a ConfigMap, and all the RBAC that
connects them — plus an Ingress if a hostname was declared. The developer
created none of these directly.

---

## 4. The .env contract

The translation of `.env` variables into Kubernetes resources requires one
decision from the developer: which variables are configuration and which are
secrets. This decision has already been made implicitly — it is the difference
between variables the developer would commit to a shared document and variables
they would not.

Orkestra formalizes this with a single inline comment: `# ork:cfg` on the same
line as a variable marks it as configuration. Variables without this tag become
Kubernetes Secrets. Variables with it become ConfigMap entries.

```bash
DATABASE_URL=postgres://user:pass@host/db   # → Secret
JWT_SECRET=abc123xyz                         # → Secret
STRIPE_KEY=sk_live_...                       # → Secret

PORT=8080           # ork:cfg → ConfigMap
LOG_LEVEL=info      # ork:cfg → ConfigMap
ENVIRONMENT=prod    # ork:cfg → ConfigMap
```

This is the only Orkestra-specific knowledge the developer needs. One comment
syntax, applied to variables they are already managing, in a file they already
have.

The generated Secret file — containing base64-encoded values suitable for
direct application to Kubernetes — is written to `.orkestra/bundle/`, which is
added to `.gitignore` automatically. The secret values never enter source
control. The developer does not manage this lifecycle; it is handled by the
generate step on every deployment.

---

## 5. Production grade by default

The deployment produced by `ork deploy` is not a minimal viable deployment
that the developer will need to enhance later. It is production-grade from the
first run.

**Autoscaling** is configured via HPA with a minimum of two replicas and a
maximum of ten, targeting 70% CPU utilization. The parameters are adjustable
in the ConfigMap; the mechanism is in place without any developer action.

**Disruption protection** is configured via PodDisruptionBudget with
`minAvailable: 1`. Rolling updates and cluster maintenance operations cannot
reduce the running replica count to zero. This is a property of the deployment
that most teams add only after experiencing an unplanned outage during a
cluster upgrade.

**Secrets management** is handled by the translation from `.env` to a
Kubernetes Secret. The running pods reference the secret via `envFrom`, which
means secret rotation requires only updating the Secret and triggering a
rolling restart — no image rebuild, no code change.

**Ingress routing** is configured when the developer has a frontend or sets a
`host` value in the ConfigMap. Orkestra detects whether an ingress controller
is running in the cluster; if not, it installs nginx-ingress automatically.
The developer never encounters "IngressClass not found" or "No address for
ingress" — these errors exist below the abstraction layer.

**Observability** is available from the first deployment. The Control Center
is accessible immediately after deploy — `ork deploy` prints the port-forward
command when no external hostname is configured:

```
kubectl port-forward svc/orkestra-cc 8081:8081 -n orkestra-system &
```

Opening `http://localhost:8081` shows the deployment's health, the CR's
status, its child resources, and the full event history from every reconcile.
No monitoring infrastructure required. The developer clicks on their Katalog,
selects their CR, and sees every resource Orkestra created, its current state,
and the timeline of events that brought it there. `kubectl logs` works
normally alongside this view.

**Rollback** is a single command. `ork deploy rollback` restores the previous
image. `ork deploy rollback --image <ref>` restores any specific image.
The state tracking in `~/.orkestra/deploy/state.json` maintains the previous
image reference across deploy runs. The rollback patches the ConfigMap, which
Orkestra reconciles into a new rolling update. The developer does not need to
understand rolling updates to use rollback.

---

## 6. Multiple projects, one cluster

A developer with multiple services — an API, a worker, a frontend — deploys
each independently from its own repository using the same three commands.
Each project has its own `.orkestra/katalog.yaml`, its own namespace, its own
autoscaling configuration. Orkestra manages all of them in one instance.

The global Komposer at `~/.orkestra/deploy/komposer.yaml` accumulates project
Katalogs as each project is deployed. When a new project is added, Orkestra
picks up the updated Komposer and begins managing the new CRD without
restarting. Existing deployments are unaffected.

Services communicate with each other via Kubernetes internal DNS:
`http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local`. Orkestra
prints this URL after every deploy alongside an `export` hint:

```
export MY_API_URL=http://my-api-orkestra-svc.my-api-orkestra-ns.svc.cluster.local:8080
```

The developer adds this to the other service's `.env` and redeploys. There is
no service mesh to configure. There is no load balancer to provision. The
cluster's DNS handles it.

---

## 7. The underlying architecture

The developer experience described in this paper is built on Orkestra's
existing declarative operator runtime. A ConfigMap is the CRD — the operator
watches a ConfigMap rather than a custom resource, which means no CRD
installation, no version management, no schema validation overhead.

The Katalog generated by `ork doctor init` uses the same primitives as any
other Orkestra operator: `onCreate` for namespace creation, `onReconcile` with
a `when:` gate for the Deployment and associated resources, `status.fields` for
the phase and URL. The developer who later wants to understand what Orkestra is
doing reads a Katalog that is identical in structure to any other Orkestra
operator. There is no separate abstraction, no different code path, no
developer-mode runtime.

This has a practical consequence: operators Orkestra manages for platform
engineers and operators generated by `ork doctor init` for developers are the
same kind of thing managed by the same runtime with the same observability and
the same failure model. A platform team that adopts Orkestra for their own
operator development and a developer who uses `ork deploy` are both users of
the same system, with the same guarantees, visible in the same Control Center.

---

## 8. What the developer does not learn

The developer who uses `ork doctor init` and `ork deploy` does not learn:

- What a pod spec is or how to write one
- The difference between a ClusterIP, NodePort, and LoadBalancer service
- How ingress controllers work or which one to choose
- How Kubernetes secrets are encoded or referenced from pod specs
- What a ServiceAccount is or why RBAC requires one
- What an HPA target utilization threshold means or how to calculate it
- What a PodDisruptionBudget is or when it applies
- How Helm charts are structured or how to write a values file
- How to write a Deployment rollout strategy
- What `kubectl rollout status` means or when to use it

They learn one thing: `# ork:cfg` marks a variable as configuration rather
than a secret. This is the total Kubernetes knowledge requirement for
production deployment with Orkestra.

The knowledge is not hidden or deferred. It is not required. The deployment is
correct without it. When the developer later wants to understand what is
running — because they are curious, because something needs debugging, because
they are growing into a platform engineering role — the Katalog, the Control
Center, and standard `kubectl` commands are all available and all consistent
with what they would learn from Kubernetes documentation. Nothing is obscured.
The abstraction is additive, not opaque.

---

## 9. Conclusion

Kubernetes operators for everyone has always been Orkestra's stated goal. The
operator runtime, the declarative layer, and the provider ecosystem address
that goal for platform engineers and DevOps practitioners. `ork doctor` and
`ork deploy` address it for the developer who has never heard the word operator
and does not need to.

The three commands — look, prepare, go — are not a simplification of
Kubernetes. They are a translation from the vocabulary developers already use
(Dockerfile, `.env`, git commit) into the vocabulary Kubernetes requires (pod
spec, Secret, ConfigMap, Deployment, HPA). The translation is correct, complete,
and production-grade. The developer provides the application; Orkestra provides
the infrastructure.

Kubernetes is not difficult because the concepts are hard. It is difficult
because there are many concepts and learning them is not the developer's job.
When the translation is handled by a tool, the difficulty disappears. What
remains is the developer's application, running in a cluster, scaled and
observed and protected, accessible at a URL that `ork open` retrieves from the
deployment status.

That is what Kubernetes for everyone means.