package note

import "text/template"

// serviceNotes registers helpers for inspecting Service networking fields and
// endpoint readiness.
//
// Usage:
//
//	tmpl.Funcs(note.serviceNotes())
//
// Template examples:
//
//	{{ serviceClusterIP .children.service }}
//	{{ serviceNodePort .children.service "http" }}
//	{{ serviceLoadBalancerIP .children.service }}
//	{{ serviceLoadBalancerHost .children.service }}
//	{{ endpointsReady .children.service }}
//
// These helpers provide a concise way to surface clusterIP, nodePort,
// load‑balancer details, and endpoint readiness—useful for gating traffic
// routing, validating exposure, or ensuring downstream components only proceed
// once a Service is fully addressable.
func serviceNotes() template.FuncMap {
	return template.FuncMap{
		"serviceClusterIP":        noteServiceClusterIP,
		"serviceNodePort":         noteServiceNodePort,
		"serviceLoadBalancerIP":   noteServiceLoadBalancerIP,
		"serviceLoadBalancerHost": noteServiceLoadBalancerHost,
		"endpointsReady":          noteEndpointsReady,
	}
}

// ── Service notes ─────────────────────────────────────────────────────────────

// noteServiceClusterIP returns the assigned ClusterIP for a Service.
// Returns "" before the IP is assigned or when the Service doesn't exist.
//
//	{{ serviceClusterIP .children.service }}  → "10.96.0.1"
func noteServiceClusterIP(obj interface{}) string {
	spec := noteGet(obj, "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return ""
	}
	v, _ := m["clusterIP"].(string)
	return v
}

// noteServiceNodePort returns the NodePort for the first port of a Service.
// Returns 0 before the NodePort is assigned.
// Use containerPortByName for named ports.
//
//	{{ serviceNodePort .children.service }}  → 31234
func noteServiceNodePort(obj interface{}) int {
	spec := noteGet(obj, "spec")
	m, ok := spec.(map[string]interface{})
	if !ok {
		return 0
	}
	ports, ok := m["ports"].([]interface{})
	if !ok || len(ports) == 0 {
		return 0
	}
	pm, ok := ports[0].(map[string]interface{})
	if !ok {
		return 0
	}
	return int(toInt64(pm["nodePort"]))
}

// noteServiceLoadBalancerIP returns the external IP assigned by the load balancer.
// Returns "" when not yet assigned (common — LB provisioning takes time).
//
//	{{ serviceLoadBalancerIP .children.service }}  → "34.123.45.67"
//
// Use in status fields to surface the external IP once available:
//
//   - path: externalIP
//     value: "{{ serviceLoadBalancerIP .children.service }}"
func noteServiceLoadBalancerIP(obj interface{}) string {
	ingress := lbIngress(obj)
	if ingress == nil {
		return ""
	}
	ip, _ := ingress["ip"].(string)
	return ip
}

// noteServiceLoadBalancerHost returns the hostname assigned by the load balancer.
// Cloud providers typically assign a hostname rather than an IP.
//
//	{{ serviceLoadBalancerHost .children.service }}
//	→ "abc123.us-east-1.elb.amazonaws.com"
func noteServiceLoadBalancerHost(obj interface{}) string {
	ingress := lbIngress(obj)
	if ingress == nil {
		return ""
	}
	host, _ := ingress["hostname"].(string)
	return host
}

func lbIngress(obj interface{}) map[string]interface{} {
	status := noteStatus(obj)
	lb, ok := status["loadBalancer"].(map[string]interface{})
	if !ok {
		return nil
	}
	ingress, ok := lb["ingress"].([]interface{})
	if !ok || len(ingress) == 0 {
		return nil
	}
	m, _ := ingress[0].(map[string]interface{})
	return m
}

// noteEndpointsReady returns true when the Endpoints resource for a Service
// has at least one ready address. Useful for gating Ingress creation on
// actual backend availability, not just Service existence.
//
//	{{ endpointsReady .children.service }}
//
// Note: this reads from the Endpoints object, not the Service itself.
// Access via cross: declaration pointing to the Endpoints resource.
func noteEndpointsReady(obj interface{}) bool {
	status := noteStatus(obj)
	subsets, ok := status["subsets"].([]interface{})
	if !ok {
		// Endpoints object stores data in subsets, not status
		// Try the top-level object
		if m, ok := obj.(map[string]interface{}); ok {
			subsets, ok = m["subsets"].([]interface{})
			if !ok {
				return false
			}
		}
	}
	for _, s := range subsets {
		sm, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		addrs, ok := sm["addresses"].([]interface{})
		if ok && len(addrs) > 0 {
			return true
		}
	}
	return false
}
