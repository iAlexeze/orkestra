package intake

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func signSlack(secret, timestamp string, body []byte) string {
	base := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(base))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyHMACSHA256_Valid(t *testing.T) {
	body := []byte(`{"target":"app"}`)
	sig := signBody("s3cr3t", body)
	if !verifyHMACSHA256("s3cr3t", body, sig) {
		t.Error("valid signature rejected")
	}
}

func TestVerifyHMACSHA256_WrongSecret(t *testing.T) {
	body := []byte(`{"target":"app"}`)
	sig := signBody("s3cr3t", body)
	if verifyHMACSHA256("wrong-secret", body, sig) {
		t.Error("signature verified against the wrong secret")
	}
}

func TestVerifyHMACSHA256_TamperedBody(t *testing.T) {
	sig := signBody("s3cr3t", []byte(`{"target":"app"}`))
	if verifyHMACSHA256("s3cr3t", []byte(`{"target":"app","extra":"field"}`), sig) {
		t.Error("signature verified against a tampered body")
	}
}

func TestVerifyHMACSHA256_MissingPrefix(t *testing.T) {
	body := []byte(`{"target":"app"}`)
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write(body)
	rawHex := hex.EncodeToString(mac.Sum(nil)) // no "sha256=" prefix
	if verifyHMACSHA256("s3cr3t", body, rawHex) {
		t.Error("signature without the sha256= prefix should be rejected")
	}
}

func TestVerifyHMACSHA256_MalformedHex(t *testing.T) {
	if verifyHMACSHA256("s3cr3t", []byte("body"), "sha256=not-hex") {
		t.Error("malformed hex signature should be rejected")
	}
}

func TestVerifySlackSignature_Valid(t *testing.T) {
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte("command=/deploy&text=app+repository=myorg/app")
	sig := signSlack("s3cr3t", ts, body)
	if !verifySlackSignature("s3cr3t", []byte(ts), body, sig, now) {
		t.Error("valid signature rejected")
	}
}

func TestVerifySlackSignature_WrongSecret(t *testing.T) {
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte("command=/deploy")
	sig := signSlack("s3cr3t", ts, body)
	if verifySlackSignature("wrong-secret", []byte(ts), body, sig, now) {
		t.Error("signature verified against the wrong secret")
	}
}

func TestVerifySlackSignature_ReplayedOldTimestamp(t *testing.T) {
	now := time.Now()
	old := now.Add(-10 * time.Minute) // outside the 5-minute window
	ts := strconv.FormatInt(old.Unix(), 10)
	body := []byte("command=/deploy")
	sig := signSlack("s3cr3t", ts, body)
	if verifySlackSignature("s3cr3t", []byte(ts), body, sig, now) {
		t.Error("a signature outside the replay window should be rejected")
	}
}

func TestVerifySlackSignature_FutureTimestamp(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	ts := strconv.FormatInt(future.Unix(), 10)
	body := []byte("command=/deploy")
	sig := signSlack("s3cr3t", ts, body)
	if verifySlackSignature("s3cr3t", []byte(ts), body, sig, now) {
		t.Error("a timestamp far in the future should be rejected too")
	}
}

func TestVerifySlackSignature_MalformedTimestamp(t *testing.T) {
	now := time.Now()
	body := []byte("command=/deploy")
	if verifySlackSignature("s3cr3t", []byte("not-a-number"), body, "v0=abcd", now) {
		t.Error("malformed timestamp should be rejected")
	}
}

func TestVerifySlackSignature_MissingPrefix(t *testing.T) {
	now := time.Now()
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte("command=/deploy")
	base := fmt.Sprintf("v0:%s:%s", ts, body)
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write([]byte(base))
	rawHex := hex.EncodeToString(mac.Sum(nil)) // no "v0=" prefix
	if verifySlackSignature("s3cr3t", []byte(ts), body, rawHex, now) {
		t.Error("signature without the v0= prefix should be rejected")
	}
}

func TestVerifyStaticToken(t *testing.T) {
	if !verifyStaticToken("s3cr3t", "s3cr3t") {
		t.Error("matching token rejected")
	}
	if verifyStaticToken("s3cr3t", "wrong") {
		t.Error("mismatched token accepted")
	}
	if verifyStaticToken("", "") {
		t.Error("two empty strings should not verify")
	}
	if verifyStaticToken("s3cr3t", "") {
		t.Error("empty token should not verify against a real secret")
	}
}
