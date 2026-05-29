package types

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// EnrichTarget is one enrichment declaration in a CRD's enrich: list.
//
// Shorthand (always enriches):
//
//	enrich:
//	  - pods
//
// Struct form (conditional enrichment):
//
//	enrich:
//	  - events:
//	      when:
//	        - field: "{{ replicasReady .children.deployment }}"
//	          equals: "false"
//	  - replicasets:
//	      anyOf:
//	        - field: spec.debug
//	          equals: "true"
type EnrichTarget struct {
	// Key is the enrichment target name (pods, events, pvcs, etc.)
	Key string

	// When gates this enrichment using AND semantics. Template expressions
	// in field: values are evaluated against the resolver's full data map
	// so all note functions (replicasReady, hasCrashingPod, …) are available.
	// Empty: always enriches (same as shorthand).
	When  []Condition `yaml:"when,omitempty"`
	AnyOf []Condition `yaml:"anyOf,omitempty"`
}

// UnmarshalYAML handles both shorthand ("pods") and struct form:
//
//   - pods              → EnrichTarget{Key: "pods"}
//   - events:           → EnrichTarget{Key: "events", When: [...]}
//     when: [...]
func (e *EnrichTarget) UnmarshalYAML(value *yaml.Node) error {
	// Scalar: "pods" → EnrichTarget{Key: "pods"}
	if value.Kind == yaml.ScalarNode {
		e.Key = value.Value
		return nil
	}

	// Mapping: first key is the target name, value is the condition block.
	if value.Kind == yaml.MappingNode {
		if len(value.Content) < 2 {
			return fmt.Errorf("enrich target mapping must have exactly one key")
		}
		e.Key = value.Content[0].Value
		var body struct {
			When  []Condition `yaml:"when"`
			AnyOf []Condition `yaml:"anyOf"`
		}
		if err := value.Content[1].Decode(&body); err != nil {
			return fmt.Errorf("enrich target %q: %w", e.Key, err)
		}
		e.When = body.When
		e.AnyOf = body.AnyOf
		return nil
	}

	return fmt.Errorf("enrich target must be a string or mapping, got node kind %d", value.Kind)
}

// MarshalYAML serializes back to the shorthand form when there are no
// conditions, and to the struct form when conditions are present.
func (e EnrichTarget) MarshalYAML() (interface{}, error) {
	if len(e.When) == 0 && len(e.AnyOf) == 0 {
		return e.Key, nil
	}
	type body struct {
		When  []Condition `yaml:"when,omitempty"`
		AnyOf []Condition `yaml:"anyOf,omitempty"`
	}
	return map[string]body{
		e.Key: {When: e.When, AnyOf: e.AnyOf},
	}, nil
}
