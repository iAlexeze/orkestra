# 03 — NotificationConfig

`NotificationConfig` represents the notification *capability* of the Orkestra runtime — whether it is able to send email or Slack messages at all. It does not encode *intent* (which CRD events trigger notifications, to which team, on which conditions). Intent lives in the Katalog YAML under `notification:`.

## Structure

```go
type NotificationConfig struct {
    Email struct {
        Enabled  bool
        SMTPHost string
        SMTPPort int
        SMTPUser string
        SMTPPass string
        From     string
    }
    Slack struct {
        Enabled bool
        Webhook string
    }
    DefaultInterval time.Duration
}
```

## Capability auto-detection

Both channels default their `Enabled` flag based on whether the required ENV vars are present:

- **Email**: `Enabled` defaults to `true` when `SMTP_HOST`, `SMTP_USER`, and `SMTP_PASS` are all set. Override explicitly with `ENABLE_EMAIL_NOTIFIER=false` if you have the SMTP vars set but want to disable notifications.
- **Slack**: `Enabled` defaults to `true` when `SLACK_WEBHOOK_URL` is set. Override with `ENABLE_SLACK_NOTIFIER=false`.

This means a deployment with SMTP vars populated is automatically email-capable without any additional configuration.

## DefaultInterval

`NOTIFY_DEFAULT_INTERVAL` (in seconds, default 900 = 15 minutes) is the fallback interval used when neither the Katalog nor the per-team configuration defines how often to re-notify for the same event. Teams that declare their own interval override this value.

## Precedence

`NotificationConfig` provides ENV-level defaults. The Katalog loader merges YAML values on top:

```yaml
notification:
  email:
    from: alerts@mycompany.com
  defaultInterval: 1800
```

These YAML values overwrite the corresponding fields in `kfg.Notification()` at Katalog load time. The `pkg/notification` package always reads from the merged `NotificationConfig` — it does not read ENV directly.

---

**Back →** [README](../README.md)
