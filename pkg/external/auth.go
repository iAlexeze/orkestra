package external

import (
	"context"
	"fmt"

	"github.com/orkspace/orkestra/pkg/secrets"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// resolveAuth resolves the credential value from an ExternalAuth declaration.
// Returns the credential string and the header name to inject it into (HTTP only).
// header defaults to "Authorization"; produces "Bearer <credential>".
// cs may be nil when secretRef is not used.
// secretRef.namespace is required — validated at ork validate time, never defaulted here.
func resolveAuth(ctx context.Context, auth *orktypes.ExternalAuth, cs kubernetes.Interface) (credential, header string, err error) {
	if auth == nil {
		return "", "", nil
	}

	header = auth.Header
	if header == "" {
		header = "Authorization"
	}

	switch {
	case auth.SecretRef != nil:
		credential, err = secrets.ReadSecretKey(ctx, cs, auth.SecretRef.Namespace, auth.SecretRef.Name, auth.SecretRef.Key)
		if err != nil {
			return "", "", fmt.Errorf("auth.secretRef: %w", err)
		}

	case auth.Env != "":
		credential = ExpandEnv(auth.Env)
		if credential == "" {
			return "", "", fmt.Errorf("auth.env: %q is not set or empty", auth.Env)
		}

	default:
		return "", "", fmt.Errorf("auth: exactly one of secretRef or env must be set")
	}

	return credential, header, nil
}
