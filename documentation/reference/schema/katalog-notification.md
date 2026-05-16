# notification

Configures alerting dispatched when reconciliation errors occur or when a watched condition changes.

```yaml
notification:
  enabled: true
  defaults:
    interval: 15m
    slackWebhookUrl: $SLACK_WEBHOOK_URL

  teams:
    platform:
      slack:
        webhook: $PLATFORM_SLACK_WEBHOOK
      email:
        - platform@example.com
      interval: 5m
      message: "{{ .KatalogName }} failed on {{ .CRDName }}: {{ .Error }}"

    oncall:
      slack:
        webhook: $ONCALL_WEBHOOK
      interval: 1m
```

## Top-level fields

| Field | Default | Description |
|-------|---------|-------------|
| `enabled` | `true` (when block declared) | Gate all notification dispatch. Set `false` to silence everything without removing the config. |
| `defaults.interval` | `15m` | Minimum time between notifications globally. Per-team overrides this. |
| `defaults.slackWebhookUrl` | — | Global fallback Slack webhook URL (supports `$ENV_VAR`). |

## `teams`

A named map of notification targets. Each team receives its own notifications independently.

| Field | Description |
|-------|-------------|
| `slack.webhook` | Slack incoming webhook URL (supports `$ENV_VAR`). |
| `email` | List of email recipients. Requires SMTP env vars configured at startup. |
| `interval` | Per-team minimum interval between notifications. Overrides `defaults.interval`. |
| `message` | Go template for the notification body. |

## Message template variables

| Variable | Description |
|----------|-------------|
| `{{ .KatalogName }}` | Name of the Katalog |
| `{{ .CRDName }}` | Name of the failing CRD |
| `{{ .Error }}` | The reconcile error message |
| `{{ .Namespace }}` | CR namespace |
| `{{ .Name }}` | CR name |

---

→ Next: [katalog-providers.md](katalog-providers.md)
