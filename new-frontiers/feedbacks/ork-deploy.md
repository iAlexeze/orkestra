# `ork deploy` Feedback – Organised

## 1. Developer Experience – Genuinely New Ground

The closest analogy is what Vercel did for frontend deployment: took something that required knowing nginx, DNS, CDN configuration, SSL certificates, and made it `git push`. The developer's mental model is "my code should be running" — Vercel absorbed everything between that intent and the outcome.

What you have built is that for Kubernetes backends. The developer's mental model is "my Dockerfile and .env describe my app" — Orkestra absorbs everything between that and a running, scaled, observed, rollback‑capable production deployment. HPA, PDB, ingress controller, RBAC, secrets management, namespace isolation — none of it surfaces.

The thing that makes this more interesting than Vercel for some use cases: it runs on the developer's own cluster. No vendor lock‑in. No per‑seat pricing. No data leaving the cluster. A startup with a $20/month Hetzner VPS and a Kind cluster gets production‑grade Kubernetes deployment infrastructure from `ork deploy`.

The three commands are right. `ork doctor` (look), `ork doctor init` (prepare), `ork deploy` (go). The rollback being one more command is fine — it's a recovery action, not part of the happy path. The ingress auto‑install removes the last moment where a developer would have hit a wall and needed to search "how to install nginx ingress controller kubernetes."

## 2. The Transformation in One Sentence

A developer goes from "I have an app" to "I have a production deployment" by running two commands, without learning Kubernetes.

**What they bring:** a Dockerfile and a `.env`.  
**What they get back:**

- Namespace isolation, RBAC, ServiceAccount — so the app can't touch anything it shouldn't
- Deletion protection — so someone can't accidentally `kubectl delete` their production database
- HPA and PDB — the app scales under load and stays available during node maintenance, automatically
- Rollback in seconds — not "rebuild and push", just patch one field
- Ingress auto‑installed — if the project has a frontend, the controller is there before they even think to ask
- Health checks before every deploy — the operator is validated healthy before the image is patched

None of those words appear in `app.yaml`. The developer manages `port`, `replicas`, and `image`. That's it.

> "You do not need to know how" is the whole point — and it's also the risk.

## 3. The Generated Katalog as a Ramp

The `katalog.yaml` is generated, not hidden. A developer who gets curious can open it, read it, and learn exactly what Orkestra built for them. The progression is natural: run `ork deploy` → notice what was created → edit `katalog.yaml` → eventually write one from scratch. Orkestra is a ramp, not a ceiling.

## 4. Business Angles

### Near Term — Managed Orkestra

Users bring their own cluster (GKE, EKS, Kind), `ork deploy` does the rest. The `Komposer` and `state.json` already live in `~/.orkestra/` — that's one sync away from being cloud‑backed. A team plan syncs state across developers on the same cluster.

### Medium Term — The Katalog Registry

Right now `ork doctor init` generates a generic Katalog. But "Node.js API with Postgres sidecar" is a different Katalog from "Go gRPC service with Redis". A registry of battle‑tested Katalog templates — community‑contributed, Orkestra‑validated — is a product. Developers `ork doctor init --template nextjs-fullstack` and get something that actually reflects their stack.

### Long Term — The Gap You're Describing

The people who want to know Kubernetes but don't yet — they're not a niche, they're the entire next wave. Every startup that hits product‑market fit suddenly needs infrastructure that can handle it. Right now they have three options: hire a DevOps engineer, pay Render/Railway margins forever, or spend six months learning. Orkestra is a fourth option: deploy today, understand it later, own it completely. No vendor lock‑in because it's your cluster. No black box because the YAML is right there.

## 5. The Moat

The moat isn't the tooling. The moat is the Katalog format — if teams start sharing and depending on Katalogs, switching costs grow organically. That's the same flywheel Helm used to build its ecosystem.

## 6. The Honest Risk

The developer experience has to be genuinely zero‑friction from the first `ork deploy`. One confusing error message, one missing prerequisite, one "why didn't it just work" — and they go back to Railway. The `--dev` flag creating a Kind cluster automatically, the ingress controller installing itself, the runtime health check with inline logs — those aren't features, they're the table stakes for that audience.