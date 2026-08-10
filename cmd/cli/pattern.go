//go:build !runtime && !gateway

package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orkspace/orkestra/pkg/registry"
	orksigning "github.com/orkspace/orkestra/pkg/signing/cosign"
	"github.com/spf13/cobra"
)

// ── pattern ───────────────────────────────────────────────────────────────────

var patternCmd = &cobra.Command{
	Use:   "pattern",
	Short: "Manage a pattern artifact",
}

var patternSignCmd = &cobra.Command{
	Use:   "sign <name>:<version>",
	Short: "Sign a pushed pattern with Cosign keyless",
	Args:  cobra.ExactArgs(1),
	Example: `  ork pattern sign postgres:v14
  ork pattern sign oci://ghcr.io/myorg/patterns/postgres:v14
  ork pattern sign postgres:v14 --local --dir ./patterns/postgres/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		local, _ := cmd.Flags().GetBool("local")
		ttl, _ := cmd.Flags().GetString("ttl")
		noTlog, _ := cmd.Flags().GetBool("no-tlog")

		if local {
			dirArg, _ := cmd.Flags().GetString("dir")
			if dirArg == "" {
				dirArg = "."
			}
			dir, err := filepath.Abs(dirArg)
			if err != nil {
				return err
			}
			ref, err := buildTTLRef(basePatternName(args[0]), ttl)
			if err != nil {
				return fmt.Errorf("constructing ttl.sh ref: %w", err)
			}
			client, err := registry.NewClient()
			if err != nil {
				return fmt.Errorf("initializing client: %w", err)
			}
			spin := StartSpinner(fmt.Sprintf("Pushing to %s...", ref.String()))
			if _, err := client.Push(cmd.Context(), ref, dir, registry.PushOptions{}, nil); err != nil {
				spin.Failure()
				return fmt.Errorf("push failed: %w", err)
			}
			spin.Stop()
			fmt.Printf("%s Pushed:  %s\n", successMark(), ref.String())
			return signLocal(cmd.Context(), ref, ttl)
		}

		ref, err := registry.ResolveForKind(args[0], registry.KatalogKind)
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		spin := StartSpinner("Signing...")
		if err := signPatternRef(cmd.Context(), ref.String(), noTlog); err != nil {
			spin.Failure()
			return fmt.Errorf("signing failed: %w", err)
		}
		spin.Stop()
		fmt.Printf("%s Signed:  %s\n", successMark(), ref.String())
		return nil
	},
}

var patternVerifyCmd = &cobra.Command{
	Use:   "verify <name>:<version>",
	Short: "Verify the Cosign keyless signature on a pattern",
	Args:  cobra.ExactArgs(1),
	Example: `  ork pattern verify postgres:v14
  ork pattern verify postgres:v14 --verbose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := registry.ResolveForKind(args[0], registry.KatalogKind)
		if err != nil {
			return fmt.Errorf("invalid reference: %w", err)
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		noTlog, _ := cmd.Flags().GetBool("no-tlog")

		var out strings.Builder
		spin := StartSpinner("Verifying signature...")
		verifyErr := verifyPatternRef(cmd.Context(), ref.String(), nil, noTlog, &out)
		spin.Stop()

		if verifyErr != nil {
			fmt.Printf("%s Not verified\n", failureMark())
			if verbose {
				for _, line := range strings.Split(strings.TrimSpace(verifyErr.Error()), "\n") {
					fmt.Printf("  %s\n", line)
				}
			}
			return verifyErr
		}

		subject := extractSubjectFromCosignOutput(out.String())
		issuer := extractIssuerFromCosignOutput(out.String())
		fmt.Printf("%s Verified (keyless)\n", successMark())
		if subject != "" {
			fmt.Printf("  subject:  %s\n", subject)
		}
		if issuer != "" {
			fmt.Printf("  issuer:   %s\n", issuer)
		}
		if verbose && out.Len() > 0 {
			fmt.Println()
			for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
				fmt.Printf("  %s\n", line)
			}
		}
		return nil
	},
}

// basePatternName extracts the bare pattern name from any ref format:
//
//	deployment-stack:v1                              → deployment-stack
//	oci://ttl.sh/ork-e31104/deployment-stack:24h    → deployment-stack
//	ghcr.io/myorg/patterns/postgres:v14             → postgres
func basePatternName(ref string) string {
	name := strings.TrimPrefix(ref, "oci://")
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, ":"); idx != -1 {
		name = name[:idx]
	}
	return name
}

// randomHex returns n random bytes as a hex string (2n chars).
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// buildTTLRef constructs a ttl.sh OCI ref for local signing tests.
// Format: oci://ttl.sh/ork-<slug>/<name>:<ttl>
func buildTTLRef(name, ttl string) (*registry.Ref, error) {
	slug := randomHex(3) // 6 hex chars
	return registry.ResolveForKind(
		fmt.Sprintf("oci://ttl.sh/ork-%s/%s:%s", slug, name, ttl),
		registry.KatalogKind,
	)
}

// signLocal signs a ttl.sh ref (no tlog, interactive OIDC) and prints the
// verify/inspect commands. Used by ork push --sign-local and ork pattern sign --local.
func signLocal(ctx context.Context, ref *registry.Ref, ttl string) error {
	// Interactive=true: let cosign run the full browser OIDC flow rather than
	// grabbing an ambient CI token that may be scoped incorrectly for Fulcio.
	fmt.Println("Opening browser for OIDC sign-in...")
	spin := StartSpinner("Signing (keyless · no tlog)...")
	if err := orksigning.Sign(ctx, ref.String(), orksigning.SignOptions{
		SkipConfirmation: false,
		IgnoreTlog:       true,
	}); err != nil {
		spin.Failure()
		return fmt.Errorf("signing failed: %w", err)
	}
	spin.Stop()
	fmt.Printf("%s Signed\n", successMark())
	fmt.Printf("\n  Expires in   %s\n", ttl)
	fmt.Printf("  Verify:      ork pattern verify %s --no-tlog\n", ref.String())
	fmt.Printf("  Inspect:     ork inspect %s\n", ref.String())
	fmt.Println()
	return nil
}

// signPatternRef signs an already-pushed pattern ref using Cosign keyless.
// ignoreTlog skips the Rekor transparency log — use for ephemeral artifacts.
// Used by ork pattern sign, ork push --sign, and ork push --sign-local.
func signPatternRef(ctx context.Context, ref string, ignoreTlog bool) error {
	return orksigning.Sign(ctx, ref, orksigning.SignOptions{
		SkipConfirmation: true,
		IgnoreTlog:       ignoreTlog,
	})
}

// verifyPatternRef verifies the Cosign keyless signature on a pattern ref.
// Returns the cosign output so callers can extract subject/issuer.
// Used by ork pattern verify and ork inspect.
func verifyPatternRef(ctx context.Context, ref string, identities []string, ignoreTlog bool, out *strings.Builder) error {
	return orksigning.Verify(ctx, ref, orksigning.VerifyOptions{
		ExpectedIdentities: identities,
		IgnoreTlog:         ignoreTlog,
		Output:             out,
	})
}

func init() {
	patternSignCmd.Flags().Bool("local", false, "Push to ttl.sh and sign for local testing")
	patternSignCmd.Flags().String("ttl", "1h", "TTL for the ttl.sh artifact when using --local (e.g. 1h, 24h)")
	patternSignCmd.Flags().String("dir", "", "Pattern directory to push when using --local (default: current dir)")
	patternSignCmd.Flags().Bool("no-tlog", false, "Skip Rekor transparency log upload (useful for ephemeral or local artifacts)")

	patternVerifyCmd.Flags().Bool("verbose", false, "Show full certificate and Rekor log detail")
	patternVerifyCmd.Flags().Bool("no-tlog", false, "Skip Rekor transparency log check (useful for local testing)")

	patternCmd.AddCommand(patternSignCmd)
	patternCmd.AddCommand(patternVerifyCmd)
	rootCmd.AddCommand(patternCmd)

	shadowGlobalCommandFlags(patternCmd, "file")
}
