# Scaling Signals

`autoscale:` conditions evaluate against the same Resolver data map used everywhere else in a Katalog. Any source resolved before deployment evaluation — `external:`, `cross:`, time notes, user-defined notes — is available as a field reference in `when:` and `anyOf:` blocks.

---

## Time conditions

```yaml
scaleUp:
  conditions:
    when:
      - dayOfWeek:
          weekday: true
      - time:
          after: "09:00"
          before: "18:00"
  target: 8

scaleDown:
  conditions:
    when:
      - field: "{{ inBusinessHours }}"
        equals: "false"
  target: 2
```

Time conditions use the built-in `time:` and `dayOfWeek:` blocks, or user-defined notes that wrap them. The reconciler re-evaluates on every resync and converges to the correct replica count for the current time.

---

## External HTTP

```yaml
onReconcile:
  external:
    - name: queue
      url: "{{ .spec.metricsEndpoint }}"
      timeout: 5s
      continueOnError: true

  deployments:
    - name: "{{ .metadata.name }}"
      autoscale:
        scaleUp:
          conditions:
            when:
              - field: external.queue.pendingJobs
                greaterThan: "100"
          increment: 2
```

`external:` resolves before deployment evaluation. The response is available at `external.<name>.*` when conditions are checked. Any JSON HTTP endpoint works: Kafka consumer lag, RabbitMQ queue depth, SQS queue attributes, Prometheus instant queries, custom APIs. When the response body is a valid JSON object, its keys are merged directly into the result — `external.queue.pendingJobs` works for `{ "pendingJobs": 100 }`, and `external.queue.metrics.pendingJobs` works for `{ "metrics": { "pendingJobs": 100 } }`.

`continueOnError: true` means a temporarily unavailable endpoint does not error the reconcile — conditions are not met, the Deployment holds its current replica count, and the operator retries on the next resync.

---

## Cross-operator metrics

```yaml
cross:
  - crd: jobqueue
    selector:
      name: "{{ .spec.queueName }}"
      namespace: "{{ .metadata.namespace }}"
    as: queue

deployments:
  - name: "{{ .metadata.name }}"
    autoscale:
      scaleUp:
        conditions:
          when:
            - field: cross.queue.status.queueDepth
              greaterThan: "100"
        increment: 2
```

`cross:` reads a sibling CRD's resolved state. In-binary lookups go through the informer cache. For cross-binary or cross-cluster, `cross:` with a `source:` block fetches via HTTP with caching. Any field the sibling publishes to status is reachable at `cross.<name>.status.*`.

---

## Notes as scaling guards

```yaml
scaleUp:
  conditions:
    when:
      - field: "{{ hasCrashingPod .children.deployment }}"
        equals: "false"
      - field: external.queue.pendingJobs
        greaterThan: "100"
  increment: 2
```

Notes can guard scale-up — here, the operator only scales when there are no crashing pods and the queue is deep. Notes that return booleans evaluate naturally in `when:` blocks.

---

## Composite conditions

```yaml
scaleUp:
  conditions:
    anyOf:
      - field: external.kafka.consumerLag
        greaterThan: "1000"
      - field: cross.worker.metrics.queueDepth
        greaterThan: "200"
      - time:
          after: "08:00"
          before: "09:00"
        dayOfWeek:
          weekday: true
```

`anyOf:` evaluates as OR — scale up if any condition passes. `when:` evaluates as AND — all conditions must pass. Both can appear in the same `conditions:` block; `when:` and `anyOf:` are evaluated independently and combined with AND.
