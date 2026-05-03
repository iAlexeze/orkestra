package generate

import (
	"fmt"
	"os"

	"github.com/orkspace/orkestra/pkg/konfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	defaultConfigMapName = "orkestra-katalog"
	defaultConfigMapKey  = "katalog.yaml"
)

func ConfigMap(inputFile, namespace, outputFile string) ([]byte, error) {
	if namespace == "" {
		namespace = konfig.GetStrEnv("ORKESTRA_NAMESPACE", "orkestra-system")
	}
	out, err := renderNamespaceAndConfigMap(inputFile, namespace)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// renderNamespaceAndConfigMap is the full standalone output: Namespace + ConfigMap.
func renderNamespaceAndConfigMap(inputFile, namespace string) ([]byte, error) {
	nsBytes, err := renderNamespace(namespace)
	if err != nil {
		return nil, err
	}
	cmBytes, err := renderConfigMapBytes(inputFile, namespace)
	if err != nil {
		return nil, err
	}
	return []byte("---\n" + string(nsBytes) + "\n---\n" + string(cmBytes)), nil
}

// renderConfigMapBytes marshals a ConfigMap from the given file.
// The Namespace is intentionally excluded — callers prepend it so that
// bundle assembly can include it exactly once.
func renderConfigMapBytes(inputFile, namespace string) ([]byte, error) {
	raw, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", inputFile, err)
	}
	cm := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultConfigMapName,
			Namespace: namespace,
			Labels:    konfig.OrkestraBaseLabels(),
		},
		Data: map[string]string{
			defaultConfigMapKey: string(raw),
		},
	}
	out, err := yaml.Marshal(cm)
	if err != nil {
		return nil, fmt.Errorf("marshal configmap: %w", err)
	}
	return out, nil
}
