package types

import "time"

// TimeWindow declares a clock-based active window.
type TimeWindow struct {
	// After — active after this time (format: "HH:MM" in 24h).
	After string `yaml:"after,omitempty" json:"after,omitempty"`

	// Before — active before this time (format: "HH:MM" in 24h).
	Before string `yaml:"before,omitempty" json:"before,omitempty"`
}

// DayOfWeekCondition declares which days the condition is active.
// Exactly one of Weekday, Weekend, In, or NotIn must be set.
type DayOfWeekCondition struct {
	// Weekday — active Mon–Fri. Shorthand for in: [Monday..Friday].
	Weekday *bool `yaml:"weekday,omitempty" json:"weekday,omitempty"`

	// Weekend — active Sat–Sun. Shorthand for in: [Saturday, Sunday].
	Weekend *bool `yaml:"weekend,omitempty" json:"weekend,omitempty"`

	// In — active on these days. Full English names: Monday, Tuesday, etc.
	In []string `yaml:"in,omitempty" json:"in,omitempty"`

	// NotIn — active on all days except these.
	NotIn []string `yaml:"notIn,omitempty" json:"notIn,omitempty"`
}

// Duration is a time.Duration that unmarshals from YAML strings like "15s", "2m", "1h".
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if s == "" {
		d.Duration = 0
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}
