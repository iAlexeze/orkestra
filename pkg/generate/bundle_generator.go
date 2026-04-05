package generate

import (
	"fmt"

	"github.com/ialexeze/orkestra/pkg/merger"
)

func Bundle(m *merger.Merger, namespace, outputFile string) error {
	var bundle string

	// 1. Generate RBAC
	rbacOut, err := renderRBAC(m, namespace)
	if err != nil {
		return fmt.Errorf("generate rbac: %w", err)
	}
	bundle += string(rbacOut)

	// 2. Add separator
	bundle += "\n---\n"

	// 3. Generate ConfigMap from the first input file
	// Note: This requires knowing the original file path
	// For now, return error suggesting separate commands
	return fmt.Errorf("bundle command requires --katalog flag pointing to a file")
}
