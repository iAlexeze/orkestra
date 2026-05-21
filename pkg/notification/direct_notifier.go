package notification

import (
	"context"

	"github.com/orkspace/orkestra/pkg/katalog"
)

// DirectNotifier dispatches via SMTP/Slack without a gateway.
// Used when gatewayEndpoint is not configured.
type DirectNotifier struct {
	kat *katalog.Katalog
}

func NewDirectNotifier(kat *katalog.Katalog) *DirectNotifier {
	return &DirectNotifier{kat: kat}
}

// Dispatch fires the event directly to the team's configured channels.
// Throttle was already enforced by ProcessConditionNotifications — dispatch is unconditional.
func (d *DirectNotifier) Dispatch(ctx context.Context, ev Event) error {
	k := d.kat
	if k.Notification == nil {
		return nil
	}
	team, ok := k.Notification.Teams[ev.TeamName]
	if !ok || team == nil {
		return nil
	}
	dispatchTeam(ctx, k, ev.TeamName, team, ev.Subject, ev.Message, ev.Data)
	return nil
}
