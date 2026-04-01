package generate

import (
	"fmt"
	"os"

	"github.com/ialexeze/orkestra/pkg/katalog"
	"github.com/ialexeze/orkestra/pkg/merger"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	ork = "orkestra"
)

func RBAC(m *merger.Merger, namespace, outputFile string) error {
	var kat katalog.Katalog
	kat.Spec = m.ToSpec()

	rules := kat.GenerateRBACRules()

	// ServiceAccount
	sa := corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ork,
			Namespace: namespace,
		},
	}

	// ClusterRole
	cr := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ork,
		},
		Rules: rules,
	}

	// ClusterRoleBinding
	crb := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ork,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     ork,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ork,
				Namespace: namespace,
			},
		},
	}

	// Marshal all three with YAML separators
	out := ""

	objs := []interface{}{sa, cr, crb}

	for i, obj := range objs {
		b, err := yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("marshal rbac: %w", err)
		}

		// Start each document with ---
		out += "---\n" + string(b)

		// Only add a separator if it's NOT the last document
		if i < len(objs)-1 {
			out += "\n"
		}
	}

	// Write or print
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(out), 0644)
	}

	fmt.Println(out)
	return nil
}
