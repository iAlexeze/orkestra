// pkg/inspect/printer.go
package inspect

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Colours — used when output is a terminal.
// Suppressed when stdout is piped or redirected.
const (
	colReset  = "\033[0m"
	colBold   = "\033[1m"
	colGreen  = "\033[32m"
	colYellow = "\033[33m"
	colRed    = "\033[31m"
	colCyan   = "\033[36m"
	colGray   = "\033[90m"
)

// TableWriter wraps tabwriter for consistent aligned output.
type TableWriter struct {
	w *tabwriter.Writer
}

// NewTableWriter returns a TableWriter writing to stdout.
func NewTableWriter() *TableWriter {
	return &TableWriter{
		w: tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0),
	}
}

// Header prints a bold header row.
func (t *TableWriter) Header(cols ...string) {
	fmt.Fprintln(t.w, Bold(strings.Join(cols, "\t")))
}

// Row prints one data row.
func (t *TableWriter) Row(cols ...string) {
	fmt.Fprintln(t.w, strings.Join(cols, "\t"))
}

// Flush flushes the tabwriter buffer.
func (t *TableWriter) Flush() {
	t.w.Flush()
}

// PrintTable prints a header + rows to stdout, flushed and aligned.
func PrintTable(w io.Writer, header []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, Bold(strings.Join(header, "\t")))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// PrintField prints a label: value line used in describe output.
func PrintField(label, value string) {
	fmt.Printf("%-20s %s\n", Bold(label+":"), value)
}

// PrintSection prints a bold section header with a separator.
func PrintSection(title string) {
	fmt.Printf("\n%s\n%s\n", Bold(title), strings.Repeat("─", 60))
}

// PrintSuccess prints a green checkmark line.
func PrintSuccess(msg string) {
	fmt.Printf("%s✓%s  %s\n", colGreen, colReset, msg)
}

// PrintWarning prints a yellow warning line.
func PrintWarning(msg string) {
	fmt.Printf("%s⚠%s  %s\n", colYellow, colReset, msg)
}

// PrintError prints a red error line.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s  %s\n", colRed, colReset, msg)
}

// PrintInfo prints a dim informational line.
func PrintInfo(msg string) {
	fmt.Printf("%s%s%s\n", colGray, msg, colReset)
}

// Bold wraps text in bold ANSI codes.
func Bold(s string) string {
	return colBold + s + colReset
}

// Cyan wraps text in cyan ANSI codes.
func Cyan(s string) string {
	return colCyan + s + colReset
}

// HumanAge returns a human-readable age string from a Kubernetes timestamp.
// Format matches kubectl: 5s, 2m, 3h, 4d, 2w
func HumanAge(t metav1.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	return formatDuration(time.Since(t.Time))
}

// FormatDuration formats a duration in human-readable short form.
// Exported so CLI commands and tests can use it directly.
func FormatDuration(d time.Duration) string {
	return formatDuration(d)
}

// formatDuration is the internal implementation.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}

// HealthIcon returns a coloured status icon based on a status string.
func HealthIcon(status string) string {
	switch strings.ToLower(status) {
	case "ready", "running", "active", "true", "healthy":
		return colGreen + "●" + colReset
	case "pending", "progressing":
		return colYellow + "●" + colReset
	case "error", "failed", "false", "unhealthy", "degraded":
		return colRed + "●" + colReset
	default:
		return colGray + "●" + colReset
	}
}

// ExtractStatus extracts a status phase or condition from an unstructured object.
// Returns "Unknown" if no status is found.
func ExtractStatus(obj *unstructured.Unstructured) string {
	// Try status.phase first
	if phase, ok, _ := unstructured.NestedString(obj.Object, "status", "phase"); ok && phase != "" {
		return phase
	}
	// Try status.state
	if state, ok, _ := unstructured.NestedString(obj.Object, "status", "state"); ok && state != "" {
		return state
	}
	// Try status.conditions[0].type + status (Ready=True etc.)
	conditions, ok, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if ok && len(conditions) > 0 {
		if cond, ok := conditions[0].(map[string]interface{}); ok {
			cType, _ := cond["type"].(string)
			cStatus, _ := cond["status"].(string)
			if cType != "" && cStatus != "" {
				return cType + "=" + cStatus
			}
		}
	}
	return "Unknown"
}

// PrintNestedMap prints a nested map as indented key: value lines.
// Used by describe to print spec and status sections.
func PrintNestedMap(m map[string]interface{}, indent int) {
	prefix := strings.Repeat("  ", indent)
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", prefix, Bold(k))
			PrintNestedMap(val, indent+1)
		case []interface{}:
			fmt.Printf("%s%s:\n", prefix, Bold(k))
			for _, item := range val {
				switch iv := item.(type) {
				case map[string]interface{}:
					PrintNestedMap(iv, indent+1)
					fmt.Println()
				default:
					fmt.Printf("%s  - %v\n", prefix, iv)
				}
			}
		default:
			fmt.Printf("%s%-20s %v\n", prefix, k+":", val)
		}
	}
}
