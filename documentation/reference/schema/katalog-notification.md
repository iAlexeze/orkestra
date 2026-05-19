# notification

Configures alerting dispatched when reconciliation errors occur or when a watched condition changes.

```yaml
notification:
  enabled: true
  standalone: true
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
| `standalone` | `false` in-cluster; `true` outside cluster | Controls the dispatch path. `true` — runtime dispatches directly to SMTP/Slack without a gateway. `false` — notifications are POSTed to the gateway's `/notify` endpoint. Defaults to `true` automatically when running outside a Kubernetes cluster (local dev). |
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

## Dispatch path

When a condition triggers, the runtime evaluates `standalone` to choose how to deliver the notification:

- **Standalone path** (`standalone: true`, or running outside a Kubernetes cluster): the runtime dispatches directly to SMTP and Slack using the configured credentials. No gateway is required.
- **Gateway path** (`standalone: false`): the runtime POSTs an `Event` payload to `<gatewayEndpoint>/notify`. The gateway then handles SMTP/Slack delivery. Use this when centralising notification dispatch through the Orkestra gateway.

If `standalone` is not declared, the default is `false` when running inside a cluster and `true` when running locally (outside a cluster).

---

→ Next: [katalog-providers.md](katalog-providers.md)
