package types

// IDPResponseConfig declares how the gateway transforms the CR before
// returning it to the caller.
//
// Evaluated at request time against the CR already fetched from the API server.
// At apply time: the submitted CR (spec, metadata, labels, annotations only —
// children not yet created). At GET time: the full stored CR including status.
//
// The gateway does not fetch additional resources, call the runtime, or reason
// about Kubernetes internals. Its job remains narrow: authenticate, apply,
// fetch, delete, and — with this config — shape what the caller sees.
//
// Example:
//
//	idp:
//	  config:
//	    response:
//	      default: true
//	      payload:
//	        phase:       '{{ .status.phase }}'
//	        serviceURL:  'https://{{ .metadata.name }}.{{ .spec.environment }}.myorg.io'
//	        nextSteps:   '{{ nextSteps }}'
//	      exclude:       '{{ toList (getAnnotation . "platform.myorg.io/exclude") }}'
type IDPResponseConfig struct {
	// Default controls whether the full CR is included in the response before
	// payload and exclude are applied.
	//
	// true  (default) — start with the full CR; payload fields are merged in
	//                   on top, exclude paths are stripped afterward.
	// false           — start with an empty map; only payload fields appear.
	//
	// Omitting this field is equivalent to true — existing callers always
	// receive the full CR unless the platform team explicitly opts out.
	Default *bool `yaml:"default,omitempty" json:"default,omitempty"`

	// Payload is a map of named template expressions added to the response.
	//
	// Keys become top-level fields in the returned JSON object (alongside
	// apiVersion, kind, metadata, spec, status when default: true, or as the
	// sole content when default: false).
	//
	// Values are Go template expressions evaluated by the full Resolver with
	// the note FuncMap. Plain strings (no {{ }}) pass through unchanged.
	// Expressions that fail to resolve or produce "<no value>" are included
	// as empty strings — the caller can poll until values are populated.
	//
	// At apply time: .spec, .metadata, labels, and annotations are available.
	// At GET time:   .status is also available because the runtime has written it.
	//
	// Example:
	//   payload:
	//     phase:      '{{ .status.phase }}'
	//     serviceURL: 'https://{{ .metadata.name }}.myorg.io'
	//     nextSteps:  '{{ nextSteps }}'
	Payload map[string]string `yaml:"payload,omitempty" json:"payload,omitempty"`

	// Exclude is a template expression that resolves to a list of dot-notation
	// field paths to strip from the response after payload is applied.
	//
	// Two forms are accepted:
	//   exclude: "metadata.managedFields,status.observedGeneration"
	//            — plain comma-separated string, trimmed and split
	//   exclude: '{{ toList (getAnnotation . "platform.myorg.io/exclude") }}'
	//            — template that resolves to a comma-separated string
	//
	// Use the built-in toList note to convert comma-separated strings (from
	// annotations, notes, or literals) into the slice that the engine expects.
	//
	// ork validate catches the case where a path appears in both payload and
	// exclude — exclude wins, but the conflict is surfaced as a warning.
	Exclude string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

// UseDefault reports whether the full CR should be the starting point.
// Returns true when Default is nil (omitted) or explicitly true.
func (r *IDPResponseConfig) UseDefault() bool {
	return r.Default == nil || *r.Default
}

// HasPayload reports whether any payload expressions are declared.
func (r *IDPResponseConfig) HasPayload() bool {
	return len(r.Payload) > 0
}

// HasExclude reports whether an exclude expression is declared.
func (r *IDPResponseConfig) HasExclude() bool {
	return r.Exclude != ""
}
