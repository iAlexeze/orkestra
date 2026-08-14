# What If the Next Ops Is IntentOps?

*Every generation of ops shifted what the primary artifact was. We might be due for another shift.*

---

## A short history of shifting artifacts

Every major ops movement of the last twenty years can be described by what it made the primary artifact — the thing that everything else was derived from, the thing you versioned, the thing you trusted.

**DevOps** made the pipeline the artifact. Automate the process. If the pipeline is right, the deployment is right. The artifact was the `.yaml` file describing the CI/CD steps, the shell scripts, the Makefiles. You shipped the automation, not just the code.

**GitOps** made the Git repository the artifact. The desired state of the system lived in Git. The cluster converged toward it. Flux and ArgoCD watched for drift and corrected it. The repository was the source of truth — not the cluster, not the pipeline, the repository.

**PlatformOps** made the platform itself the artifact. Internal developer platforms, service catalogs, self-service portals. The platform team built the thing that other teams used to deploy things. The artifact was the platform — Backstage plugins, Port integrations, Terraform modules, custom webhooks — and its job was to give developers a simpler interface than raw Kubernetes.

Each movement answered a real problem. Each one created a new problem by solving the previous one.

GitOps solved "the cluster is the source of truth" but created "the repository is full of Kubernetes YAML that developers don't understand." PlatformOps solved the developer experience problem but created "someone has to build and maintain the portal, the plugin, the form, and keep it in sync with the actual infrastructure." Every generation of ops made things better for one audience and harder for another.

---

## The problem none of them solved

Every ops movement has treated the Kubernetes resource as the fundamental unit. The manifest. The CR. The object with `apiVersion`, `kind`, `metadata`, `spec`.

GitOps delivers manifests from Git. PlatformOps builds forms that construct manifests. CI pipelines apply manifests. The manifest is where everything converges — the translation target that every tool, every workflow, every developer ultimately has to produce.

The manifest is a Kubernetes concept. It reflects how Kubernetes stores state, not how humans think about intent.

A developer thinking about deploying an application thinks: *repository, environment, team, replicas.* They do not think: *apiVersion: platform.myorg.io/v1, kind: App, metadata.labels.team: payments, spec.deployment.replicas: 2.* The manifest is the Kubernetes representation of their intent. It is not the intent itself.

Every ops movement has solved the problem of getting the manifest to the cluster. None of them solved the problem of the manifest being the wrong artifact in the first place.

---

## What IntentOps would mean

IntentOps is a different premise. The primary artifact is not the manifest. It is the intent.

Intent is what the person or system actually means to express — in their own vocabulary, at their own level of abstraction, without reference to how Kubernetes stores it.

```yaml
target: app
repository: myorg/payments-api
environment: staging
team: payments
replicas: 2
```

This is an intent file. It has no `apiVersion`. No `kind`. No `spec`. No `metadata`. It expresses what the developer wants. It does not express how Kubernetes represents what the developer wants.

In an IntentOps model:

- **The intent is versioned** — in a file, in a Git repository, reviewed in pull requests, recoverable by commit SHA
- **The intent is the stable contract** — field names that reflect the developer's vocabulary, not the CRD's structure
- **The delivery surface translates** — reads the intent, maps it to the current Kubernetes representation, applies it
- **The cluster is an implementation detail** — the intent is the thing you care about; how the cluster stores it is the translation layer's concern

When the CRD schema changes — when a field moves, when a structure deepens — the intent file does not change. The translation declaration changes. The callers notice nothing.

This is not a small shift. It inverts the relationship between intent and infrastructure that every ops movement has taken for granted. Instead of asking "how do we get this manifest to the cluster efficiently," it asks "how do we keep the manifest from being the thing people have to think about at all."

---

## The delivery surface

IntentOps requires a delivery surface — the layer between intent and infrastructure that does the translation, validates the intent, stamps the provenance, and delivers to the cluster.

The delivery surface has to answer several questions that current tools leave to callers:

**Where does this intent go?** The surface resolves the target — which CRD, which cluster, which operator — from the intent's vocabulary without requiring the caller to know.

**What does it become?** The surface translates the intent's flat vocabulary into the CRD's nested structure. `"cpu": "500m"` becomes `spec.app.resources.cpu`. A cron string `"0 2 * * 1-5"` fans out into five structured schedule fields. The caller's vocabulary and the CRD's structure never have to match.

**Is it valid?** The surface validates against admission rules that can reason about context — the time of day, external metrics, the caller's identity, cross-field dependencies — not just static field predicates.

**Who sent it?** The surface stamps provenance — which target, which alias, which source, cryptographically verified — so every downstream system knows how the intent arrived without a separate audit log.

**Where does it go next?** In a multi-cluster world, the delivery surface routes intent to the right cluster based on whatever the platform team declares — environment, region, alias, token identity.

None of this is possible when the manifest is the artifact. It becomes possible when intent is the artifact and the surface does the translation.

---

## The sources

IntentOps also changes the relationship between delivery and source. When intent is the artifact, the source becomes irrelevant to the runtime.

A developer opens a browser form. A CI pipeline runs a curl command. A GitHub push event fires a webhook. A Slack command is typed. A PagerDuty alert triggers a webhook. A cron job runs at 2am.

In every case, the delivery surface receives intent — in whatever form the source produces it — translates it, and delivers it. The runtime reconciling behind the delivery surface doesn't know or care where the intent came from. The operator sees a CR. The CR carries provenance annotations. The notes can reason about the source if behavior should vary. But the fundamental reconciliation loop is source-agnostic.

This is qualitatively different from GitOps, where the source — the Git repository — is load-bearing. The GitOps tool watches the repository. The repository is the source of truth. Remove the repository and the model breaks. In IntentOps, the source is just a delivery mechanism. The intent is the source of truth. The delivery surface can receive it from anywhere.

---

## The provenance dimension

IntentOps adds something GitOps never had: provenance at the intent level.

In GitOps, you know what commit caused a deployment. You know the author, the timestamp, the commit message. But you don't know the intent — you know the manifest that was applied. If the CRD schema has changed since the manifest was written, the manifest might not reflect what the author intended anymore.

In IntentOps, the intent is preserved. It is stamped on the CR as an annotation. The delivery surface records which target was used, which alias, which source. If the source was GitHub Actions, the verified OIDC sub claim is stamped — cryptographic proof of which workflow, which repository, which branch delivered this intent.

The operator can read this at reconcile time. The admission layer can gate on it. The status layer can surface it. The compliance layer can assert on it. Every CR carries not just what was intended but the full context in which the intent was expressed — verified, immutable, readable by the operator without any additional infrastructure.

---

## Why now

The reason IntentOps hasn't existed before is not that nobody wanted it. It's that it requires owning enough of the stack to make it work.

GitOps tools own the delivery layer. They don't own the admission layer or the reconciliation layer. Admission webhook frameworks own the validation layer. They don't own the delivery layer or the reconciliation layer. Operator frameworks own the reconciliation layer. They don't own the delivery layer or the admission layer.

IntentOps requires owning all three — delivery, admission, and reconciliation — under a unified model so that intent can flow from source to cluster without losing context at each boundary. When delivery, admission, and reconciliation are separate tools, provenance gets lost at every handoff. When they share a model, provenance travels end to end inside the CR itself.

This is why IntentOps has been hard to build even though the need has been obvious. It's not a new feature for an existing tool. It's a new layer that sits across all of them.

---

## The shift

DevOps shifted the artifact from code to pipeline.
GitOps shifted the artifact from pipeline to repository.
PlatformOps shifted the artifact from repository to platform.
IntentOps shifts the artifact from platform to intent.

Each shift absorbed complexity from the caller and placed it in the infrastructure. DevOps absorbed manual deployment. GitOps absorbed manual kubectl. PlatformOps absorbed Kubernetes knowledge. IntentOps absorbs the manifest itself — the last Kubernetes concept that callers have had to understand.

When intent is the artifact, the caller knows their vocabulary and their target. Everything else is the delivery surface's problem. The CRD schema can change. The cluster can change. The operator can change. The intent stays the same.

---

## I've been building this without knowing that's what I was building

I started Orkestra to solve a specific problem: writing Kubernetes operators required writing Go, and most of what you wrote was boilerplate. I built a declarative runtime. That was the plan.

The plan didn't survive contact with the next question. And the question after that. And the question after that.

The runtime became a gateway. The gateway became a serve layer. The serve layer got a translation layer. The translation layer enabled schema evolution. Schema evolution combined with provenance enabled context-aware reconciliation. Context-aware reconciliation made the cluster an implementation detail.

I kept following the same principle — separate intent from infrastructure — and it kept leading somewhere new. Three years later I'm looking at what exists and trying to name it.

IntentOps might be the name.

The intent file is the artifact. The delivery surface translates it. The cluster is an implementation detail.

If that's right, we're not at the end of this shift. We're at the beginning of it.

---

*Orkestra is an open source implementation of the IntentOps model for Kubernetes. Early access is open.*

*[orkestra.sh](https://orkestra.sh)*