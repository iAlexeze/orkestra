package generate

import (
	"fmt"

	"github.com/orkspace/orkestra/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	ork   = "orkestra"
	orkcc = "orkestra-cc"
)

func RBAC(rules []rbacv1.PolicyRule, namespace, outputFile string) ([]byte, error) {
	out, err := renderNamespaceAndRBAC(rules, namespace)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// renderNamespaceAndRBAC is the full standalone output: Namespace + ServiceAccounts + ClusterRole + ClusterRoleBinding.
func renderNamespaceAndRBAC(rules []rbacv1.PolicyRule, namespace string) ([]byte, error) {
	nsBytes, err := renderNamespace(namespace)
	if err != nil {
		return nil, err
	}
	rbacBytes, err := renderRBAC(rules, namespace)
	if err != nil {
		return nil, err
	}
	return prependNamespaceDoc(nsBytes, rbacBytes), nil
}

// renderNamespace marshals a Namespace object. Used by RBAC, ConfigMap, and Bundle
// to ensure the namespace exists before any namespaced resources are applied.
func renderNamespace(namespace string) ([]byte, error) {
	ns := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: labels.OrkestraBaseLabels(),
		},
	}
	return yaml.Marshal(ns)
}

// renderRBAC marshals ServiceAccounts, ClusterRole, and ClusterRoleBinding only.
// The Namespace is intentionally excluded — callers prepend it via renderNamespace
// so that bundle assembly can include it exactly once.
func renderRBAC(rules []rbacv1.PolicyRule, namespace string) ([]byte, error) {
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
				Labels:    labels.OrkestraResourceLabels(),
			},
		}
		serviceAccounts = append(serviceAccounts, sa)
	}

	cr := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ork,
			Labels: labels.OrkestraResourceLabels(),
		},
		Rules: rules,
	}

	crb := rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   ork,
			Labels: labels.OrkestraResourceLabels(),
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

	objs := []interface{}{}
	for _, sa := range serviceAccounts {
		objs = append(objs, sa)
	}
	objs = append(objs, cr, crb)

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

// prependNamespaceDoc places the Namespace as the first document in a multi-doc
// YAML stream, separated from the remaining docs by a blank line.
func prependNamespaceDoc(nsBytes, rest []byte) []byte {
	return []byte("---\n" + string(nsBytes) + "\n" + string(rest))
}
