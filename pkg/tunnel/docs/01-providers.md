# 01 — Providers

## The Provider interface

Every supported tunnel backend implements `Provider`:

```go
type Provider interface {
    Name()        string
    Available()   bool
    Install(ctx)  error
    Authenticate(ctx, token string) error
    Start(ctx, localPort int) (url string, pid int, err error)
    Stop(pid int) error
}
```

`Start` launches a **detached background daemon** and returns:
- the public URL once it is live and DNS-registered (not just announced)
- the daemon's PID so `Stop` and state persistence can manage it

The daemon must outlive the CLI process that launched it. See [05 — Process Survival](05-survival.md) for how this is achieved.

## Cloudflared (default)

`CloudflaredProvider` uses [Cloudflare Quick Tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/do-more-with-tunnels/trycloudflare/) (`trycloudflare.com`). No account, login, or config file is required.

**Binary location**: looked up in `$PATH` first, then `~/.orkestra/bin/cloudflared`. Auto-downloaded from the Cloudflare GitHub release if neither exists.

**Authenticate** is a no-op for cloudflared. Quick tunnels are anonymous.

**Why cloudflared is the default**: zero friction for local development. A developer can run `ork deploy --expose` on a fresh machine and get a working public URL without signing up anywhere.

**Log file**: `~/.orkestra/cloudflared-<port>.log`. Each tunnel instance writes to its own file keyed by local port so concurrent tunnels never intermix output.

## ngrok

`NgrokProvider` uses ngrok's free-tier HTTP tunnels. Requires an account and an auth token.

**Binary**: must be installed by the user — not auto-downloaded because ngrok's installer is platform-specific and requires accepting terms.

**Authenticate**: runs `ngrok config add-authtoken <token>` before starting the tunnel.

**Log file**: `~/.orkestra/ngrok-<port>.log`. ngrok writes structured JSON lines; the URL is extracted by parsing for a `"url"` field.

**Install** returns an error with a link to ngrok's download page and the exact command to re-run after installing.

## Provider selection

```go
// Auto-select: prefers an already-installed provider
p, err := tunnel.Select()

// Explicit: fail fast if the named provider isn't supported
p, err := tunnel.SelectByName("ngrok")
```

Auto-selection order:
1. Cloudflared (if binary found in PATH or `~/.orkestra/bin`)
2. Ngrok (if binary found in PATH)
3. Cloudflared (fallback — it can install itself on demand)

The `Expose` function calls `p.Install(ctx)` when `!p.Available()`, so a user who has never run `ork deploy --expose` before gets cloudflared downloaded and installed transparently.
