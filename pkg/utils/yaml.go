package utils

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func StrictUnmarshal(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(out); err != nil {
		return errors.New(FormatYAMLError(err, data))
	}

	return nil
}

func FormatYAMLError(err error, data []byte) string {
	if err == nil {
		return ""
	}

	var errorLines []string
	lines := strings.Split(err.Error(), "\n")

	// Extract context lines for each error
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse yaml unmarshal error format
		if strings.Contains(line, "line ") && strings.Contains(line, "field") {
			// Format: "line X: field Y not found in type Z"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}

			lineInfo := strings.TrimSpace(parts[0]) // "line X"
			errorMsg := strings.TrimSpace(parts[1]) // "field Y not found in type Z"

			// Extract line number
			var lineNum int
			fmt.Sscanf(lineInfo, "line %d", &lineNum)

			// Extract field name
			fieldWords := strings.Fields(errorMsg)
			if len(fieldWords) >= 2 && fieldWords[0] == "field" {
				fieldName := fieldWords[1]

				// Get the line content for context
				contextLine := getLineFromData(data, lineNum)

				// Format like Docker Compose
				errorLines = append(errorLines,
					fmt.Sprintf("• %s: additional properties '%s' not allowed",
						strings.TrimSpace(contextLine),
						fieldName))
			}
		} else if strings.Contains(line, "line ") && strings.Contains(line, "cannot unmarshal") {
			// Handle type mismatches
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}

			lineInfo := strings.TrimSpace(parts[0])
			errorMsg := strings.TrimSpace(parts[1])

			var lineNum int
			fmt.Sscanf(lineInfo, "line %d", &lineNum)

			contextLine := getLineFromData(data, lineNum)

			errorLines = append(errorLines,
				fmt.Sprintf("• %s: %s",
					strings.TrimSpace(contextLine),
					errorMsg))
		}
	}

	if len(errorLines) == 0 {
		return err.Error()
	}

	return "Validation failed:\n" + strings.Join(errorLines, "\n")
}

func getLineFromData(data []byte, lineNum int) string {
	if lineNum <= 0 {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	if lineNum-1 < len(lines) {
		return strings.TrimSpace(lines[lineNum-1])
	}
	return ""
}
