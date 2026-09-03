# The Katalog Is a File

The Katalog is a file. Not a Kubernetes object, not a cluster record, not a CRD applied to etcd. A file — read at startup, validated without a cluster, versioned in Git, distributed as an OCI artifact, reviewed in a pull request.

That choice is not incidental. It is the decision that every other property of Orkestra depends on.

---

## The schema belongs to Orkestra

When configuration lives in etcd as a Kubernetes object, the schema is a contract between you, the cluster, and every tool that has ever read that object. Changing it requires a version conversion path, migration webhooks, and careful coordination with stored state that might be months old. The operator that manages the CRD and the CRD that describes the operator become entangled — each requiring the other to already be in the right state before it can change.

When configuration is a file, the schema belongs to Orkestra alone. Only Orkestra reads it. That means Orkestra can change it — improve it, restructure it, break it intentionally — without asking the cluster's permission. A field that belongs in a different place can move. A concept that was modelled wrong the first time can be remodelled. The update path is: change the YAML, run `ork validate`, start the runtime. No migration controller. No conversion webhook. No stored objects to reconcile against the new shape.

This is what makes refactoring possible in a way that CRD-based configuration never could be. The schema can be changed aggressively because the only thing that must change alongside it is your file.

---

## Configuration is a human artifact

Files are written by people. They are reviewed in pull requests, diffed line by line, discussed before they are merged, validated in CI before they reach a cluster. A field that changes from `warn` to `deny` appears as one changed line. The reviewer sees it. The pipeline validates it. The change is traceable from intent to production.

A cluster object is a record. Records are applied, observed, and patched. They accumulate history that nobody asked for and express state that is not quite what anyone wrote. The Katalog is not a record of what was applied — it is the statement of what should happen. That distinction belongs in a file, not in etcd.

`ork validate` runs against the file. No cluster. No credentials. No running runtime. The full configuration surface is checkable before anything is deployed. That is only possible because the configuration lives outside the cluster.

---

## Distribution requires a file

The OrkestraRegistry works because the Katalog is a file. Patterns are OCI artifacts — YAML files packaged, signed, annotated with quality signals, versioned, and distributed through the same infrastructure as container images. Any OCI-compatible registry can host them. Any team with the right credentials can pull them.

If the Katalog were a CRD, distribution would mean installing CRDs in a target cluster to describe how other CRDs should behave. You would need a running cluster before you could inspect a pattern. You would need Orkestra running before you could validate the pattern you are about to give to Orkestra. The registry model exists because the artifact is self-contained — a file you can inspect, validate, and reason about without any cluster at all.

---

## Your cluster surface belongs to you

When `kubectl get crds` lists the CRDs in a cluster managed by Orkestra, it shows your CRDs. Not Orkestra's. The infrastructure that runs your operators does not add itself to the surface it manages.

This is only possible because the Katalog is a file. An operator platform that configures itself via CRDs must install those CRDs in your cluster — they appear alongside your domain objects, require RBAC entries, become part of the surface every engineer encounters when they look at the cluster, and can be deleted just like any other Kubernetes resource. Orkestra refuses to add that noise. The cluster belongs to your domain. Orkestra's configuration belongs in your repository.

---

## Two interfaces, one principle

There are two things you need from an operator platform: a way to declare what you want, and a way to observe what is happening.

The file is the right interface for declaration. You write it. You own it. You version it.

The `/katalog` API is the right interface for observation. It reflects live runtime state — queue depth, reconcile totals, active workers, conversion latency, admission stats. State that changes every second cannot live in a file, and it should not live in a Kubernetes object that shows you what was applied rather than what is happening.

Orkestra answers both through the right interface for each. The file is not authoritative about what is running — the API is. The API is not the place to express intent — the file is. Neither is a substitute for the other. Neither steps on the other.

The Katalog is a file because that is what it is: a statement of human intent. Orkestra reads it and speaks native Kubernetes. The distance between what you mean and how you express it stays at zero.
