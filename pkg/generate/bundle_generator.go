package generate

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/konfig"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Bundle(kfg *konfig.Konfig, rules []rbacv1.PolicyRule, namespace, outputFile string) error {
	var bundle string

	// 1. Generate RBAC
	rbacOut, err := renderRBAC(kfg, rules, namespace)
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
