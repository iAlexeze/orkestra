# I've Been Trying to Describe Orkestra in One Sentence for Two Years

*And every time I thought I had it, the project outgrew the words.*

---

I shipped the first version of Orkestra with a GitHub description that said: **"Declarative runtime for Kubernetes operators."**

I was confident about it. It was accurate. It described exactly what existed — a runtime that let you declare operator behavior in YAML instead of writing Go. You wrote a Katalog. Orkestra reconciled. No controllers. No informers. No workqueues. Just behavior, declared.

That description lasted about four months.

---

## The first problem

The first time someone asked me what Orkestra was and I said "a declarative runtime for Kubernetes operators," I watched their face. They nodded in the way people nod when they don't want to ask a follow-up question.

The word "operators" was the problem. To most engineers, operators mean something specific — the Go programs that extend Kubernetes. Saying Orkestra was a runtime for operators made it sound like something that ran those Go programs. Which is not what it does. It replaces them.

So I changed it to **"Declarative runtime for Kubernetes behaviors."**

Behaviors instead of operators. More honest. You're not writing an operator — you're declaring behavior, and Orkestra implements it. The word "behaviors" at least gestured at the right thing.

That description lasted about two months.

---

## The gateway

Then I built the gateway.

The gateway was a new component alongside the runtime — an HTTP server that sat in front of the cluster and accepted intent from any caller. CI pipelines. Browser forms. Slack commands. curl. Any system that could make an HTTP POST could now deliver intent to the platform without touching kubectl, without knowing what a CR was, without understanding Kubernetes at all.

The runtime reconciled. The gateway delivered. These were different concerns. "Declarative runtime" described the first one. It said nothing about the second.

I changed it to **"Declarative control plane for Kubernetes operators."**

Control plane felt right because it covered both components. A control plane coordinates. Orkestra coordinated.

But I wasn't sure about it. "Control plane" is a Kubernetes-internal term. It sounds like something that manages the cluster infrastructure — the API server, etcd, the scheduler. That's not what Orkestra is. Orkestra runs on the cluster. It uses the cluster. It is not the cluster's control plane.

I left it in the docs and kept building.

---

## The serve layer

Then I built the serve layer.

This is the part that broke every description I had.

The serve layer started as a small experiment. The platform team had built these beautiful operators using Orkestra. But developers who needed to use the platform still had to write YAML and run kubectl. The gap between "the platform works" and "developers can use the platform" was still a Kubernetes-shaped gap.

So I added `idp:` to the CRD entry and a Create button appeared in the Control Center. The form was generated from the CRD schema. A developer filled it in. A CR appeared. The operator reconciled.

Clean. But I noticed something wrong. The Control Center was doing too much. It fetched the full CRD schema. It knew which fields went into `spec` versus `metadata.labels`. It built the complete Kubernetes CR before sending it. Every caller — the Control Center, a CI pipeline, a curl command — was constructing Kubernetes objects.

I spent days thinking about why this felt wrong.

Then it clicked: the caller shouldn't know any of that. The Katalog already declared all of it. If the gateway read the Katalog and the CRD entry had a target identifier, the gateway could look up the CRD, read the field declarations, and build the CR itself. The caller would just submit:

```json
{"target": "app", "repository": "myorg/payments-api", "environment": "staging"}
```

No apiVersion. No kind. No spec. No Kubernetes knowledge.

And it worked.

---

## The path addition — and the schema evolution accident

Then I added `path` to field declarations. A dot-notation address within the spec so nested CRD structures could be supported. A caller submitting `"cpu": "500m"` could have that value routed to `spec.app.resources.cpu` without knowing it existed.

And then I noticed something I hadn't set out to build.

When a CRD field moved — when `spec.app.repository` restructured into `spec.source.repository` — the platform team updated one line in the serve declaration. One line. Every caller submitted the same intent they always had. The CRD had evolved. They had noticed nothing.

I had accidentally solved Kubernetes schema evolution.

---

## The translation layer

But path routing only got you so far. It answered the question of *where* a value goes in the spec. It didn't answer the question of *what form* it should take when it gets there.

Real platforms have this problem constantly. A developer thinks in cron strings — `"0 2 * * 1-5"`. The CRD expects five structured fields: minute, hour, day of month, month, day of week. These are the same thing expressed in two completely different shapes. Before the translation layer, the caller had to know the CRD's shape. They had to submit the five fields. The abstraction leaked.

So I added `value` and `values` to field declarations.

`value` transforms a single submitted value before it reaches the spec — strip a prefix, normalise a format, apply a note function. `values` fans one submitted value out into multiple spec fields:

```yaml
serve:
  fields:
    schedule:
      label: "Schedule (cron)"
      values:
        schedule.minute:     '{{ cronMinute .value }}'
        schedule.hour:       '{{ cronHour   .value }}'
        schedule.dayOfMonth: '{{ cronDom    .value }}'
        schedule.month:      '{{ cronMonth  .value }}'
        schedule.dayOfWeek:  '{{ cronDow    .value }}'
```

The caller submits `"0 2 * * 1-5"`. The CR receives five structured fields. Neither side sees the other's format. The serve layer is the contract between them.

This is when I understood what the gateway actually was. Not a delivery mechanism. A translation layer. It stood between the world's vocabulary — whatever form intent naturally takes for the people expressing it — and Kubernetes's vocabulary — whatever structure the CRD required. Those two vocabularies never had to match. The gateway translated between them.

The serve layer was not an IDP. It was not a portal. It was a stable interface — a surface through which intent could arrive in any form, from any source, and be translated into exactly what the cluster expected, without either side knowing what the other looked like.

---

## The names stopped fitting

At this point I had to rename everything.

The gateway was called the Apply API. But it wasn't just applying anymore. It was resolving targets, transforming values, fanning out fields, building CRs, validating intent, stamping provenance, and returning structured responses.

The configuration block was called `idp:`. But what I'd built wasn't an Internal Developer Platform. It was a way to serve a CRD to the world — to any caller, from any source, in any vocabulary — without that caller needing to know Kubernetes existed.

I renamed everything. `pkg/gateway/applyapi` became `pkg/gateway/api`. The YAML key `idp:` became `serve:`.

And then I tried to update the GitHub description.

I stared at it for a long time.

---

## The pattern I hadn't noticed

Somewhere around the sixth rewrite I noticed the pattern.

Every description I'd written was correct. Each one accurately described Orkestra at the moment I wrote it. And each one became wrong — not because Orkestra changed direction, but because Orkestra kept growing in the same direction. The implementation kept revealing new implications of the same core idea.

The runtime was a consequence of "separate intent from infrastructure." The gateway was a consequence of the same principle applied to delivery. The serve layer was a consequence of the same principle applied to the caller interface. Schema evolution fell out of following the principle to its logical conclusion for field routing. The translation layer fell out of asking what happens when the caller's vocabulary and the CRD's vocabulary don't match.

One principle. Many consequences. Each consequence needed a different description. No single sentence covered them all.

I wasn't struggling to describe Orkestra because I couldn't write. I was struggling to describe Orkestra because I was still discovering what it was.

---

## Where I landed

After two years and six descriptions, here is the one I keep returning to:

**"The missing layer between Kubernetes and the teams who use it."**

It doesn't describe features. It describes the gap. The gap between what Kubernetes gives you and what the teams using Kubernetes actually need — a stable interface they can build on, deliver through, and evolve without coordination overhead.

Every team running Kubernetes has felt that gap. Platform engineers feel it when they realise developers still need to write YAML to use the platform they built. Developers feel it when they need to learn kubectl to deploy an application. SREs feel it when a CRD schema change requires migrating every CI pipeline.

Orkestra fills that gap.

---

## What comes next

I am currently experimenting with something that would have been unthinkable when I wrote that first description.

The gateway now has multiple receiving points — direct API calls, GitHub webhooks, GitLab webhooks, Slack commands, generic HTTP integrations. Intent arrives from anywhere. The gateway translates and delivers.

The question I'm asking now: what if the gateway could deliver to multiple clusters?

I've built internal platforms before that connected multiple clusters with ArgoCD using Terraform to manage the connections — a setup that is genuinely difficult to build and almost impossible to find clear guidance on. I understand the shape of the problem. The gateway's architecture — a single translation layer with multiple intake points — suggests a natural extension to multiple delivery points. One intent, multiple clusters, the gateway routing based on whatever logic the platform team declares.

I'm close. But that's a story for another post.

---

I've rewritten the README six times. Each time was correct. Each time was temporary.

If you're building something that keeps outgrowing its description, that's not a communication problem. That's what it looks like to build something the ecosystem didn't have a word for yet.

Keep building until you find the word. Or until the word finds you.

---

*Orkestra is in early access. The missing layer is open source.*

*[orkestra.sh](https://orkestra.sh)*