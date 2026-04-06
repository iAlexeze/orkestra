// pkg/note/random.go
//
// Random generation notes — for use with once: true on secrets.
//
// IMPORTANT: These notes are pure in the mathematical sense — they produce
// random output. They are NOT idempotent. They MUST only be used in template
// sources that declare once: true (SecretTemplateSource.Once), which ensures
// the template is only evaluated once (when the Secret does not yet exist).
//
// Using random notes without once: true causes a new value to be generated
// on every reconcile cycle — passwords change every 30 seconds, breaking
// every application that uses them.
//
// Correct usage:
//
//	secrets:
//	  - name: "{{ .metadata.name }}-credentials"
//	    once: true                              # ← REQUIRED with random notes
//	    data:
//	      password: "{{ randomAlphanumeric 32 }}"
//	      apiKey:   "{{ randomHex 16 }}"
//
// The notes are registered in note.Map() and available in all template
// expressions — the once: guard is the semantic safeguard, not a code restriction.
package note

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
)

// charset for randomAlphanumeric
const alphanumCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomAlphanumeric returns a cryptographically random alphanumeric string
// of exactly n characters. Panics when the system random source is unavailable
// (should never happen in practice).
//
// Template: {{ randomAlphanumeric 32 }}
// Produces: "k7Xm3pQs9vR2nTwY8cL1jF6bH0dE4gA5"
//
// Use with once: true on secrets to generate stable passwords.
func randomAlphanumeric(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("note.randomAlphanumeric: system random source unavailable: " + err.Error())
	}
	result := make([]byte, n)
	charsetLen := byte(len(alphanumCharset))
	for i := range result {
		result[i] = alphanumCharset[b[i]%charsetLen]
	}
	return string(result)
}

// randomHex returns a cryptographically random hex-encoded string.
// n is the number of random bytes — the output string is 2n characters.
//
// Template: {{ randomHex 16 }}
// Produces: "a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5"  (32 hex chars from 16 bytes)
//
// Useful for: API keys, session tokens, CSRF tokens.
func randomHex(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("note.randomHex: system random source unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// randomBase64 returns a cryptographically random URL-safe base64 string.
// n is the number of random bytes — the output length is ceil(n*4/3).
//
// Template: {{ randomBase64 32 }}
// Produces: "k7Xm3pQs9vR2nTwY8cL1jF6bH0dE4gA5zP..."
//
// Useful for: JWT secrets, HMAC keys, cookie secrets.
// URL-safe encoding (uses - and _ instead of + and /) — safe in HTTP headers.
func randomBase64(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic("note.randomBase64: system random source unavailable: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// randomNotes returns the FuncMap entries for random notes.
// Called by note.Map() to register them with the template engine.
func randomNotes() map[string]interface{} {
	return map[string]interface{}{
		// randomAlphanumeric n — returns n random alphanumeric characters
		"randomAlphanumeric": randomAlphanumeric,

		// randomHex n — returns 2n hex characters (n random bytes)
		"randomHex": randomHex,

		// randomBase64 n — returns URL-safe base64 from n random bytes
		"randomBase64": randomBase64,
	}
}
