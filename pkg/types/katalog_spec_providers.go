// pkg/types/katalog_spec_providers.go
//
// In ReconcilerConfig:
//   ProviderBlocks []ProviderBlock `yaml:"-" json:"-"` // parsed from providers: map
//
// And a new raw field for YAML parsing:
//   RawProviders map[string][]map[string]interface{} `yaml:"providers,omitempty"`
//
// The raw field is parsed into ProviderBlocks during Katalog loading.
// See ParseProviderBlocks below.

package types

import "fmt"

// ParseProviderBlocks converts the raw YAML map from the providers: key into
// a structured []ProviderBlock. Called during Katalog loading after YAML unmarshal.
//
// Input (from YAML unmarshal):
//
//	map[string][]map[string]interface{}{
//	  "aws": [
//	    {"s3": {"bucket": "my-bucket", "region": "us-east-1"}},
//	    {"rds": {"instanceClass": "db.t3.micro", "engine": "postgres"}},
//	  ],
//	  "mongodb": [
//	    {"database": {"name": "mydb"}},
//	  ],
//	}
//
// Output:
//
//	[]ProviderBlock{
//	  {Name: "aws", Declarations: [
//	    {Kind: "s3", Fields: {"bucket": "my-bucket", "region": "us-east-1"}},
//	    {Kind: "rds", Fields: {"instanceClass": "db.t3.micro", "engine": "postgres"}},
//	  ]},
//	  {Name: "mongodb", Declarations: [
//	    {Kind: "database", Fields: {"name": "mydb"}},
//	  ]},
//	}
func ParseProviderBlocks(raw map[string][]map[string]interface{}) ([]ProviderBlock, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	blocks := make([]ProviderBlock, 0, len(raw))

	for blockName, rawDecls := range raw {
		block := ProviderBlock{
			Name:         blockName,
			Declarations: make([]RawProviderDeclaration, 0, len(rawDecls)),
		}

		for i, rawDecl := range rawDecls {
			decl, err := parseOneDeclaration(rawDecl)
			if err != nil {
				return nil, fmt.Errorf("provider %q, declaration[%d]: %w", blockName, i, err)
			}
			block.Declarations = append(block.Declarations, decl)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// parseOneDeclaration parses one map entry from a provider block list.
//
// Each entry is a single-key map where the key is the resource kind
// and the value is the fields map plus optional "when" / "anyOf" keys.
//
//	{"s3": {"bucket": "my-bucket", "region": "us-east-1", "when": [...], "anyOf": [...]}}
func parseOneDeclaration(raw map[string]interface{}) (RawProviderDeclaration, error) {
	// The declaration must have exactly one "kind" key.
	// "when" and "anyOf" are special — they hold conditions, not fields.
	var kind string
	var fieldsRaw map[string]interface{}

	for k, v := range raw {
		if k == "when" || k == "anyOf" {
			continue
		}
		if kind != "" {
			return RawProviderDeclaration{}, fmt.Errorf("declaration has multiple kind keys: %q and %q", kind, k)
		}
		kind = k
		if m, ok := v.(map[string]interface{}); ok {
			fieldsRaw = m
		}
	}

	if kind == "" {
		return RawProviderDeclaration{}, fmt.Errorf("declaration has no kind key")
	}

	decl := RawProviderDeclaration{
		Kind:   kind,
		Fields: flattenFields(fieldsRaw, ""),
	}

	// Parse when: conditions (AND)
	if whenRaw, ok := raw["when"]; ok {
		conditions, err := parseConditions(whenRaw)
		if err != nil {
			return decl, fmt.Errorf("parsing when: conditions: %w", err)
		}
		decl.Conditions = conditions
	}

	// Parse anyOf: conditions (OR)
	if anyOfRaw, ok := raw["anyOf"]; ok {
		conditions, err := parseConditions(anyOfRaw)
		if err != nil {
			return decl, fmt.Errorf("parsing anyOf: conditions: %w", err)
		}
		decl.AnyOf = conditions
	}

	return decl, nil
}

// flattenFields recursively flattens a nested map into dot-notation keys.
//
//	{"credentials": {"secretName": "my-secret"}, "bucket": "my-bucket"}
//	→ {"credentials.secretName": "my-secret", "bucket": "my-bucket"}
func flattenFields(m map[string]interface{}, prefix string) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch typed := v.(type) {
		case string:
			result[key] = typed
		case int, int64, float64, bool:
			result[key] = fmt.Sprintf("%v", typed)
		case map[string]interface{}:
			// Recurse into nested maps
			for subKey, subVal := range flattenFields(typed, key) {
				result[subKey] = subVal
			}
		default:
			if v != nil {
				result[key] = fmt.Sprintf("%v", v)
			}
		}
	}
	return result
}

// parseConditions converts a raw YAML when: value into []Condition.
// The raw value is []interface{} where each element is map[string]interface{}.
func parseConditions(raw interface{}) ([]Condition, error) {
	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("when: must be a list")
	}

	conditions := make([]Condition, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("when[%d]: must be a map", i)
		}

		cond := Condition{}
		if f, ok := m["field"].(string); ok {
			cond.Field = f
		}
		if op, ok := m["operator"].(string); ok {
			cond.Operator = ConditionOperator(op)
		}
		if v, ok := m["value"].(string); ok {
			cond.Value = v
		}
		if v, ok := m["equals"].(string); ok {
			cond.Equals = v
		}
		if v, ok := m["notEquals"].(string); ok {
			cond.NotEquals = v
		}
		if v, ok := m["hasPrefix"].(string); ok {
			cond.Prefix = v
		}
		if v, ok := m["hasSuffix"].(string); ok {
			cond.Suffix = v
		}
		if v, ok := m["contains"].(string); ok {
			cond.Contains = v
		}
		if v, ok := m["gt"].(string); ok {
			cond.GreaterThan = v
		}
		if v, ok := m["lt"].(string); ok {
			cond.LessThan = v
		}

		conditions = append(conditions, cond)
	}

	return conditions, nil
}
