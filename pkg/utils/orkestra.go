package utils

import (
	"strings"
)

const (
	ColorReset     = "\033[0m"
	ColorBlack     = "\033[30m"
	ColorWhite     = "\033[37m"
	ColorRed       = "\033[31m"
	ColorGreen     = "\033[32m"
	ColorYellow    = "\033[33m"
	ColorBlue      = "\033[34m"
	ColorMagenta   = "\033[35m"
	ColorCyan      = "\033[36m"
	ColorBold      = "\033[1m"
	ColorDim       = "\033[2m"
	ColorItalic    = "\033[3m"
	ColorUnderline = "\033[4m"
	ColorBlink     = "\033[5m"
	ColorReverse   = "\033[7m"
	ColorHidden    = "\033[8m"
	ColorGray      = "\033[90m"
)

func Colorize(color, text string) string {
	return color + text + ColorReset
}

func Black(text string) string     { return Colorize(ColorBlack, text) }
func White(text string) string     { return Colorize(ColorWhite, text) }
func Red(text string) string       { return Colorize(ColorRed, text) }
func Green(text string) string     { return Colorize(ColorGreen, text) }
func Yellow(text string) string    { return Colorize(ColorYellow, text) }
func Blue(text string) string      { return Colorize(ColorBlue, text) }
func Magenta(text string) string   { return Colorize(ColorMagenta, text) }
func Cyan(text string) string      { return Colorize(ColorCyan, text) }
func Gray(text string) string      { return Colorize(ColorGray, text) }
func Bold(text string) string      { return Colorize(ColorBold, text) }
func Dim(text string) string       { return Colorize(ColorDim, text) }
func Underline(text string) string { return Colorize(ColorUnderline, text) }

// Reset removes all styling from the given text.
func Reset(text string) string { return Colorize(ColorReset, text) }

// SuccessMark returns a green checkmark symbol for successful operations.
func SuccessMark() string      { return Green("✓") }
func SuccessMarkPlain() string { return "✓" }

// FailureMark returns a red cross symbol for failed operations.
func FailureMark() string      { return Red("✗") }
func FailureMarkPlain() string { return "✗" }

// WarningMark returns a yellow warning symbol for non-fatal issues.
func WarningMark() string      { return Yellow("⚠") }
func WarningMarkPlain() string { return "⚠" }

// SecureMark returns a shield emoji indicating full security protection.
// Used in CLI output to show that deletion protection is enabled and active.
func SecureMark() string { return "🛡️" }

// SomeSecureMark returns an unlocked padlock emoji indicating partial or
// no security protection. Used in CLI output to show that deletion protection
// is disabled or only partially active (e.g., protectCRD=false but protectCRs=true).
func SomeSecureMark() string { return "🔓" }

// NoSecurityMark returns a cross mark indicating no security protection.
// Used when deletion protection is completely disabled for a resource
// (e.g., security.deletionProtection.enabled = false or per‑CRD protectCRs=false).
func NoSecurityMark() string { return "⛔" }

// InfoMark returns a cyan arrow symbol for informational messages.
func InfoMark() string      { return Cyan("→") }
func InfoMarkPlain() string { return "→" }

var OrkestraLogo = `
   ____             _           _
  / __ \___  ____ _(_)___ _____(_)___  ____
 / / / / _ \/ __ / / __ / ___/ / __ \/ __ \
/ /_/ /  __/ /_/ / / /_/ / /  / / /_/ / / / /
\____/\___/\__, /_/\__,_/_/  /_/\____/_/ /_/
           /____/     O R K E S T R A  R U N T I M E
`

var OrkestraLogoCLI = `
  ___       _              _        
 / _ \ _  _| |___ _ _  ___| |_ _ _  
| (_) | || | / -_) ' \/ -_)  _| ' \ 
 \___/ \_,_|_\___|_||_\___|\__|_||_|
         O R K E S T R A
`

// Center text to 60 columns
func Center(text string) string {
	width := 60
	lines := strings.Split(text, "\n")
	var out string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			out += "\n"
			continue
		}
		padding := (width - len(trim)) / 2
		if padding < 0 {
			padding = 0
		}
		out += strings.Repeat(" ", padding) + trim + "\n"
	}
	return out
}
