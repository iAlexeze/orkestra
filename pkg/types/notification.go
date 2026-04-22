// pkg/types/notification_refined.go
//
// This file replaces the KatalogNotification type in pkg/types.
// It adds:
//   - enabled: true/false at the notification block level
//   - per-team Slack webhookUrl (overrides the global default)
//   - a global default Slack webhook
//   - message template field per team for customisation
//   - rollback integration (notify on rollback via existing team references)
//
// YAML shape:
//
//	notification:
//	  enabled: true
//	  defaults:
//	    interval: 15m
//	    slackWebhookUrl: "https://hooks.slack.com/services/..."  # global default
//	  teams:
//	    platform:
//	      email: ["platform@company.io"]
//	      slack: ["#platform-alerts"]
//	      slackWebhookUrl: "https://hooks.slack.com/services/..."  # override
//	      interval: 5m
//	      message: "{{ .metadata.name }} in {{ .metadata.namespace }}: {{ .status.conditions.Ready.message }}"
//	    oncall:
//	      slack: ["#oncall"]
//	      interval: 1m
//
//	# On conditions:
//	when:
//	  - field: status.phase
//	    equals: Degraded
//	    notify: [platform, oncall]
//
//	# On rollback:
//	rollback:
//	  trigger:
//	    consecutiveFailures: 3
//	  notify: [oncall]     # fires when rollback activates
//
// SMTP config is read from pkg/konfig env vars — not declared in YAML.
// Slack webhook URL is per-team or global default — not from env (user choice).
package types

// KatalogNotification holds the complete notification configuration for a Katalog.
// Declared at the same level as spec:, metadata:, and security:.
type KatalogNotification struct {
	// Enabled gates all notification dispatch. Default: true when the
	// notification: block is declared. Set false to silence all channels
	// without removing the configuration.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Defaults defines global notification behavior applied when a team does
	// not specify its own override.
	Defaults *NotificationDefaults `yaml:"defaults,omitempty"`

	// Teams declares named notification targets. Conditions and rollback blocks
	// reference teams by name via notify: ["platform", "oncall"].
	Teams map[string]*NotificationTeam `yaml:"teams,omitempty"`
}

// NotificationDefaults defines global defaults applied when a team does not
// override specific settings.
type NotificationDefaults struct {
	// Interval is the minimum time between notifications for the same
	// condition+team pair while the condition remains true. Default: 15m.
	Interval Duration `yaml:"interval,omitempty"`

	// SlackWebhookUrl is the global default Slack incoming webhook URL.
	// Used when a team declares slack: channels but no slackWebhookUrl.
	// Can be overridden per team.
	SlackWebhookUrl string `yaml:"slackWebhookUrl,omitempty"`
}

// NotificationTeam defines channels and behavior for one named notification target.
type NotificationTeam struct {
	// Email is the list of email recipients. SMTP config comes from pkg/konfig
	// env vars (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM).
	Email []string `yaml:"email,omitempty"`

	// Slack is the list of Slack channels to notify (e.g. "#platform-alerts").
	// Requires SlackWebhookUrl (team or global default) to be configured.
	Slack []string `yaml:"slack,omitempty"`

	// SlackWebhookUrl is the incoming webhook URL for this team's Slack workspace.
	// Overrides notification.defaults.slackWebhookUrl for this team.
	// Required when Slack is non-empty and no global default is set.
	SlackWebhookUrl string `yaml:"slackWebhookUrl,omitempty"`

	// Interval overrides the global default for this team. Zero duration falls
	// back to notification.defaults.interval or the built-in default of 15m.
	Interval Duration `yaml:"interval,omitempty"`

	// Message is a Go template expression evaluated against the full resolver
	// data (.spec.*, .metadata.*, .status.*, .children.*, metrics.*).
	// When empty, Orkestra uses a default message format:
	//   "[orkestra] {katalogName}/{crdKind} {crName}: {condition.Field} {op} {value}"
	Message string `yaml:"message,omitempty"`
}

// IsEnabled returns true when notifications are configured and not explicitly disabled.
func (n *KatalogNotification) IsEnabled() bool {
	if n == nil {
		return false
	}
	if len(n.Teams) == 0 {
		return false
	}
	if n.Enabled != nil && !*n.Enabled {
		return false
	}
	return true
}

// HasTeams returns true when at least one team is declared.
func (n *KatalogNotification) HasTeams() bool {
	return n != nil && len(n.Teams) > 0
}

// EffectiveInterval returns the notification interval for the given team.
// Resolution: team.Interval → defaults.Interval → fallback (15m).
func (n *KatalogNotification) EffectiveInterval(teamName string) Duration {
	defaultInterval := Duration{}
	defaultInterval.Duration = 15 * 60 * 1e9 // 15m in nanoseconds

	if n == nil {
		return defaultInterval
	}
	if team, ok := n.Teams[teamName]; ok && team != nil {
		if team.Interval.Duration > 0 {
			return team.Interval
		}
	}
	if n.Defaults != nil && n.Defaults.Interval.Duration > 0 {
		return n.Defaults.Interval
	}
	return defaultInterval
}

// EffectiveSlackWebhook returns the Slack webhook URL for the given team.
// Resolution: team.SlackWebhookUrl → defaults.SlackWebhookUrl → "".
func (n *KatalogNotification) EffectiveSlackWebhook(teamName string) string {
	if n == nil {
		return ""
	}
	if team, ok := n.Teams[teamName]; ok && team != nil && team.SlackWebhookUrl != "" {
		return team.SlackWebhookUrl
	}
	if n.Defaults != nil {
		return n.Defaults.SlackWebhookUrl
	}
	return ""
}

// HasEmailTargets returns true when the team has at least one email recipient.
func (n *KatalogNotification) HasEmailTargets(teamName string) bool {
	if n == nil {
		return false
	}
	team, ok := n.Teams[teamName]
	return ok && team != nil && len(team.Email) > 0
}

// HasSlackTargets returns true when the team has Slack channels AND a webhook URL.
func (n *KatalogNotification) HasSlackTargets(teamName string) bool {
	if n == nil {
		return false
	}
	team, ok := n.Teams[teamName]
	if !ok || team == nil || len(team.Slack) == 0 {
		return false
	}
	return n.EffectiveSlackWebhook(teamName) != ""
}

// IsEmailEnabled returns true when email is usable for at least one team.
// hasSMTPConfig is passed from pkg/konfig — avoids importing konfig here.
func (n *KatalogNotification) IsEmailEnabled(hasSMTPConfig bool) bool {
	if !n.IsEnabled() || !hasSMTPConfig {
		return false
	}
	for name := range n.Teams {
		if n.HasEmailTargets(name) {
			return true
		}
	}
	return false
}

// IsSlackEnabled returns true when Slack is usable for at least one team.
func (n *KatalogNotification) IsSlackEnabled() bool {
	if !n.IsEnabled() {
		return false
	}
	for name := range n.Teams {
		if n.HasSlackTargets(name) {
			return true
		}
	}
	return false
}

// EffectiveMessage returns the notification message template for a team,
// falling back to the default format when none is declared.
func (n *KatalogNotification) EffectiveMessage(teamName string) string {
	const defaultMsg = "[orkestra] {{ .metadata.namespace }}/{{ .metadata.name }}: condition triggered"
	if n == nil {
		return defaultMsg
	}
	team, ok := n.Teams[teamName]
	if !ok || team == nil || team.Message == "" {
		return defaultMsg
	}
	return team.Message
}
