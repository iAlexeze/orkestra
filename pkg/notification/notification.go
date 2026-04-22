package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/orkspace/orkestra/pkg/katalog"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// NotificationState tracks last-send timestamps per condition+team.
type NotificationState struct {
	// key: operatorName + "|" + conditionKey + "|" + teamName
	LastSent map[string]time.Time
}

type NotificationStack struct {
	Katalog      *katalog.Katalog
	State        *NotificationState
	ResolverData map[string]interface{}
}

func NewNotificationState() *NotificationState {
	return &NotificationState{
		LastSent: make(map[string]time.Time),
	}
}

// conditionKey is a stable identifier for a condition within an operatorBox.
func conditionKey(cond orktypes.Condition) string {
	op, val := orktypes.ResolveConditionOp(cond)
	return fmt.Sprintf("%s|%s|%s", cond.Field, op, val)
}

// ProcessConditionNotifications evaluates notify: for a single condition,
// tracking transitions and enforcing per-team intervals.
//
// Called after we've already decided the condition is "passed" (true).
func (s *NotificationState) ProcessConditionNotifications(
	ctx context.Context,
	k *katalog.Katalog,
	data map[string]interface{},
	cond orktypes.Condition,
	now time.Time,
) {
	if cond.Notify == nil || len(cond.Notify.Teams) == 0 {
		return
	}
	if !k.HasTeams() {
		return
	}

	ck := conditionKey(cond)

	for _, teamName := range cond.Notify.Teams {
		team, ok := k.Notification.Teams[teamName]
		if !ok || team == nil {
			continue
		}

		key := fmt.Sprintf("%s|%s|%s", k.Meta().Name, ck, teamName)
		last := s.LastSent[key]
		interval := k.NotificationInterval(teamName)

		if !last.IsZero() && now.Sub(last) < interval.Duration {
			continue
		}

		// Message: condition-level override > team default
		message := cond.Notify.Message
		if message == "" {
			message = k.Notification.EffectiveMessage(teamName)
		}

		s.dispatchTeamNotifications(ctx, k, teamName, team, cond, message, data)
		s.LastSent[key] = now
	}
}

// dispatchTeamNotifications fans out to email/slack/etc for a single team.
func (s *NotificationState) dispatchTeamNotifications(
	ctx context.Context,
	k *katalog.Katalog,
	teamName string,
	team *orktypes.NotificationTeam,
	cond orktypes.Condition,
	message string,
	data map[string]interface{},
) {
	if len(team.Email) > 0 && k.IsEmailNotificationEnabled() {
		host, port, user, pass, from := k.SMTPConfig()
		if host != "" && user != "" && pass != "" {
			cfg := SMTPConfig{
				Host: host,
				Port: fmt.Sprintf("%d", port),
				User: user,
				Pass: pass,
				From: from,
			}
			subject := fmt.Sprintf("%s condition triggered", cond.Field)
			_ = sendEmailNotification(ctx, cfg, team.Email, k.Meta().Name, teamName, subject, message)
		}
	}

	if len(team.Slack) > 0 && k.IsSlackNotificationEnabled() {
		webhook := k.Notification.EffectiveSlackWebhook(teamName)
		if webhook == "" {
			webhook = k.SlackWebhook()
		}
		if webhook != "" {
			_ = sendSlackNotification(ctx, webhook, team.Slack, k.Meta().Name, teamName, message, "warning")
		}
	}
}
