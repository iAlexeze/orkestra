package types

// KatalogNotification holds the global notification configuration for a Katalog.
// It defines who can be notified and how often, while conditions declare when
// notifications should fire via their `notify:` field.
//
// Notifications are resolved in two steps:
//
//  1. Conditions declare intent:
//     when:
//     - field: status.phase
//     equals: Degraded
//     notify:
//     - platform
//     - application
//
//  2. KatalogNotification resolves targets and behavior:
//     notification:
//     defaults:
//     interval: 15m
//     teams:
//     platform:
//     email: ["platform@example.com"]
//     interval: 5m
//     application:
//     email: ["app@example.com"]
//     slack: ["#app-alerts"]
//
// At runtime, when a condition transitions from false → true, Orkestra:
//   - looks up each team in Teams
//   - applies the effective interval (team override → defaults → hardcoded)
//   - sends notifications only if the interval has elapsed since the last send
//     while the condition remains true.
//
// nil pointer: notifications not configured; `notify:` on conditions is ignored.
type KatalogNotification struct {
	// Defaults defines global notification behavior applied when a team does not
	// override specific settings (e.g. interval).
	//
	// nil pointer: no explicit defaults; runtime falls back to built-in values.
	Defaults *NotificationDefaults `yaml:"defaults,omitempty"`

	// Teams declares named notification targets that conditions can reference
	// via `notify:`. Each team can define one or more channels (email, Slack)
	// and an optional interval override.
	//
	// Example:
	//   teams:
	//     platform:
	//       email: ["platform@example.com"]
	//       interval: 5m
	//     application:
	//       email: ["app@example.com"]
	//       slack: ["#app-alerts"]
	//
	// Empty map or nil: no teams available; `notify:` references will fail
	// validation or be treated as no-op, depending on policy.
	Teams map[string]*NotificationTeam `yaml:"teams,omitempty"`
}

// NotificationDefaults defines global notification behavior applied when a team
// does not specify its own settings.
type NotificationDefaults struct {
	// Interval is the minimum time between notifications for the same
	// condition+team pair while the condition remains true.
	//
	// Example: 15m means:
	//   - first transition to true → send immediately
	//   - subsequent reconciles while still true → suppressed until 15m passes
	//
	// Zero duration: runtime falls back to a built-in default (e.g. 15m).
	Interval Duration `yaml:"interval,omitempty"`
}

// NotificationTeam defines the channels and behavior for a single named team.
// Conditions reference teams by name via `notify: ["platform", "application"]`.
type NotificationTeam struct {
	// Email is the list of email recipients for this team.
	// Empty slice or nil: no email notifications for this team.
	Email []string `yaml:"email,omitempty"`

	// Slack is the list of Slack channels or users for this team.
	// Format is implementation-defined (e.g. "#channel", "@user").
	// Empty slice or nil: no Slack notifications for this team.
	Slack []string `yaml:"slack,omitempty"`

	// Interval overrides the global default for this team. When set, it defines
	// the minimum time between notifications for this team while the condition
	// remains true.
	//
	// Zero duration: fall back to NotificationDefaults.Interval or built-in
	// default if no global default is configured.
	Interval Duration `yaml:"interval,omitempty"`
}

// ── Effective notification helpers ─────────────────────────────────────────────

// HasNotificationsConfigured returns true when any notification configuration
// is present. This is a coarse check — channels may still be disabled or
// missing credentials.
func (n *KatalogNotification) HasNotificationsConfigured() bool {
	if n == nil {
		return false
	}
	if len(n.Teams) == 0 {
		return false
	}
	return true
}

// EffectiveInterval returns the effective notification interval for a team.
// Resolution order:
//  1. Team.Interval
//  2. Defaults.Interval
//  3. fallback (envDefault or hardcoded)
func (n *KatalogNotification) EffectiveInterval(teamName string, fallback Duration) Duration {
	if n == nil {
		return fallback
	}

	if team, ok := n.Teams[teamName]; ok {
		if team.Interval.Duration > 0 {
			return team.Interval
		}
	}

	if n.Defaults != nil && n.Defaults.Interval.Duration > 0 {
		return n.Defaults.Interval
	}

	return fallback
}

// HasEmailTargets returns true if the given team has at least one email target.
func (n *KatalogNotification) HasEmailTargets(teamName string) bool {
	if n == nil {
		return false
	}
	team, ok := n.Teams[teamName]
	if !ok || team == nil {
		return false
	}
	return len(team.Email) > 0
}

// HasSlackTargets returns true if the given team has at least one Slack target.
func (n *KatalogNotification) HasSlackTargets(teamName string) bool {
	if n == nil {
		return false
	}
	team, ok := n.Teams[teamName]
	if !ok || team == nil {
		return false
	}
	return len(team.Slack) > 0
}

// IsEmailNotificationEnabled returns true when email notifications are
// effectively enabled for at least one team AND the required SMTP environment
// configuration is present (checked by the caller via hasSMTPConfig).
func (n *KatalogNotification) IsEmailNotificationEnabled(hasSMTPConfig bool) bool {
	if n == nil || !hasSMTPConfig {
		return false
	}
	for name := range n.Teams {
		if n.HasEmailTargets(name) {
			return true
		}
	}
	return false
}

// IsSlackNotificationEnabled returns true when Slack notifications are
// effectively enabled for at least one team AND the required Slack webhook
// configuration is present (checked by the caller via hasSlackWebhook).
func (n *KatalogNotification) IsSlackNotificationEnabled(hasSlackWebhook bool) bool {
	if n == nil || !hasSlackWebhook {
		return false
	}
	for name := range n.Teams {
		if n.HasSlackTargets(name) {
			return true
		}
	}
	return false
}
