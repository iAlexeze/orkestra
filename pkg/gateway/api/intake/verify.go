package intake

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// slackReplayWindow is how old a signed Slack request is allowed to be.
// Slack's own docs recommend 5 minutes to guard against replay attacks.
// https://api.slack.com/authentication/verifying-requests-from-slack
const slackReplayWindow = 5 * time.Minute

// verifyHMACSHA256 reports whether signatureHeader is a valid HMAC-SHA256
// signature of body under secret. signatureHeader is expected in
// "sha256=<hex>" form — the same convention GitHub uses for
// X-Hub-Signature-256, reused here for the generic source too so callers
// don't need to learn a second scheme.
// https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
//
// Constant-time comparison — a timing side channel here would let an
// attacker recover the signature byte by byte.
func verifyHMACSHA256(secret string, body []byte, signatureHeader string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signatureHeader, prefix))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return subtle.ConstantTimeCompare(sig, expected) == 1
}

// verifyStaticToken reports whether token equals secret, compared in
// constant time. GitLab's default webhook auth (X-Gitlab-Token) is a
// shared-secret comparison, not a signature over the body.
// https://docs.gitlab.com/user/project/integrations/webhooks/#validate-payloads-by-using-a-secret-token
func verifyStaticToken(secret, token string) bool {
	if secret == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}

// verifySlackSignature reports whether signature is a valid Slack request
// signature for body under secret, at the given timestamp. Slack signs
// "v0:{timestamp}:{body}" with HMAC-SHA256, prefixed "v0=" in the header —
// a different scheme from GitHub's plain HMAC-over-body, because Slack also
// binds the timestamp into what's signed, closing the replay window
// checked below.
// https://api.slack.com/authentication/verifying-requests-from-slack
//
// now is passed in (not read internally) so tests can control it instead of
// racing the wall clock at slackReplayWindow's edges.
func verifySlackSignature(secret string, timestamp, body []byte, signature string, now time.Time) bool {
	ts, err := strconv.ParseInt(string(timestamp), 10, 64)
	if err != nil {
		return false
	}
	age := now.Sub(time.Unix(ts, 0))
	if math.Abs(age.Seconds()) > slackReplayWindow.Seconds() {
		return false
	}

	const prefix = "v0="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return false
	}

	base := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	expected := mac.Sum(nil)

	return subtle.ConstantTimeCompare(sig, expected) == 1
}
