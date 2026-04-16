package generate

import (
	"fmt"
	"io"
	"os"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/merger"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	ork   = "orkestra"
	orkcc = "orkestra-cc"
)

func RBAC(m *merger.Merger, namespace, outputFile string) error {
	out, err := renderRBAC(m, namespace)
	if err != nil {
		return err
	}

	if outputFile != "" {
		return os.WriteFile(outputFile, out, 0644)
	}

	fmt.Println(string(out))
	return nil
}

func renderRBAC(m *merger.Merger, namespace string) ([]byte, error) {
	var kat katalog.Katalog
	kat.Spec = m.ToSpec()

	rules := kat.GenerateRBACRules()

	// Build ServiceAccounts dynamically
	var serviceAccounts []corev1.ServiceAccount
	for _, name := range []string{ork, orkcc} {
		sa := corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		serviceAccounts = append(serviceAccounts, sa)
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

	// Collect all objects
	objs := []interface{}{}
	for _, sa := range serviceAccounts {
		objs = append(objs, sa)
	}
	objs = append(objs, cr, crb)

	// Marshal with separators
	out := ""
	for i, obj := range objs {
		b, err := yaml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("marshal rbac: %w", err)
		}
		out += "---\n" + string(b)
		if i < len(objs)-1 {
			out += "\n"
		}
	}

	return []byte(out), nil
}

func rBACToWriter(w io.Writer, m *merger.Merger, namespace string) error {
	out, err := renderRBAC(m, namespace)
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	return err
}

func RenderRBACToString(m *merger.Merger, namespace string) (string, error) {
	out, err := renderRBAC(m, namespace)
	return string(out), err
}
