# Example 03 — Rollback and Ingress

Deploy a broken image, watch it fail, and restore the last good version in seconds.
Then expose your app at a real public URL using `--add-ingress`.

**What you learn:**

- How deploy state is recorded before every push
- `ork doctor deploy rollback` — instant image restore, no rebuild
- Exposing an app publicly with `--add-ingress` and a hostname

!!! note
    Complete [Example 01](../01-one-project/README.md) first.
    `my-api` must already be deployed with at least one successful deploy.

---

## Part A — Rollback

### Step 1 — Break the app intentionally

Edit `app/main.go` to crash on startup:

```go
func main() {
    panic("intentional crash to demonstrate rollback")
}
```

Commit and deploy:

```bash
cd my-api
git add . && git commit -m "intentional break"
ork doctor deploy --registry ghcr.io/myorg
```

Watch the deploy proceed normally through build and push.
When `kubectl rollout status` runs, the pods will crash-loop.
After 5 minutes (or sooner if logs show CrashLoopBackOff), Orkestra prints:

```
--- my-api-orkestra logs (last 20 lines) ---
panic: intentional crash to demonstrate rollback

goroutine 1 [running]:
...
---

  A previous good image is available.
  Roll back with: ork doctor deploy rollback

  ~ could not confirm readiness: timed out waiting for my-api-orkestra
```

---

### Step 2 — Roll back

```bash
ork doctor deploy rollback
```

Output:

```
Rolling back my-api...
  → ghcr.io/myorg/my-api:<previous-sha>

  ✓ Image set to ghcr.io/myorg/my-api:<previous-sha>

Waiting for rollback...
  ✓ Deployment ready

  Status: Ready
  Image:  ghcr.io/myorg/my-api:<previous-sha>
```

The previous image is read from `~/.orkestra/deploy/state.json`, which is updated
**before** every image patch — so rollback is always available instantly, even if
the cluster is unreachable.

---

### Step 3 — Verify in the Control Center

1. Open [http://localhost:8081](http://localhost:8081)
2. Click `my-api` → **View Resources** → click the CR
3. **Status** section: `phase: Ready` — the rollback succeeded
4. **Data** section: `image` shows the restored previous SHA
5. **Events** section: see the full timeline — deploy attempted, rollout failed,
   rollback patched, rollout completed

The Events tab is where you see the whole story without running a single `kubectl` command.

---

### Step 4 — Fix the app and redeploy

Revert the change:

```bash
git revert HEAD --no-edit
ork doctor deploy --registry ghcr.io/myorg
```

Rollback is reversible too — `ork doctor deploy rollback` immediately after a successful
rollback will re-roll-forward to the image you just rolled back from.

---

## Part B — Public URL with Ingress

### Step 5 — Add an Ingress to the backend

If `my-api` was initialised without `--add-ingress`, re-init:

```bash
ork doctor init --name my-api --add-ingress
```

!!! note
    Re-running `ork doctor init` regenerates `katalog.yaml` and `app.yaml`
    but will NOT overwrite `values.yaml` if it already exists.

Open `.orkestra/app.yaml` and set a hostname:

```yaml
data:
  host: "my-api.example.com"   # public hostname for this app
```

Deploy:

```bash
ork doctor deploy --registry ghcr.io/myorg
```

`ork doctor deploy` detects the frontend/ingress and installs an nginx ingress controller
automatically if one is not already present.

!!! note "kind cluster"
    On a kind cluster with the kind-specific nginx manifest, the Ingress will be
    reachable at `localhost` if you add `my-api.example.com` to your `/etc/hosts`:

    ```bash
    echo "127.0.0.1 my-api.example.com" | sudo tee -a /etc/hosts
    curl http://my-api.example.com/
    # Hello from my-api
    ```

---

### Step 6 — View the Ingress in the Control Center

1. Click `my-api` → **View Resources** → click the CR
2. **Children** section now shows a new row: **Ingress**
3. Click the Ingress to see its host, rules, and backend service

---

## Cleanup

```bash
chmod +x cleanup.sh && ./cleanup.sh 03
```

---

**← Previous** [02 — Frontend + backend](../02-frontend-backend/README.md) | **Next →** [04 — Notifications](../04-notify/README.md)
