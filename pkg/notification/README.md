# pkg/notification

The notification package dispatches alerts when a Katalog condition evaluates to true. It supports email (SMTP) and Slack channels, with per-team interval enforcement to prevent alert floods.

Notifications are condition-driven: a condition that is true triggers its `notify:` block. Each team listed in the block receives a message through every channel configured for that team. A per-condition, per-team timestamp gate suppresses re-delivery until the declared interval has elapsed.

## What lives here

| File | Role |
|------|------|
| `notification.go` | `NotificationState` — per-team timestamp tracking; `ProcessConditionNotifications` entry point; `dispatchTeamNotifications` fan-out |
| `email.go` | `sendEmailNotification` — SMTP dispatch; `SMTPConfig`; STARTTLS upgrade |
| `slack.go` | `sendSlackNotification` — Incoming Webhook POST; `SlackPayload` JSON builder |

## Developer documentation

Full step-by-step documentation is in [docs/](docs/README.md).

| I want to… | Go to |
|-----------|-------|
| Understand the notification flow from condition to delivery | [01 — Overview](docs/01-overview.md) |
| Configure `notify:` on a condition | [02 — NotifyBlock](docs/02-notify-block.md) |
| Add a new channel or understand email / Slack dispatch | [03 — Channels](docs/03-channels.md) |
