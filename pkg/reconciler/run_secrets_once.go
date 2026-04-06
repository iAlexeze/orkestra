// pkg/reconciler/run_secrets_once.go
//
// once: true on secrets — idempotent random secret generation.
//
// The Kubernetes reconcile loop is called on every resync. A secret with
// random data (password, API key) would change every 30 seconds without
// the once: guard. once: true makes the create step check for existence
// first — if the Secret already exists, skip template evaluation entirely.
//
// This is the check-before-generate pattern used by cert-manager and every
// other tool that generates credentials idempotently.
//
// YAML:
//
//	secrets:
//	  - name: "{{ .metadata.name }}-credentials"
//	    once: true
//	    data:
//	      password: "{{ randomAlphanumeric 32 }}"
//	      apiKey:   "{{ randomHex 16 }}"
//	      jwtSecret: "{{ randomBase64 32 }}"
//
// Behaviour:
//   - First reconcile: Secret does not exist → evaluate templates → create
//   - Every subsequent reconcile: Secret exists → skip completely (no-op)
//   - CR deleted: Secret deleted via owner reference (garbage collection)
//
// once: true is IGNORED when update=true (onReconcile with reconcile: true).
// A secret in onReconcile should use static or deterministic template values.
// Combining once: true with reconcile: true logs a warning and skips the
// once: guard — the caller opted into continuous reconciliation.
//
// once: false (default): standard create/update behavior, unchanged.
package reconciler

import (
	"context"
	"fmt"

	"github.com/ialexeze/orkestra/pkg/kubeclient"
	"github.com/ialexeze/orkestra/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// secretExists checks whether a Secret with the given name exists in the namespace.
// Uses ResourceVersion: "0" to read from the API server watch cache (not etcd).
// Returns true if the secret exists, false on NotFound, error on API failure.
func secretExists(ctx context.Context, kube *kubeclient.Kubeclient, namespace, name string) (bool, error) {
	_, err := kube.Clientset().CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{
		ResourceVersion: "0", // watch cache — avoids etcd round-trip
	})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking secret %s/%s: %w", namespace, name, err)
	}
	return true, nil
}

// isNotFoundErr returns true when err is a Kubernetes 404 Not Found error.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	// k8s.io/apimachinery/pkg/api/errors.IsNotFound — check by message
	// Avoids importing the errors package here to keep the file focused.
	return containsStr404(err.Error())
}

func containsStr404(s string) bool {
	return len(s) >= 9 && (containsSubstr(s, "not found") || containsSubstr(s, "NotFound"))
}

func containsSubstr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// logOnceReconcileConflict warns when once: true and reconcile: true are
// both set on the same secret. These are contradictory: once means
// "never re-evaluate", reconcile means "always re-evaluate".
func logOnceReconcileConflict(ctx context.Context, name string) {
	logger.FromContext(ctx).Warn().
		Str("secret", name).
		Msg("secret has both once: true and reconcile: true — " +
			"once: is ignored. Remove reconcile: true to enable once: semantics, " +
			"or remove once: true if you want continuous reconciliation.")
}
