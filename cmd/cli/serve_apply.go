//go:build !runtime && !gateway

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/orkspace/orkestra/pkg/gateway/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var serveApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply an intent or CR to a live gateway",
	Long: `Apply an intent or CR to a live gateway via POST /api/v1/apply.

The file may be a flat intent (target mode) or a full CR (apiVersion + kind).
Both YAML and JSON are accepted. Defaults to intent.yaml or intent.json in the
current directory if --file is not set.

The gateway handles target resolution, token validation, provenance stamping,
and admission. This command only sends the body and prints the response.

Example (intent mode):
  ork serve apply --api https://gateway.myorg.io --token "$ORK_TOKEN"

Example (explicit file):
  ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN"

Example (dry run — no CR applied):
  ork serve apply -f cr.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --dry-run

Example (override routing surface conflict):
  ork serve apply -f intent.yaml --api https://gateway.myorg.io --token "$ORK_TOKEN" --override`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		apiURL, _ := cmd.Flags().GetString("api")
		token, _ := cmd.Flags().GetString("token")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		override, _ := cmd.Flags().GetBool("override")

		if file == "" {
			file = resolveDefaultIntentFile()
			if file == "" {
				return fmt.Errorf("%s no intent file found — create intent.yaml or intent.json, or pass --file <file>", failureMark())
			}
		}

		raw, err := readIntentFile(file)
		if err != nil {
			return fmt.Errorf("%s reading file %q: %w", failureMark(), file, err)
		}

		body, err := json.Marshal(raw)
		if err != nil {
			return fmt.Errorf("%s marshalling request body: %w", failureMark(), err)
		}

		endpoint := strings.TrimRight(apiURL, "/") + "/api/v1/apply"
		switch {
		case dryRun && override:
			endpoint += "?dryRun=true&override=true"
		case dryRun:
			endpoint += "?dryRun=true"
		case override:
			endpoint += "?override=true"
		}

		req, err := http.NewRequestWithContext(cmd.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("%s building request: %w", failureMark(), err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("%s gateway unreachable at %s: %w", failureMark(), apiURL, err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%s reading response: %w", failureMark(), err)
		}

		var result api.ApplyResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("%s unexpected response (HTTP %d): %s", failureMark(), resp.StatusCode, string(respBody))
		}

		verbose, _ := rootCmd.PersistentFlags().GetBool("verbose")
		if verbose {
			var pretty bytes.Buffer
			if err := json.Indent(&pretty, respBody, "  ", "  "); err == nil {
				fmt.Printf("\n%s\n", pretty.String())
			} else {
				fmt.Printf("\n%s\n", string(respBody))
			}
		}

		return printApplyResult(file, result, dryRun)
	},
}

func printApplyResult(file string, r api.ApplyResponse, dryRun bool) error {
	mode := "apply"
	if dryRun {
		mode = "dry-run"
	}
	fmt.Printf("\n  %s %s  %s\n\n", gray("serve"), cyan(mode), gray(file))

	if !r.Accepted {
		fmt.Printf("  %s %s\n", red("✗"), red(r.Message))
		for _, v := range r.Violations {
			fmt.Printf("    %s %s  %s\n", red("↳"), gray(v.Field), red(v.Message))
		}
		return fmt.Errorf("%s", red("apply rejected"))
	}

	ref := r.Kind
	if r.Namespace != "" {
		ref += "  " + gray(r.Namespace+"/"+r.Name)
	} else if r.Name != "" {
		ref += "  " + gray(r.Name)
	}
	fmt.Printf("  %s %s\n", green("✓"), ref)

	for _, w := range r.Warnings {
		fmt.Printf("  %s %s\n", yellow("warn"), yellow(w))
	}

	if r.PollURL != "" {
		fmt.Printf("  %s %s\n", gray("poll:"), cyan(r.PollURL))
	}

	if r.Payload != nil {
		b, err := yaml.Marshal(r.Payload)
		if err == nil {
			lines := strings.TrimRight(string(b), "\n")
			indented := "  " + strings.ReplaceAll(lines, "\n", "\n  ")
			fmt.Printf("\n%s\n", indented)
		}
	}

	marker := green("accepted")
	if dryRun {
		marker = cyan("dry-run accepted")
	}
	fmt.Printf("\n%s\n\n", marker)
	return nil
}

func init() {
	serveApplyCmd.Flags().StringP("file", "f", "", "Intent or CR file to apply (YAML or JSON; default: intent.yaml or intent.json in cwd)")
	serveApplyCmd.Flags().StringP("api", "a", "http://localhost:8080", "Gateway base URL")
	serveApplyCmd.Flags().StringP("token", "t", "", "Bearer token for the gateway")
	serveApplyCmd.Flags().Bool("dry-run", false, "Preview without applying — the CR is not written to the cluster")
	serveApplyCmd.Flags().Bool("override", false, "Override routing surface conflict — allows switching a resource to a different target")

	_ = serveApplyCmd.MarkFlagRequired("token")

	serveCmd.AddCommand(serveApplyCmd)
}
