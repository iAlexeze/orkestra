// pkg/note/kube_service_test.go
package note

import (
	"testing"
)

func TestNoteServiceClusterIP(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want string
	}{
		{"nil object", nil, ""},
		{"no spec", map[string]interface{}{}, ""},
		{"spec without clusterIP", map[string]interface{}{
			"spec": map[string]interface{}{},
		}, ""},
		{"valid clusterIP", map[string]interface{}{
			"spec": map[string]interface{}{
				"clusterIP": "10.96.0.1",
			},
		}, "10.96.0.1"},
		{"clusterIP empty string", map[string]interface{}{
			"spec": map[string]interface{}{
				"clusterIP": "",
			},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteServiceClusterIP(tt.obj); got != tt.want {
				t.Errorf("noteServiceClusterIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteServiceNodePort(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want int
	}{
		{"nil object", nil, 0},
		{"no spec", map[string]interface{}{}, 0},
		{"spec without ports", map[string]interface{}{
			"spec": map[string]interface{}{},
		}, 0},
		{"ports empty slice", map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": []interface{}{},
			},
		}, 0},
		{"single port with nodePort", map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{"nodePort": 31234},
				},
			},
		}, 31234},
		{"first port only", map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{"nodePort": 30000},
					map[string]interface{}{"nodePort": 30001},
				},
			},
		}, 30000},
		{"nodePort missing", map[string]interface{}{
			"spec": map[string]interface{}{
				"ports": []interface{}{
					map[string]interface{}{"port": 80},
				},
			},
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteServiceNodePort(tt.obj); got != tt.want {
				t.Errorf("noteServiceNodePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteServiceLoadBalancerIP(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want string
	}{
		{"nil", nil, ""},
		{"no status", map[string]interface{}{}, ""},
		{"status no loadBalancer", map[string]interface{}{
			"status": map[string]interface{}{},
		}, ""},
		{"loadBalancer no ingress", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{},
			},
		}, ""},
		{"ingress empty slice", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"ingress": []interface{}{},
				},
			},
		}, ""},
		{"ingress with ip", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{"ip": "1.2.3.4"},
					},
				},
			},
		}, "1.2.3.4"},
		{"ingress with hostname but no ip", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{"hostname": "elb.amazonaws.com"},
					},
				},
			},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteServiceLoadBalancerIP(tt.obj); got != tt.want {
				t.Errorf("noteServiceLoadBalancerIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteServiceLoadBalancerHost(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want string
	}{
		{"nil", nil, ""},
		{"ingress with hostname", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{"hostname": "test.elb.amazonaws.com"},
					},
				},
			},
		}, "test.elb.amazonaws.com"},
		{"ingress with ip only", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{
					"ingress": []interface{}{
						map[string]interface{}{"ip": "1.2.3.4"},
					},
				},
			},
		}, ""},
		{"no ingress", map[string]interface{}{
			"status": map[string]interface{}{
				"loadBalancer": map[string]interface{}{},
			},
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteServiceLoadBalancerHost(tt.obj); got != tt.want {
				t.Errorf("noteServiceLoadBalancerHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoteEndpointsReady(t *testing.T) {
	tests := []struct {
		name string
		obj  interface{}
		want bool
	}{
		{"nil", nil, false},
		{"empty object", map[string]interface{}{}, false},
		{"subsets empty", map[string]interface{}{
			"subsets": []interface{}{},
		}, false},
		{"subsets with no addresses", map[string]interface{}{
			"subsets": []interface{}{
				map[string]interface{}{},
			},
		}, false},
		{"subsets with addresses slice but empty", map[string]interface{}{
			"subsets": []interface{}{
				map[string]interface{}{
					"addresses": []interface{}{},
				},
			},
		}, false},
		{"subsets with one address", map[string]interface{}{
			"subsets": []interface{}{
				map[string]interface{}{
					"addresses": []interface{}{
						map[string]interface{}{"ip": "10.0.0.1"},
					},
				},
			},
		}, true},
		{"multiple subsets, first has addresses", map[string]interface{}{
			"subsets": []interface{}{
				map[string]interface{}{
					"addresses": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}},
				},
				map[string]interface{}{},
			},
		}, true},
		{"status field (some objects store subsets in status)", map[string]interface{}{
			"status": map[string]interface{}{
				"subsets": []interface{}{
					map[string]interface{}{
						"addresses": []interface{}{map[string]interface{}{"ip": "10.0.0.1"}},
					},
				},
			},
		}, true},
		// EndpointSlice format (discovery.k8s.io/v1)
		{"endpointslice no endpoints key", map[string]interface{}{
			"kind": "EndpointSlice",
		}, false},
		{"endpointslice empty endpoints", map[string]interface{}{
			"endpoints": []interface{}{},
		}, false},
		{"endpointslice endpoint not ready", map[string]interface{}{
			"endpoints": []interface{}{
				map[string]interface{}{
					"addresses":  []interface{}{"10.0.0.1"},
					"conditions": map[string]interface{}{"ready": false},
				},
			},
		}, false},
		{"endpointslice endpoint ready but no addresses", map[string]interface{}{
			"endpoints": []interface{}{
				map[string]interface{}{
					"addresses":  []interface{}{},
					"conditions": map[string]interface{}{"ready": true},
				},
			},
		}, false},
		{"endpointslice one ready endpoint", map[string]interface{}{
			"endpoints": []interface{}{
				map[string]interface{}{
					"addresses":  []interface{}{"10.0.0.1"},
					"conditions": map[string]interface{}{"ready": true},
				},
			},
		}, true},
		{"endpointslice mixed ready/not-ready, one ready", map[string]interface{}{
			"endpoints": []interface{}{
				map[string]interface{}{
					"addresses":  []interface{}{"10.0.0.2"},
					"conditions": map[string]interface{}{"ready": false},
				},
				map[string]interface{}{
					"addresses":  []interface{}{"10.0.0.3"},
					"conditions": map[string]interface{}{"ready": true},
				},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noteEndpointsReady(tt.obj); got != tt.want {
				t.Errorf("noteEndpointsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
