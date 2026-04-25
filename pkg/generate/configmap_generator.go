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

func ConfigMap(inputFile, namespace, outputFile string) error {
	// Read the file
	raw, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputFile, err)
	}

	// Use ORKESTRA_NAMESPACE env var when the caller does not pass --namespace.
	if namespace == "" {
		namespace = konfig.GetStrEnv("ORKESTRA_NAMESPACE", "orkestra-system")
	}

	// Build ConfigMap
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

	// Marshal to YAML
	out, err := yaml.Marshal(cm)
	if err != nil {
		return fmt.Errorf("marshal configmap: %w", err)
	}

	// Write output
	if outputFile != "" {
		return os.WriteFile(outputFile, out, 0644)
	}

	fmt.Println(string(out))
	return nil
}

func RenderConfigMapToString(inputFile, namespace string) (string, error) {
	raw, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", inputFile, err)
	}

	if namespace == "" {
		namespace = "orkestra-system"
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
		return "", fmt.Errorf("marshal configmap: %w", err)
	}

	return string(out), nil
}
