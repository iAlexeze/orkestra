# ork control

Manage the Orkestra Control Center (CC).

```bash
ork control [command]
```

The Control Center provides a web‑based UI for monitoring multiple Orkestra runtime instances.

---

## Commands

### start

Start the Orkestra Control Center.

```bash
ork control [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `-p, --port <port>` | Port to run the Control Center on (default: `8081`) |
| `-u, --urls <list>` | Comma‑separated list of Orkestra runtime URLs (default: `http://localhost:8080`) |
| `--refresh <duration>` | Auto‑refresh interval for fetching Katalogs (default: `10s`) |
| `--log-level <level>` | Log level (`debug`, `info`, `warn`, `error`) |
| `-i, --ignore-default` | Do not add the default `localhost:8080` instance |

#### Examples

Start with defaults:

```bash
ork control
```

Custom port and multiple instances:

```bash
ork control --port 9090 --urls "http://localhost:8080,http://localhost:8082"
```

Debug logging:

```bash
ork control --log-level debug --refresh 5s
```

Remote runtimes:

```bash
ork control --urls "https://prod.orkestra:8080,https://staging.orkestra:8080"
```

---

### version

Show the installed Control Center version.

```bash
ork control version
```

---

## Behavior

- `ork control` locates the `orkcc` binary in:
  - `$PATH`
  - next to the `ork` binary
  - `~/.orkestra/bin/`
- If not found, Orkestra attempts installation (placeholder logic until implemented).
- The Control Center runs as a standalone binary (`orkcc`) and takes over the terminal.

---

## Notes

- `ork control` does not interact with Kubernetes directly.
- It only manages the local Control Center binary.
- Runtime URLs must point to running Orkestra runtime instances (`ork run` or deployed operators).
