package doktor

import (
	"fmt"
	"sort"
	"strings"
)

const (
	NotificationSecretName = "orkestra-notification"
	NotificationSecretFile = "orkestra-notification-secret.yaml"
)

// NotificationEnvVars maps developer .env keys to Orkestra konfig env var names.
// Only keys with non-empty values are included.
func NotificationEnvVars(vars []EnvVar) map[string]string {
	mapping := map[string]string{
		"SMTP_HOST":         "SMTP_HOST",
		"SMTP_PORT":         "SMTP_PORT",
		"SMTP_USER":         "SMTP_USER",
		"SMTP_PASS":         "SMTP_PASS",
		"SMTP_PASSWORD":     "SMTP_PASS",
		"SMTP_FROM":         "SMTP_FROM",
		"SLACK_WEBHOOK_URL": "SLACK_WEBHOOK_URL",
		"SLACK_WEBHOOK":     "SLACK_WEBHOOK_URL",
	}

	result := make(map[string]string)
	for _, v := range vars {
		key := strings.ToUpper(v.Key)
		if orkKey, ok := mapping[key]; ok && v.Value != "" {
			result[orkKey] = v.Value
		}
	}
	return result
}

// BuildNotificationSecret generates the orkestra-notification Kubernetes Secret
// YAML. When applied to orkestra-system and referenced by runtime.extraEnvFrom,
// it injects SMTP/Slack credentials into the Orkestra runtime so pkg/konfig
// can pick them up as normal env vars.
func BuildNotificationSecret(envMap map[string]string) string {
	if len(envMap) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("apiVersion: v1\n")
	b.WriteString("kind: Secret\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + NotificationSecretName + "\n")
	b.WriteString("  namespace: " + OrkestraNamespace + "\n")
	b.WriteString("  labels:\n")
	b.WriteString("    app.kubernetes.io/managed-by: orkestra\n")
	b.WriteString("type: Opaque\n")
	b.WriteString("stringData:\n")

	keys := make([]string, 0, len(envMap))
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %s: %q\n", k, envMap[k]))
	}
	return b.String()
}
