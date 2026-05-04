# ork deploy rollback

Instantly restore the previous deployed image by patching the ConfigMap CR.
No Docker build, no push, no bundle regeneration.

```bash
ork deploy rollback [app-name] [flags]
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--image` | previous image | Specific image to restore (skips state/annotation lookup) |

---

## Usage

### Roll back to the previous image

```bash
ork deploy rollback
```

Reads the previous image from `~/.orkestra/deploy/state.json`, patches the
ConfigMap CR, and watches the rollout.

### Roll back a specific app from outside its directory

```bash
ork deploy rollback my-api
```

### Roll back to a specific image

```bash
ork deploy rollback --image ghcr.io/myorg/my-api:d1e2f30
```

Bypasses all state lookups and patches directly to the given image.

---

## How it finds the previous image

1. `~/.orkestra/deploy/state.json` — checked first; the previous image is recorded
   there before every deploy, so this is always accurate when available
2. `orkestra.io/previous-image` annotation on the ConfigMap CR — fallback for
   environments where state.json was not written or was cleared

If neither source has a previous image, rollback fails with:

```
no previous image found for my-api — use --image to specify
```

---

## State swap

Before patching the cluster, rollback swaps `currentImage` and `previousImage` in
state.json. This means a second `ork deploy rollback` immediately after the first will
re-roll-forward — each rollback is reversible.

The annotation is updated with the same swap logic so both sources stay consistent.

---

## Watch behaviour

After patching, rollback watches the rollout with the same logic as `ork deploy`:

```bash
kubectl rollout status deployment/<cr-name> -n <namespace> --timeout=5m
```

On success it prints the ready summary. On failure it shows pod logs and notes that
there is no previous-good image to roll back to from this state (since the intended
previous image just failed).

---

## When to use rollback

| Situation | Recommended action |
|-----------|-------------------|
| Replicas stuck in `Deploying` for more than a few minutes | `ork deploy rollback` |
| Pod crash loop after a deploy | `ork deploy rollback` |
| Orkestra sends a deployment-not-ready notification | `ork deploy rollback` |
| Bad config pushed with the image | Fix `.env` or `app.yaml`, then `ork deploy` |
| First deploy that has never succeeded | Fix the Dockerfile or config — no previous image exists |

---

## Related

- `ork deploy` records state before every patch — [deploy docs](./deploy.md)
- Notifications that suggest rollback — [notification docs](../../../notification/docs/04-developer-notifications.md)

---

**← Previous** [ork deploy](./deploy.md) | **Back to index →** [Developer CLI](./__index.md)
