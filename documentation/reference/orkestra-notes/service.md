# Service Notes

Inspect Kubernetes Service networking fields and endpoint readiness.

## Reference

| Note | Signature | Returns |
|------|-----------|---------|
| `serviceClusterIP` | `obj` | `string` — `""` before assigned |
| `serviceNodePort` | `obj` | `int` — first port's NodePort, `0` before assigned |
| `serviceLoadBalancerIP` | `obj` | `string` — `""` before assigned |
| `serviceLoadBalancerHost` | `obj` | `string` — hostname assigned by cloud LB |
| `endpointsReady` | `obj` | `bool` — at least one ready address in the Endpoints |

## Examples

```yaml
# Surface the external IP once the load balancer assigns it
- path: externalIP
  value: "{{ serviceLoadBalancerIP .children.service }}"

# Cloud LB hostname (AWS, GCP, Azure)
- path: externalHostname
  value: "{{ serviceLoadBalancerHost .children.service }}"

# Gate Ingress creation on backend availability
when:
  - field: "{{ endpointsReady .children.service }}"
    equals: "true"

# ClusterIP for internal DNS construction
- path: internalAddress
  value: "{{ serviceClusterIP .children.service }}"
```

## Notes

`endpointsReady` reads from the Endpoints object (which Orkestra injects alongside the Service). It returns `true` when at least one address is in the `addresses` array of any subset — meaning at least one Pod is ready to receive traffic.

`serviceLoadBalancerIP` and `serviceLoadBalancerHost` return `""` while the load balancer is provisioning. Use with a `when:` condition or `notEmpty` to gate dependent resources.
