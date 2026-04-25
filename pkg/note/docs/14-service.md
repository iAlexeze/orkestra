# 14 — Service Notes

Service notes surface networking details from Service and Endpoints objects — cluster IP, node ports, load balancer addresses, and endpoint readiness. Use them to gate Ingress creation, surface external access points in status fields, or verify that a Service is actually routing traffic before proceeding.

---

## Reference

### `serviceClusterIP`

Return `spec.clusterIP`. Returns `""` before the IP is assigned or when the Service is absent.

```yaml
# value: "{{ serviceClusterIP .children.service }}"  → "10.96.0.1"

# Surface the cluster IP in status:
- path: clusterIP
  value: "{{ serviceClusterIP .children.service }}"
```

---

### `serviceNodePort`

Return the `nodePort` of the first Service port. Returns `0` before the port is assigned.

```yaml
# value: "{{ serviceNodePort .children.service }}"  → 31234
```

---

### `serviceLoadBalancerIP`

Return the external IP assigned by the load balancer (`status.loadBalancer.ingress[0].ip`). Returns `""` while provisioning — cloud LB provisioning typically takes 30–90 seconds.

```yaml
# value: "{{ serviceLoadBalancerIP .children.service }}"  → "34.123.45.67"

# Gate downstream on LB being ready:
when:
  - field: "{{ serviceLoadBalancerIP .children.service }}"
    notEquals: ""
```

---

### `serviceLoadBalancerHost`

Return the hostname assigned by the load balancer (`status.loadBalancer.ingress[0].hostname`). Cloud providers (AWS, GCP) typically assign a hostname rather than an IP.

```yaml
# value: "{{ serviceLoadBalancerHost .children.service }}"
# → "abc123.us-east-1.elb.amazonaws.com"

# Compose a full URL:
- path: externalURL
  value: "https://{{ serviceLoadBalancerHost .children.service }}"
```

---

### `endpointsReady`

Return `true` when the Endpoints resource for a Service has at least one ready address. Reads from the Endpoints object, not the Service itself.

```yaml
# Gate Ingress creation on actual backend availability:
when:
  - field: "{{ endpointsReady .children.service }}"
    equals: "true"
```

Note: access via a `cross:` declaration pointing to the Endpoints resource — the Endpoints object has the same name as the Service but is a separate resource.

---

## Complete pattern: surface and gate on load balancer

```yaml
status:
  fields:
    - path: loadBalancerIP
      value: "{{ serviceLoadBalancerIP .children.service }}"
    - path: loadBalancerHost
      value: "{{ serviceLoadBalancerHost .children.service }}"

resources:
  - kind: Ingress
    name: app
    when:
      - field: "{{ serviceLoadBalancerHost .children.service }}"
        notEquals: ""
      - field: "{{ endpointsReady .children.service }}"
        equals: "true"
```

---

## Quick reference

| Note | Signature | Returns |
|------|-----------|---------|
| `serviceClusterIP` | `(obj any)` | `string` |
| `serviceNodePort` | `(obj any)` | `int` |
| `serviceLoadBalancerIP` | `(obj any)` | `string` |
| `serviceLoadBalancerHost` | `(obj any)` | `string` |
| `endpointsReady` | `(obj any)` | `bool` |

---

**Next →** [15 — Field Notes](15-fields.md)
