# Testing Leader-Led Deployments

Many Kubernetes operators run with multiple replicas for high availability. Only one replica holds the leader election Lease at a time — it is the only one actively reconciling. Followers stand by and may return stale or empty state for any endpoint that reflects reconciler activity.

This creates a problem for E2E tests that use `kubectl port-forward svc/<service>`: Kubernetes routes the connection to a random pod. If the test lands on a follower, assertions about reconciler state will fail or return misleading defaults — not because the operator is broken, but because the wrong pod answered.

---

## The solution: `leaderElection` on kubectl subcommands

Four kubectl subcommands support a `leaderElection` block that resolves the target pod from a Kubernetes `coordination.k8s.io/v1` Lease before running the command: `port-forward`, `logs`, `delete`, and `exec`.

```yaml
kubectl:
  port-forward:
    - namespace: my-operator-system
      port: 8080
      path: /metrics
      leaderElection:
        lease: my-operator-leader
      outputContains: "reconcile_total"
```

At test time, the harness runs:

```bash
kubectl get lease my-operator-leader -n my-operator-system \
  -o jsonpath='{.spec.holderIdentity}'
# → my-operator-7d9f8b6c4-xkqpz
```

Then opens:

```bash
kubectl port-forward pod/my-operator-7d9f8b6c4-xkqpz 8080:8080 -n my-operator-system
```

Every assertion now runs against the leader. There is no randomness in pod selection.

---

## Logs from the leader pod

`kubectl.logs` with `leaderElection` asserts a log line that only the leader emits — a reconcile event, a lock acquisition message, an election confirmation:

```yaml
kubectl:
  logs:
    - leaderElection:
        lease: my-operator-leader
        namespace: my-operator-system
      outputContains: "acquired leader lock"
```

The harness resolves the Lease holder, then runs `kubectl logs <holder-pod> -n <namespace>`. Label selectors would fan out across all replicas including followers; `leaderElection` pins the log read to exactly the pod that holds the Lease.

---

## Deleting the leader pod

`kubectl.delete` with `leaderElection` deletes the current leader pod by name — without knowing it in advance. This is the clean way to test failover: kill the leader, then assert that a new one is elected and resumes serving.

```yaml
kubectl:
  delete:
    - leaderElection:
        lease: my-operator-leader
        namespace: my-operator-system
```

This replaces the shell anti-pattern that was previously the only option:

```bash
# fragile — breaks on restart, fails validate, hard to read
kubectl delete pod \
  $(kubectl get lease my-operator-leader -n my-operator-system \
    -o jsonpath='{.spec.holderIdentity}') \
  -n my-operator-system
```

---

## Exec into the leader pod

`kubectl.exec` with `leaderElection` runs a command inside the leader pod — useful for inspecting in-memory state or files that only the leader writes:

```yaml
kubectl:
  exec:
    - leaderElection:
        lease: my-operator-leader
        namespace: my-operator-system
      command: [cat, /var/run/leader-token]
      outputContains: "valid"
```

---

## When to use it

Use `leaderElection` whenever the assertion depends on state that only the leader maintains:

- reconciler metrics or counters
- in-memory cache endpoints
- health endpoints that reflect actual CRD state
- any endpoint where a follower would return `pending`, `0`, or empty values
- log lines emitted only by the elected leader (lock acquisition, reconcile events)

For endpoints that are consistent across replicas (e.g. `/readyz`, `/livez`, static config), a service port-forward is fine.

---

## Lease namespace

The Lease namespace defaults to the `namespace` field of the port-forward entry. Set it explicitly if the Lease lives elsewhere:

```yaml
kubectl:
  port-forward:
    - namespace: default
      port: 8080
      path: /metrics
      leaderElection:
        lease: my-operator-leader
        namespace: kube-system       # lease is here
      outputContains: "reconcile_total"
```

---

## If the Lease has no holder yet

If no pod holds the Lease at the time the test runs, the step retries until the checkpoint `timeout` is reached. This handles startup races where the operator is still negotiating leadership.

---

## Example: Orkestra runtime

Orkestra runtime deployments use a Lease named `orkestra-konductor`. The elected leader (the konductor) is the only replica that runs reconcilers and serves authoritative katalog state. E2E tests that assert health endpoints or error state use `leaderElection` to reach it:

```yaml
kubectl:
  port-forward:
    - namespace: orkestra-system
      port: 8080
      path: /katalog/myapp/health
      leaderElection:
        lease: orkestra-konductor
      jq: state
      equals: "healthy"
```

## Try it

```bash
ork init --pack resilience/leader-failover

# Follow the steps in README
```
---

## Schema reference

→ [kubectl.port-forward leaderElection](../../reference/schema/04-e2e/07-kubectl.md#leaderelection)  
→ [kubectl.logs](../../reference/schema/04-e2e/07-kubectl.md#kubectllogs)  
→ [kubectl.delete](../../reference/schema/04-e2e/07-kubectl.md#kubectldelete)  
→ [kubectl.exec](../../reference/schema/04-e2e/07-kubectl.md#kubectlexec)
