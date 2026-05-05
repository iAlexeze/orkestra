# pkg/tunnel — Developer Documentation

This directory explains how the tunnel package works and why it is designed the way it is.

## Documents

| File | What it covers |
|------|----------------|
| [01-providers.md](01-providers.md) | The `Provider` interface, cloudflared vs ngrok, auto-selection |
| [02-state.md](02-state.md) | Multi-tunnel state map, persistence, reuse guards, lifecycle |
| [03-expose.md](03-expose.md) | `Expose()` and `ExposeOptions` — the primary entry point |
| [04-port-detection.md](04-port-detection.md) | Port resolution priority, `startPortForward`, deterministic port hashing |
| [05-survival.md](05-survival.md) | SIGPIPE root cause, file-based stderr, Setsid, DNS timing fix |

Read them in order the first time. [05 — Process Survival](05-survival.md) documents the hardest bugs; read it before touching cloudflare.go or ngrok.go.
