# 03 — Channels

## Overview

`dispatchTeam` fans out to every channel configured for the team. Currently supported: **email** (SMTP) and **Slack** (incoming webhooks). Both are optional — a team can declare only one or both.

```go
func dispatchTeam(
    ctx context.Context,
    k *katalog.Katalog,
    teamName string,
    team *orktypes.NotificationTeam,
    subject, message string,
    data map[string]interface{},
)
```

Channel dispatch is conditional: email fires only when `k.IsEmailNotificationEnabled()` and the SMTP host is configured; Slack fires only when `k.IsSlackNotificationEnabled()` and a webhook URL is available.

---

## Email (SMTP)

### Configuration

SMTP credentials are read from the environment by `pkg/konfig`:

| Env var | Required | Default | Description |
|---------|----------|---------|-------------|
| `SMTP_HOST` | yes | — | SMTP server hostname |
| `SMTP_PORT` | no | `587` | SMTP port (STARTTLS submission) |
| `SMTP_USER` | yes | — | SMTP login username |
| `SMTP_PASS` | yes | — | SMTP login password |
| `SMTP_FROM` | yes | — | Envelope from address |

### Katalog team declaration

```yaml
notification:
  teams:
    ops:
      email:
        - ops@example.com
        - oncall@example.com
```

### Delivery model

- One SMTP connection per `dispatchTeam` call.
- Each recipient gets a separate `MAIL FROM / RCPT TO / DATA` transaction within the same connection — avoids exposing the recipient list.
- STARTTLS is attempted; if the server does not support it, the connection continues unencrypted with a warning log.
- Failed deliveries per recipient are logged as warnings. Other recipients in the same call are still attempted.

### Message format

The email uses plain text with these headers:

```
From: <SMTP_FROM>
To: <recipient>
Subject: [orkestra] <cond.Field> condition triggered
X-Orkestra-Katalog: <katalogName>
X-Orkestra-Team: <teamName>

<message body>
```

---

## Slack

### Configuration

```yaml
notification:
  slack:
    webhook: https://hooks.slack.com/services/...   # global webhook
  teams:
    ops:
      slack:
        - "#ops-alerts"
        - "#incidents"
```

The webhook URL is resolved per team: `katalog.Notification.EffectiveSlackWebhook(teamName)` checks for a team-specific webhook first, falling back to the global one.

### Delivery model

- One HTTP POST to the webhook URL per send, regardless of how many channel names are listed.
- Channel names in the team config are included in the payload's attachment fields (informational) — Slack routing is controlled by the webhook's configuration, not the payload.
- The HTTP request has a 5-second timeout. Non-200 responses are returned as errors and logged.

### Payload format

```json
{
  "text": "<message>",
  "attachments": [{
    "color": "warning",
    "fields": [
      {"title": "Katalog", "value": "<katalogName>", "short": true},
      {"title": "Team",    "value": "<teamName>",    "short": true}
    ]
  }]
}
```

Severity colours: `"good"` (info), `"warning"` (warning — default for condition triggers), `"danger"`.

---

## Adding a new channel

1. Create `<channel>.go` in `pkg/notification/` with a `send<Channel>Notification(ctx, ...)` function.
2. Add the team config field to `orktypes.NotificationTeam` in `pkg/types/notification.go`.
3. Add a guard (`k.Is<Channel>NotificationEnabled()`) and call in `dispatchTeam`.
4. Add the new channel to this document.
