# pkg/spinner

`spinner` renders an animated terminal progress indicator. It degrades gracefully on non-TTY outputs (CI logs, piped output) by printing a single line without animation.

```go
s := spinner.Start("Generating RBAC...")
// ... do work ...
s.Done("✅ RBAC generated")
// or on failure:
s.Fail("❌ generation failed")
```

On a TTY the spinner animates with braille frames (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`). On non-TTY it prints the message once and is immediately considered finalised — subsequent `Done` / `Fail` calls are no-ops.

Used by `ork generate`, `ork simulate`, and `ork doctor` to indicate long-running operations without leaving raw log output in interactive sessions.
