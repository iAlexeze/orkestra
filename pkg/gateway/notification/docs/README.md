# Notification — Developer Documentation

This directory explains how the `pkg/notification` package works and how to configure notifications in a Katalog.

## Documents

| File | What it covers |
|------|----------------|
| [01-overview.md](01-overview.md) | The notification flow from condition evaluation to delivery |
| [02-notify-block.md](02-notify-block.md) | `notify:` on a condition — `teams`, `message`, and interval enforcement |
| [03-channels.md](03-channels.md) | Email (SMTP) and Slack dispatch — configuration and failure handling |
| [04-developer-notifications.md](04-developer-notifications.md) | Out-of-the-box deployment readiness notifications for every developer |

Read them in order the first time. For a quick reference when adding `notify:` to a condition, jump straight to [02-notify-block.md](02-notify-block.md).

For the "how does this work for me on day one" story, read [04-developer-notifications.md](04-developer-notifications.md).
