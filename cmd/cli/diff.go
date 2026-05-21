//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Show a unified diff between two files",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file1 := args[0]
		file2 := args[1]
		verbose, _ := cmd.Flags().GetBool("verbose")

		a, err := os.ReadFile(file1)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file1, err)
		}

		b, err := os.ReadFile(file2)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file2, err)
		}

		diff := unifiedDiff(file1, file2, a, b, verbose)
		fmt.Println(diff)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)

	// Shadow global flags so they don't appear under `ork init`
	diffCmd.Flags().Bool("debug", false, "")
	diffCmd.Flags().String("kubeconfig", "", "")
	diffCmd.Flags().StringSlice("katalog", nil, "")
	diffCmd.Flags().Bool("verbose", false, "")

	// Hide them from help output
	diffCmd.Flags().MarkHidden("debug")
	diffCmd.Flags().MarkHidden("kubeconfig")
	diffCmd.Flags().MarkHidden("katalog")
	diffCmd.Flags().MarkHidden("verbose")
}

// Helper
func unifiedDiff(nameA, nameB string, a, b []byte, verbose bool) string {
	alines := strings.Split(string(a), "\n")
	blines := strings.Split(string(b), "\n")

	var out []string
	out = append(out, utils.Gray("--- "+nameA))
	out = append(out, utils.Gray("+++ "+nameB))

	max := len(alines)
	if len(blines) > max {
		max = len(blines)
	}

	for i := 0; i < max; i++ {
		var A, B string
		if i < len(alines) {
			A = alines[i]
		}
		if i < len(blines) {
			B = blines[i]
		}

		switch {
		case A == B:
			if verbose {
				out = append(out, " "+A)
			}
		case A == "":
			out = append(out, utils.Green("+"+B))
		case B == "":
			out = append(out, utils.Red("-"+A))
		default:
			out = append(out, utils.Red("-"+A))
			out = append(out, utils.Green("+"+B))
		}
	}

	return strings.Join(out, "\n")
}
