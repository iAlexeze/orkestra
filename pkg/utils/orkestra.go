package utils

import (
	"strings"
)

const (
	ColorReset     = "\033[0m"
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
