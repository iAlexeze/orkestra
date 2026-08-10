# Gateway Webhook Intake — GitHub, GitLab, Slack, Generic

**Four delivery surfaces, one apply pipeline.** A GitHub push, a GitLab
push, a Slack slash command, and a generic JSON webhook (PagerDuty-shaped)
all resolve through the exact same target-mode path a direct
`POST /api/v1/apply` call does. This example wires up all four against one
`ServiceRequest` CRD so you can see the differences side by side.

---

## How the four sources differ

| | GitHub | GitLab | Slack | Generic |
|---|---|---|---|---|
| Trigger | push to `branch` | push to `branch` | `/deploy ...` slash command | any JSON POST |
| Verifies via | `X-Hub-Signature-256` (HMAC) | `X-Gitlab-Token` (static) | `X-Slack-Signature` (HMAC, ±5min) | `X-Signature-256` (HMAC) |
| Payload → intent | fetch changed file via Contents API | fetch changed file via Repository Files API | parse `"<target> key=value ..."` | body is the field map directly |
| Response | ack, per-file result in body | ack, per-file result in body | ack in 3s, apply in background | apply result directly |
| Extra credential | `contentTokenRef` (read file content) | `contentTokenRef` (read file content) | none | none |

Every one of them ends at the same target/alias
resolution, same `serve.tokens` permission check, same provenance stamping,
same SSA patch. The source only decides how the flat field map gets built.

---

## 1. Install the ork CLI

```bash
curl get.orkestra.sh | bash
ork version
```

## 2. Validate

```bash
ork serve validate
ork serve validate --full
```

`ork validate` (run automatically by the above) enforces webhook-specific
rules statically — every entry needs a name, a unique path, the right
credential(s) for its source, and `gateway.api.enabled: true`. Try
disabling `gateway.api.enabled` in `katalog.yaml` to see the rejection.

## 2b. Test locally before touching a cluster

`ork webhook list` shows every configured entry:

```bash
ork webhook list -f katalog.yaml
```

`ork webhook play` runs a simulated payload through the exact same chain
every real delivery does — target resolution, token check, CR construction,
provenance stamping, admission validation — no cluster, no HTTP server, no
real GitHub/GitLab/Slack account required. Signature/token verification is
skipped; this tests the logic each source applies to its payload (branch
filtering, watch-pattern matching, command parsing), not the transport:

```bash
# generic
ork webhook play --source generic -f katalog.yaml --webhook pagerduty \
  --body repo-example/services/payments/intent.yaml

# slack
ork webhook play --source slack -f katalog.yaml --webhook platform-workspace \
  --command /deploy --text "servicerequest name=payments-api team=platform image=myorg/payments:1.4.2"

# github — --fetch simulates what the Contents API would have returned for
# each matched changed file, since there's no real repo to call in play mode
ork webhook play --source github -f katalog.yaml --webhook payments-repo \
  --event repo-example/github-push-event.json \
  --fetch services/payments/intent.yaml=repo-example/services/payments/intent.yaml

# gitlab
ork webhook play --source gitlab -f katalog.yaml --webhook payments-repo-gitlab \
  --event repo-example/gitlab-push-event.json \
  --fetch services/payments/intent.yaml=repo-example/services/payments/intent.yaml
```

Add `--simulate` to any of these to hand the built CR to `ork simulate`
afterward — the full path from payload to a reconciled resource, still with
no cluster:

```bash
ork webhook play --source github -f katalog.yaml --webhook payments-repo \
  --event repo-example/github-push-event.json \
  --fetch services/payments/intent.yaml=repo-example/services/payments/intent.yaml \
  --simulate
```

This catches a wrong `watch:` pattern, a branch mismatch, or a
`serve.tokens` misconfiguration in seconds — all three are checked in the
play chain's early stages, before anything touches a real repo's webhook
settings. The `curl`-based steps under "Set up each source" below are still
worth knowing — they're what actually happens on the wire — but `ork
webhook play` is the faster loop while you're getting the config right.

## 3. Cluster + CRD

```bash
ork create cluster        # skip if you already have one
kubectl apply -f crd.yaml
```

## 4. Generate and apply the operator bundle

```bash
ork generate bundle -f katalog.yaml -o bundle.yaml
kubectl apply -f bundle.yaml
```

## 5. Install Orkestra with the Gateway enabled

```bash
helm repo add orkestra https://orkspace.github.io/orkestra
helm upgrade --install orkestra orkestra/orkestra \
  --namespace orkestra-system \
  --create-namespace \
  -f values.yaml \
  --wait --timeout 120s
```

`secretRef` / `signingSecretRef` (GitHub's, GitLab's, Slack's, and
PagerDuty's webhook-verification secrets) self-bootstrap on first startup —
the gateway generates and stores each one if it doesn't already exist. Read
it back out to hand to the real GitHub/GitLab/Slack/PagerDuty side:

```bash
kubectl get secret ork-payments-github-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d
kubectl get secret ork-slack-signing-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d
kubectl get secret ork-pagerduty-webhook-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d
```

## 6. Create the content-read tokens (GitHub + GitLab only)

`contentTokenRef` is **not** self-bootstrapped — it's a credential minted
by the external system (a GitHub App installation token, a GitLab personal
or project access token with `read_repository`), not something the gateway
can invent. Create these two Secrets yourself before pushing anything:

```bash
kubectl create secret generic ork-payments-github-app-token \
  -n orkestra-system --from-literal=token="$GITHUB_CONTENT_TOKEN"

kubectl create secret generic ork-payments-gitlab-api-token \
  -n orkestra-system --from-literal=token="$GITLAB_CONTENT_TOKEN"
```

---

## 7. Set up each source

### Generic (PagerDuty-shaped)

No repo, no app registration — just point any system that can POST JSON
and sign it at the endpoint:

```bash
ork proxy --for gateway   # in another terminal

SECRET=$(kubectl get secret ork-pagerduty-webhook-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d)
BODY='{"target":"servicerequest","name":"payments-api","team":"platform","image":"myorg/payments:1.4.2"}'
SIG="sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')"

curl -X POST http://localhost:8080/webhooks/generic/pagerduty \
  -H "X-Signature-256: $SIG" -H "Content-Type: application/json" -d "$BODY"
```

### Slack

1. Create a Slack app (api.slack.com/apps) → **Slash Commands** → New
   Command → `/deploy`, Request URL `https://<your-gateway>/webhooks/slack`.
2. **Basic Information** → copy the **Signing Secret** →
   `kubectl create secret generic ork-slack-signing-secret -n orkestra-system --from-literal=secret="$SLACK_SIGNING_SECRET" --dry-run=client -o yaml | kubectl apply -f -`
   (or let it self-bootstrap and paste the generated value into the Slack
   app config instead — either direction works, they just need to match).
3. In Slack: `/deploy servicerequest name=payments-api team=platform image=myorg/payments:1.4.2`

Local test without a real Slack app:

```bash
SECRET=$(kubectl get secret ork-slack-signing-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d)
TS=$(date +%s)
BODY='command=%2Fdeploy&text=servicerequest+name%3Dpayments-api+team%3Dplatform+image%3Dmyorg%2Fpayments%3A1.4.2&response_url=https://example.test/response'
SIG="v0=$(printf 'v0:%s:%s' "$TS" "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')"

curl -X POST http://localhost:8080/webhooks/slack \
  -H "X-Slack-Signature: $SIG" -H "X-Slack-Request-Timestamp: $TS" \
  -H "Content-Type: application/x-www-form-urlencoded" -d "$BODY"
```

The ack comes back in under 3 seconds ("Deploying servicerequest...");
the apply itself runs in the background and posts the outcome to
`response_url` — `response_url=https://example.test/response` above won't
actually receive it without a real Slack app, but the ack, signature
verification, and command parsing all exercise for real.

### GitHub

1. In the watched repo: **Settings → Webhooks → Add webhook**
   - Payload URL: `https://<your-gateway>/webhooks/github/payments`
   - Content type: `application/json`
   - Secret: the value from `ork-payments-github-secret`
   - Events: **Just the push event**
2. Content token: a fine-grained PAT or GitHub App installation token with
   `contents: read` on the repo → the Secret from step 6 above.
3. Push a change to `services/payments/intent.yaml` on `main` (see
   `repo-example/services/payments/intent.yaml` in this directory for the
   file shape) — the gateway fetches it via the Contents API and applies it.

Local test without a real repo (exercises signature verification, branch
filtering, and watch-pattern matching in full; the content fetch itself
will fail against a repo that doesn't exist — that's expected, and the
per-file result reports it instead of failing the whole request):

```bash
SECRET=$(kubectl get secret ork-payments-github-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d)
BODY='{"ref":"refs/heads/main","repository":{"name":"payments","owner":{"login":"myorg"}},"commits":[{"added":["services/payments/intent.yaml"],"modified":[]}],"after":"abc123"}'
SIG="sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')"

curl -X POST http://localhost:8080/webhooks/github/payments \
  -H "X-Hub-Signature-256: $SIG" -H "Content-Type: application/json" -d "$BODY"
```

### GitLab

1. In the watched project: **Settings → Webhooks**
   - URL: `https://<your-gateway>/webhooks/gitlab/payments`
   - Secret token: the value from `ork-payments-gitlab-secret`
   - Trigger: **Push events**, branch filter `main`
2. Content token: a project or personal access token with `read_api` →
   the Secret from step 6 above.
3. Push a change to `services/payments/intent.yaml` on `main`.

Local test without a real project:

```bash
TOKEN=$(kubectl get secret ork-payments-gitlab-secret -n orkestra-system -o jsonpath='{.data.secret}' | base64 -d)
BODY='{"ref":"refs/heads/main","checkout_sha":"abc123","project":{"id":123},"commits":[{"added":["services/payments/intent.yaml"],"modified":[]}]}'

curl -X POST http://localhost:8080/webhooks/gitlab/payments \
  -H "X-Gitlab-Token: $TOKEN" -H "Content-Type: application/json" -d "$BODY"
```

---

## 8. Verify

```bash
kubectl get servicerequests.demo.orkestra.io
```

```text
NAME           TEAM       IMAGE                        SOURCE           AGE
payments-api   platform   myorg/payments:1.4.2         payments-repo    12s
```

The `SOURCE` column is the stamped `serve-source` provenance annotation —
the webhook entry's own `Name`. Every apply through this Katalog, no matter
which of the four surfaces it came through, is traceable back to exactly
which entry delivered it.

```bash
kubectl get servicerequest payments-api -o jsonpath='{.metadata.annotations}' | jq
```

shows all three provenance annotations (`serve-target`, `serve-alias`,
`serve-source`) plus the raw `serve-intent` JSON — what the caller actually
submitted, available as `.request.*` in `validation.rules`/`mutation.rules`.

---

## Cleanup

```bash
./cleanup.sh
```

---

## Files

| File | Purpose |
|------|---------|
| `crd.yaml` | Single-version ServiceRequest CRD |
| `katalog.yaml` | Operator with `serve` config + all four `gateway.webhooks` sources |
| `serve/servicerequest.yaml` | Serve field config — included by `katalog.yaml` |
| `repo-example/services/payments/intent.yaml` | Sample file as it would exist in the *watched* GitHub/GitLab repo — not part of this Katalog itself |
| `values.yaml` | Helm values — `gateway.enabled` + the dev token |
| `cleanup.sh` | Removes the CR, CRD, and every self-bootstrapped Secret |
| `README.md` | This file |
