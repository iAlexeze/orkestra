// Package cosign provides Cosign keyless signing and verification for Orkestra
// OCI artifacts. It uses the cosign CLI binary — checking PATH first, then
// downloading the latest release and caching it at ~/.orkestra/tools/cosign.
//
// No separate install step required: the first sign or verify call that needs
// cosign fetches it automatically.
package cosign

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	cosignVersion    = "v2.4.3"
	cosignGitHubRepo = "sigstore/cosign"
	cosignCacheDir   = ".orkestra/tools"
	cosignBinaryName = "cosign"
)

// SignOptions configures a Sign call.
type SignOptions struct {
	// SkipConfirmation suppresses the interactive "Are you sure?" prompt.
	// Always true in CI / ork push --sign.
	SkipConfirmation bool

	// IgnoreTlog skips uploading the signature entry to the Rekor transparency
	// log. Useful for ephemeral artifacts (e.g. ttl.sh) where a permanent log
	// entry makes no sense.
	IgnoreTlog bool
}

// VerifyOptions configures a Verify call.
type VerifyOptions struct {
	// ExpectedIdentities is the list of trusted OIDC subject claims.
	// Any one match passes. Empty means any valid keyless signature is accepted.
	ExpectedIdentities []string

	// IgnoreTlog skips Rekor transparency log verification.
	IgnoreTlog bool

	// Output receives cosign's verification output. Defaults to io.Discard.
	Output io.Writer
}

// Sign signs an OCI artifact at ref using Cosign keyless signing.
func Sign(ctx context.Context, ref string, opts SignOptions) error {
	bin, err := resolveBinary(ctx)
	if err != nil {
		return err
	}

	// cosign expects standard registry format — strip oci:// scheme if present.
	cosignRef := strings.TrimPrefix(ref, "oci://")

	args := []string{"sign"}
	if opts.SkipConfirmation {
		args = append(args, "--yes")
	}
	if opts.IgnoreTlog {
		args = append(args, "--tlog-upload=false")
	}
	args = append(args, cosignRef)

	return runCosign(ctx, bin, args, nil)
}

// Verify verifies the Cosign keyless signature on an OCI artifact at ref.
func Verify(ctx context.Context, ref string, opts VerifyOptions) error {
	bin, err := resolveBinary(ctx)
	if err != nil {
		return err
	}

	out := opts.Output
	if out == nil {
		out = io.Discard
	}

	// cosign expects standard registry format — strip oci:// scheme if present.
	cosignRef := strings.TrimPrefix(ref, "oci://")

	switch len(opts.ExpectedIdentities) {
	case 0:
		// Any valid keyless signature accepted.
		args := buildVerifyArgs(cosignRef, "", opts.IgnoreTlog)
		args = append(args, "--certificate-identity-regexp", ".*", "--certificate-oidc-issuer-regexp", ".*")
		return runCosign(ctx, bin, args, out)
	case 1:
		id := opts.ExpectedIdentities[0]
		args := buildVerifyArgs(cosignRef, id, opts.IgnoreTlog)
		args = append(args, "--certificate-oidc-issuer", extractIssuer(id))
		return runCosign(ctx, bin, args, out)
	default:
		// Multiple identities — try each; return nil on first success.
		for _, id := range opts.ExpectedIdentities {
			args := buildVerifyArgs(cosignRef, id, opts.IgnoreTlog)
			args = append(args, "--certificate-oidc-issuer", extractIssuer(id))
			if runCosign(ctx, bin, args, io.Discard) == nil {
				return nil
			}
		}
		return fmt.Errorf("cosign verify %s: none of the expected identities matched", ref)
	}
}

func buildVerifyArgs(ref, identity string, ignoreTlog bool) []string {
	args := []string{"verify"}
	if identity != "" {
		args = append(args, "--certificate-identity", identity)
	}
	if ignoreTlog {
		args = append(args, "--insecure-ignore-tlog")
	}
	args = append(args, ref)
	return args
}

func runCosign(ctx context.Context, bin string, args []string, out io.Writer) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	if out != nil {
		cmd.Stdout = out
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cosign %s: %w\n%s", args[0], err, stderr.String())
	}
	return nil
}

// resolveBinary returns the path to the cosign binary.
// Checks PATH first; downloads and caches if not found.
func resolveBinary(ctx context.Context) (string, error) {
	if path, err := exec.LookPath(cosignBinaryName); err == nil {
		return path, nil
	}

	cached, err := cachedBinaryPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}

	return downloadBinary(ctx, cached)
}

// cachedBinaryPath returns the path where the cosign binary is cached.
func cachedBinaryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	name := cosignBinaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home, cosignCacheDir, name), nil
}

// downloadBinary downloads the cosign binary for the current OS/arch and caches it.
func downloadBinary(ctx context.Context, dest string) (string, error) {
	url := cosignDownloadURL()

	fmt.Fprintf(os.Stderr, "cosign not found — downloading %s to %s\n", cosignVersion, dest)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("creating tools dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("cosign download: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cosign download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cosign download: unexpected status %d from %s", resp.StatusCode, url)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", fmt.Errorf("cosign download: creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("cosign download: writing binary: %w", err)
	}

	fmt.Fprintf(os.Stderr, "cosign downloaded to %s\n", dest)
	return dest, nil
}

// cosignDownloadURL returns the GitHub release download URL for the current OS/arch.
func cosignDownloadURL() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Cosign uses amd64/arm64 naming directly.
	name := fmt.Sprintf("cosign-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}

	return fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/%s",
		cosignGitHubRepo, cosignVersion, name,
	)
}

// extractIssuer infers the OIDC issuer from a subject claim prefix.
func extractIssuer(identity string) string {
	switch {
	case strings.HasPrefix(identity, "github.com/"):
		return "https://token.actions.githubusercontent.com"
	case strings.HasPrefix(identity, "gitlab.com/"):
		return "https://gitlab.com"
	default:
		return "https://oauth2.sigstore.dev/auth"
	}
}
