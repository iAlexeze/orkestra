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
// You can refine this later (e.g. include source name, index, etc.).
func conditionKey(cond orktypes.Condition) string {
	// Minimal but stable-ish: field + operator + value
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
	if len(cond.Notify) == 0 {
		return
	}
	if !k.HasTeams() {
		return
	}

	ck := conditionKey(cond)

	for _, teamName := range cond.Notify {
		team, ok := k.Notification.Teams[teamName]
		if !ok || team == nil {
			// Unknown team name — you may want to log this.
			continue
		}

		key := fmt.Sprintf("%s|%s|%s", k.Meta().Name, ck, teamName)
		last := s.LastSent[key]
		interval := k.NotificationInterval(teamName)

		if !last.IsZero() && now.Sub(last) < interval.Duration {
			// Interval not elapsed yet — skip.
			continue
		}

		// Dispatch notifications for this team.
		s.dispatchTeamNotifications(ctx, k, teamName, team, cond, data)

		// Record send time.
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
	data map[string]interface{},
) {
	if len(team.Email) > 0 && k.IsEmailNotificationEnabled() {
		host, port, user, pass, from := k.SMTPConfig()
		if host != "" && user != "" && pass != "" {
			_ = sendEmailNotification(ctx, host, port, user, pass, from, team.Email, k.Meta().Name, teamName, cond, data)
		}
	}

	if len(team.Slack) > 0 && k.IsSlackNotificationEnabled() {
		webhook := k.SlackWebhook()
		if webhook != "" {
			_ = sendSlackNotification(ctx, webhook, team.Slack, k.Meta().Name, teamName, cond, data)
		}
	}
}
