# 05 — Process Survival

This document records three bugs that were found and fixed while building the tunnel package. Each required understanding a non-obvious OS or network behaviour. The fixes are not removable — removing any of them silently breaks tunnels.

---

## Bug 1: SIGPIPE kills cloudflared on parent exit

### Root cause

`cmd.Stderr = pipe` (from `cmd.StderrPipe()`) creates an OS pipe. The pipe has two ends: a write fd (held by cloudflared) and a read fd (held by the CLI process). When the CLI process exits, the read end is closed. cloudflared, actively writing startup logs to the write end, receives `SIGPIPE` and dies.

The app tunnel appeared to survive because the app was in a quiet state when the CLI exited. The Control Center tunnel always died because the CC service was actively writing SSE keep-alive logs, keeping cloudflared's write loop active at the moment of exit.

### Fix

```go
// cloudflare.go — Start()
logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
cmd.Stderr = logFile   // file, not a pipe
cmd.Start()
logFile.Close()        // CLI closes its copy; cloudflared keeps its own fd
```

Files do not have read-end / write-end semantics. When the CLI process exits and closes its fd, cloudflared's fd to the same file is unaffected. cloudflared continues writing logs. No SIGPIPE.

**The log file path** is `~/.orkestra/cloudflared-<localPort>.log`. Using the port as the discriminator ensures concurrent tunnels never write to the same file.

---

## Bug 2: port-forward dies on parent exit (process group)

### Root cause

When a process is started with `exec.Command` and the parent exits, the kernel sends SIGHUP to the process group if the parent was the session leader. `kubectl port-forward` is in the same session as the CLI, so it receives SIGHUP and exits. Without the port-forward, cloudflared has no local endpoint and serves 530 errors.

### Fix

```go
// expose.go — startPortForward()
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
```

`Setsid` (set session ID) places the subprocess in a **new session**. It is no longer part of the CLI's process group. The kernel's SIGHUP delivery on parent exit does not reach it.

This was also applied to cloudflared's `exec.Command` in `cloudflare.go` for the same reason.

---

## Bug 3: ERR_NAME_NOT_RESOLVED — URL returned before DNS is live

### Root cause

cloudflared prints the tunnel URL to its log approximately 1 second before the tunnel connection is fully registered with Cloudflare's edge. During that window, the subdomain has no DNS record. If the CLI returns the URL immediately after seeing it in the log, the user's browser gets `ERR_NAME_NOT_RESOLVED`.

### Fix

`tailLogForURL` uses a two-stage scan:

```go
func tailLogForURL(logPath string, urlCh chan<- string) {
    var foundURL string
    for {
        // scan new lines ...
        if foundURL == "" {
            if m := cloudflaredURL.FindString(line); m != "" {
                foundURL = m   // stage 1: capture URL
            }
        }
        // stage 2: wait for the stronger signal
        if foundURL != "" && strings.Contains(line, "Registered tunnel connection") {
            urlCh <- foundURL
            return
        }
    }
}
```

`"Registered tunnel connection"` appears in cloudflared's log once Cloudflare's edge has registered the tunnel and DNS propagation is complete. Only then is the URL sent back to `Expose`. The user receives a URL that works immediately in the browser.

The 30-second timeout in `Start` is measured from process launch to this confirmation, not from the URL announcement line.

---

## Checklist before modifying cloudflare.go or ngrok.go

- `cmd.Stderr` must be a **file** (`os.OpenFile`), not a pipe (`StderrPipe`). Files survive parent exit without SIGPIPE.
- `cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}` must be present on all subprocess `exec.Command` calls, including `startPortForward`.
- The URL must be returned only after confirming the tunnel connection is registered (for cloudflared: `"Registered tunnel connection"` line; for ngrok: the JSON `"url"` field implies the tunnel is live).
