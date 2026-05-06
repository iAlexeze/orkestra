package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const deletionProtectionLabel = "orkestra.io/deletion-protection"

// GenerateOptions controls which sections are included in the generated Katalog.
type GenerateOptions struct {
	NoHA       bool   // skip HPA and PDB; single replica
	NoSecure   bool   // skip deletionProtection and protection labels
	Clean      bool   // add cleanupOnShutdown: true to deletionProtection
	Name       string // override app name
	AddIngress bool   // force-include Ingress even when no frontend was detected
	NotifyMe   bool   // add notification block
	UseCompose string // path to docker-compose.yaml (expand stateful services via Motifs)
	OutDir     string // override .orkestra/ directory; empty = info.Dir/.orkestra

	// InjectStateful, when non-nil, specifies exactly which stateful services
	// to inject into this app's katalog and app.yaml. When set it takes
	// precedence over the UseCompose-based auto-detection, allowing the caller
	// to supply a pre-filtered, per-app slice derived from depends_on analysis.
	// Use nil (not an empty slice) to fall back to UseCompose detection.
	InjectStateful []StatefulService
}

// Init generates .orkestra/katalog.yaml and .orkestra/app.yaml, creates the
// .orkestra/ directory, and updates .gitignore to exclude the bundle directory.
func Init(info *ProjectInfo, opts GenerateOptions) error {
	name := opts.Name
	if name == "" {
		name = info.AppName
	}

	orkDir := opts.OutDir
	if orkDir == "" {
		orkDir = filepath.Join(info.Dir, ".orkestra")
	}
	if err := os.MkdirAll(orkDir, 0o755); err != nil {
		return fmt.Errorf("creating %s/: %w", orkDir, err)
	}

	// Resolve the stateful services to inject.
	// InjectStateful (pre-filtered per-app slice) takes precedence over
	// UseCompose-based auto-detection. The latter is used for single-app
	// compose workflows; multi-app uses InjectStateful so each app only
	// receives the stateful services it actually depends on.
	var stateful []StatefulService
	if opts.InjectStateful != nil {
		stateful = opts.InjectStateful
	} else if opts.UseCompose != "" {
		cf, err := ParseCompose(opts.UseCompose)
		if err != nil {
			return fmt.Errorf("reading compose file: %w", err)
		}
		_, stateful = ClassifyServices(cf)
	}

	katalogContent := buildKatalog(name, info, opts)
	if len(stateful) > 0 {
		katalogContent = injectStatefulServices(katalogContent, name, stateful)
	}
	if err := os.WriteFile(filepath.Join(orkDir, "katalog.yaml"), []byte(katalogContent), 0o644); err != nil {
		return fmt.Errorf("writing katalog.yaml: %w", err)
	}

	crContent := buildCR(name, info, opts)
	if len(stateful) > 0 {
		crContent = injectStatefulAppYAML(crContent, stateful, info)
	}
	if err := os.WriteFile(filepath.Join(orkDir, ApplicationFile), []byte(crContent), 0o644); err != nil {
		return fmt.Errorf("writing app.yaml: %w", err)
	}

	if err := updateGitignore(info.Dir); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	return nil
}

// injectStatefulServices appends Motif import blocks for each stateful service
// to the generated Katalog YAML string.
func injectStatefulServices(katalogYAML, appName string, services []StatefulService) string {
	if len(services) == 0 {
		return katalogYAML
	}

	var b strings.Builder
	b.WriteString(katalogYAML)
	b.WriteString("\n        # Stateful services expanded from docker-compose.yaml\n")
	b.WriteString("        imports:\n")
	for _, s := range services {
		ref := s.Motif.MotifRef
		b.WriteString(fmt.Sprintf("          - motif: %s\n", ref))
		b.WriteString("            with:\n")
		switch ref {
		case "postgres":
			b.WriteString("              image: \"{{ .data.postgresImage }}\"\n")
			b.WriteString(fmt.Sprintf("              passwordSecretName: %s-secrets\n", appName))
			b.WriteString("              user: \"{{ .data.postgresUser }}\"\n")
			b.WriteString("              volumeSize: \"{{ .data.postgresVolumeSize }}\"\n")
			b.WriteString("              adminEmail: \"{{ .data.adminEmail }}\"\n")
			b.WriteString("              adminPassword: \"{{ .data.adminPassword }}\"\n")
		case "redis":
			b.WriteString("              image: \"{{ .data.redisImage }}\"\n")
			b.WriteString("              volumeSize: \"{{ .data.redisVolumeSize }}\"\n")
		case "mysql":
			b.WriteString("              image: \"{{ .data.mysqlImage }}\"\n")
			b.WriteString(fmt.Sprintf("              passwordSecretName: %s-secrets\n", appName))
			b.WriteString("              user: \"{{ .data.mysqlUser }}\"\n")
			b.WriteString("              volumeSize: \"{{ .data.mysqlVolumeSize }}\"\n")
		case "kafka":
			b.WriteString("              image: \"{{ .data.kafkaImage }}\"\n")
		case "rabbitmq":
			b.WriteString("              image: \"{{ .data.rabbitmqImage }}\"\n")
		case "mongodb":
			b.WriteString("              image: \"{{ .data.mongoImage }}\"\n")
			b.WriteString("              volumeSize: \"{{ .data.mongoVolumeSize }}\"\n")
		default:
			fmt.Fprintf(&b, "              image: \"{{ .data.%sImage }}\"\n", ref)
		}
	}
	return b.String()
}

// injectStatefulAppYAML appends stateful service configuration keys to app.yaml.
// Each key includes a simple, developer-friendly comment explaining its purpose.
func injectStatefulAppYAML(appYAML string, services []StatefulService, info *ProjectInfo) string {
	if len(services) == 0 {
		return appYAML
	}

	author, _ := LastCommitAuthor()
	if author.Notfound {
		author.Email = "dev@orkestra.sh"
		author.Name = "admin"
	}

	appUser := truncate(info.AppName+"_"+"user", 15)
	appUser = cleanUp(appUser)

	var b strings.Builder
	b.WriteString(appYAML)

	for _, s := range services {
		fmt.Fprintf(&b, "\n  # ── %s (from %s) ────────────────────────────────────────\n",
			strings.Title(s.Motif.MotifRef), filepath.Base(info.ComposePath))

		switch s.Motif.MotifRef {

		case "postgres":
			fmt.Fprintf(&b, "  postgresImage: \"%s\"        # Container image for your database\n", s.Image)
			b.WriteString("  postgresVolumeSize: \"10Gi\"          # Volume size for your database\n")
			fmt.Fprintf(&b, "  postgresUser: \"%s\"               # Default database user\n", appUser)
			fmt.Fprintf(&b, "  adminEmail: \"%s\"                # Admin email for database access\n", author.Email)
			fmt.Fprintf(&b, "  adminPassword: \"%s\"             # Admin password (auto-generated)\n", author.Name)

		case "redis":
			fmt.Fprintf(&b, "  redisImage: \"%s\"           # Container image for Redis\n", s.Image)
			b.WriteString("  redisVolumeSize: \"2Gi\"            # Volume size for Redis data\n")

		case "mysql":
			fmt.Fprintf(&b, "  mysqlImage: \"%s\"           # Container image for MySQL\n", s.Image)
			b.WriteString("  mysqlVolumeSize: \"10Gi\"           # Volume size for your database\n")
			fmt.Fprintf(&b, "  mysqlUser: \"%s\"                # Default database user\n", appUser)

		case "kafka":
			fmt.Fprintf(&b, "  kafkaImage: \"%s\"           # Container image for Kafka\n", s.Image)

		case "rabbitmq":
			fmt.Fprintf(&b, "  rabbitmqImage: \"%s\"        # Container image for RabbitMQ\n", s.Image)

		case "mongodb":
			fmt.Fprintf(&b, "  mongoImage: \"%s\"           # Container image for MongoDB\n", s.Image)
			b.WriteString("  mongoVolumeSize: \"10Gi\"           # Volume size for MongoDB data\n")
		}
	}

	return b.String()
}

// protectionLabels returns the YAML block for deletion-protection labels,
// indented to the given depth. Returns empty string when secure is false.
func protectionLabels(indent string, secure bool) string {
	if !secure {
		return ""
	}
	return indent + "labels:\n" +
		indent + "  - key: " + deletionProtectionLabel + "\n" +
		indent + "    value: \"true\"\n"
}

func buildKatalog(name string, info *ProjectInfo, opts GenerateOptions) string {
	ns := name + "-orkestra-ns"
	secure := !opts.NoSecure
	notifyME := opts.NotifyMe
	var b strings.Builder

	b.WriteString("# Generated by ork doctor init — edit freely\n")
	b.WriteString("# Re-run 'ork doctor init --name <my-project>' to regenerate from scratch\n\n")

	b.WriteString("apiVersion: orkestra.orkspace.io/v1\n")
	b.WriteString("kind: Katalog\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + name + "\n")
	b.WriteString("  version: latest\n")
	b.WriteString("  description: \"Orkestra")
	if !opts.NoHA {
		b.WriteString(" HA")
	}
	b.WriteString(" deployment for " + name + "\"\n\n")

	if secure {
		b.WriteString("security:\n")
		b.WriteString("  deletionProtection:\n")
		b.WriteString("    enabled: true\n")
		if opts.Clean {
			b.WriteString("    cleanupOnShutdown: true\n")
		}
		b.WriteString("\n")
	}

	if notifyME {
		author, _ := LastCommitAuthor()

		b.WriteString("notification:\n")
		b.WriteString("  enabled: true\n")
		b.WriteString("  defaults:\n")
		b.WriteString("    interval: 15m\n")
		b.WriteString("    # slackWebhookUrl injected via SLACK_WEBHOOK_URL in orkestra-notification Secret\n")
		b.WriteString("  teams:\n")
		b.WriteString("    " + name + ":\n")
		if info.HasSMTP && author != nil && author.Email != "" {
			b.WriteString("      email:\n")
			b.WriteString("        - " + author.Email + "\n")
		}
		if info.HasSlack {
			b.WriteString("      slack:\n")
			b.WriteString("        - \"#deployments\"\n")
			b.WriteString("      # slackWebhookUrl falls back to notification.defaults.slackWebhookUrl\n")
		}
		b.WriteString("      message: \"{{ .metadata.name }}: {{ .status.phase }}\"\n")
		b.WriteString("\n")
	}

	b.WriteString("spec:\n")
	b.WriteString("  crds:\n")
	b.WriteString("    " + name + ":\n")
	b.WriteString("      apiTypes:\n")
	b.WriteString("        kind: ConfigMap\n")
	b.WriteString("      labelSelector:\n")
	b.WriteString("        ork.io/app: " + name + "-orkestra\n")
	b.WriteString("      allowedNamespaces: [\"" + ns + "\"]\n\n")

	b.WriteString("      operatorBox:\n")
	b.WriteString("        onCreate:\n")

	// ServiceAccount — created once, referenced by the Deployment
	b.WriteString("          serviceAccounts:\n")
	b.WriteString("            - name: \"{{ .metadata.name }}-sa\"\n")
	b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString(protectionLabels("              ", secure))
	b.WriteString("\n")

	// Role — grants the ServiceAccount access only to this app's Deployment
	b.WriteString("          roles:\n")
	b.WriteString("            - name: \"{{ .metadata.name }}-role\"\n")
	b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString("              rules:\n")
	b.WriteString("                - apiGroups: [\"apps\"]\n")
	b.WriteString("                  resources: [\"deployments\"]\n")
	b.WriteString("                  verbs: [\"get\", \"list\", \"watch\", \"update\", \"patch\"]\n")
	b.WriteString("                  resourceNames: [\"{{ .metadata.name }}\"]\n")
	b.WriteString("\n")

	// RoleBinding — binds the Role to the ServiceAccount
	b.WriteString("          roleBindings:\n")
	b.WriteString("            - name: \"{{ .metadata.name }}-rolebinding\"\n")
	b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString("              roleRef:\n")
	b.WriteString("                name: \"{{ .metadata.name }}-role\"\n")
	b.WriteString("              subjects:\n")
	b.WriteString("                - kind: ServiceAccount\n")
	b.WriteString("                  name: \"{{ .metadata.name }}-sa\"\n")
	b.WriteString("                  namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString("\n")

	// Deployment — only when data.image is set
	replicas := "2"
	if opts.NoHA {
		replicas = "1"
	}
	b.WriteString("          deployments:\n")
	b.WriteString("            - name: \"{{ .metadata.name }}\"\n")
	b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString("              image: \"{{ .data.image }}\"\n")
	b.WriteString("              replicas: \"{{ .data.replicas | default \\\"" + replicas + "\\\" }}\"\n")
	b.WriteString("              serviceAccountName: \"{{ .metadata.name }}-sa\"\n")
	if len(info.Secrets) > 0 || len(info.Config) > 0 {
		b.WriteString("              envFrom:\n")
		if len(info.Secrets) > 0 {
			b.WriteString("                - secretRef: " + name + "-secrets\n")
		}
		if len(info.Config) > 0 {
			b.WriteString("                - configMapRef: " + name + "-config\n")
		}
	}
	b.WriteString("              resourceProfile: \"{{ .data.resourceProfile | default \\\"burst\\\" }}\"\n")
	b.WriteString(protectionLabels("              ", secure))
	b.WriteString("              reconcile: true\n")
	b.WriteString("              when:\n")
	b.WriteString("                - field: data.image\n")
	b.WriteString("                  exists: true\n\n")

	// Service
	b.WriteString("          services:\n")
	b.WriteString("            - name: \"{{ .metadata.name }}-svc\"\n")
	b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
	b.WriteString("              port: \"{{ .data.port | default \\\"" + info.Port + "\\\" }}\"\n")
	b.WriteString("              targetPort: \"{{ .data.port | default \\\"" + info.Port + "\\\" }}\"\n")
	b.WriteString(protectionLabels("              ", secure))
	b.WriteString("              reconcile: true\n\n")

	// Ingress — when frontend was detected or explicitly requested via --add-ingress
	if info.HasFrontend || opts.AddIngress {
		b.WriteString("          ingresses:\n")
		b.WriteString("            - name: \"{{ .metadata.name }}-ingress\"\n")
		b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
		b.WriteString("              host: \"{{ .data.host }}\"\n")
		b.WriteString("              serviceName: \"{{ .metadata.name }}-svc\"\n")
		b.WriteString(protectionLabels("              ", secure))
		b.WriteString("              reconcile: true\n")
		b.WriteString("              when:\n")
		b.WriteString("                - field: data.host\n")
		b.WriteString("                  exists: true\n\n")
	}

	// HPA and PDB
	if !opts.NoHA {
		b.WriteString("          hpa:\n")
		b.WriteString("            - name: \"{{ .metadata.name }}-hpa\"\n")
		b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
		b.WriteString("              scaleTargetRef:\n")
		b.WriteString("                apiVersion: apps/v1\n")
		b.WriteString("                kind: Deployment\n")
		b.WriteString("                name: \"{{ .metadata.name }}\"\n")
		b.WriteString("              minReplicas:  \"{{ .data.replicas | default \\\"" + replicas + "\\\" }}\"\n")
		b.WriteString("              maxReplicas: \"{{ .data.maxReplicas | default \\\"10\\\" }}\"\n")
		b.WriteString("              targetCPUUtilizationPercentage: \"70\"\n")
		b.WriteString(protectionLabels("              ", secure))
		b.WriteString("              reconcile: true\n")
		b.WriteString("              when:\n")
		b.WriteString("                - field: data.image\n")
		b.WriteString("                  exists: true\n\n")

		b.WriteString("          pdb:\n")
		b.WriteString("            - name: \"{{ .metadata.name }}-pdb\"\n")
		b.WriteString("              namespace: \"{{ .metadata.name }}-ns\"\n")
		b.WriteString("              minAvailable: 1\n")
		b.WriteString(protectionLabels("              ", secure))
		b.WriteString("              reconcile: true\n")
		b.WriteString("              when:\n")
		b.WriteString("                - field: data.image\n")
		b.WriteString("                  exists: true\n\n")
	}

	// Status fields
	b.WriteString("        status:\n")
	b.WriteString("          fields:\n")

	b.WriteString("            - path: phase\n")
	b.WriteString("              value: \"Pending\"\n")
	b.WriteString("              when:\n")
	b.WriteString("                - field: data.image\n")
	b.WriteString("                  notExists: true\n\n")

	b.WriteString("            - path: phase\n")
	b.WriteString("              value: \"Deploying\"\n")
	b.WriteString("              when:\n")
	b.WriteString("                - field: data.image\n")
	b.WriteString("                  exists: true\n")
	b.WriteString("                - field: \"{{ replicasReady .children.deployment }}\"\n")
	b.WriteString("                  equals: \"false\"\n")
	b.WriteString("                  notify:\n")
	b.WriteString("                    teams: [developer]\n")
	b.WriteString("                    message: \"{{ .metadata.name }} is deploying but replicas are not ready yet.")
	b.WriteString(" Check logs: kubectl logs -n {{ .metadata.namespace }} -l ork.io/app={{ .metadata.name }} --tail=50")
	b.WriteString(" | Roll back if stuck: ork deploy rollback\"\n")
	b.WriteString("\n")

	b.WriteString("            - path: phase\n")
	b.WriteString("              value: \"Ready\"\n")
	b.WriteString("              when:\n")
	b.WriteString("                - field: \"{{ replicasReady .children.deployment }}\"\n")
	b.WriteString("                  equals: \"true\"\n\n")

	if info.HasFrontend || opts.AddIngress {
		b.WriteString("            - path: url\n")
		b.WriteString("              value: \"https://{{ .data.host }}\"\n")
		b.WriteString("              when:\n")
		b.WriteString("                - field: data.host\n")
		b.WriteString("                  exists: true\n\n")
	}

	b.WriteString("            - path: image\n")
	b.WriteString("              value: \"{{ .data.image }}\"\n")

	return b.String()
}

// buildCR constructs the developer-facing ConfigMap that defines an app’s
// deployment settings. This file becomes .orkestra/app.yaml and is the only
// configuration a developer edits; ork deploy reads it and manages all
// Kubernetes resources from it.
func buildCR(name string, info *ProjectInfo, opts GenerateOptions) string {
	crName := name + "-orkestra"
	ns := name + "-orkestra-ns"

	replicas := "2"
	if opts.NoHA {
		replicas = "1"
	}

	var b strings.Builder

	b.WriteString("# This is the only Kubernetes object you manage.\n")
	b.WriteString("# Run 'ork deploy' to apply — do not apply this file manually.\n\n")

	b.WriteString("apiVersion: v1\n")
	b.WriteString("kind: ConfigMap\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + crName + "\n")
	b.WriteString("  namespace: " + ns + "\n")
	b.WriteString("  labels:\n")
	b.WriteString("    ork.io/app: " + crName + "\n")
	if !opts.NoSecure {
		b.WriteString("    " + deletionProtectionLabel + ": \"true\"\n")
	}

	b.WriteString("data:\n")
	b.WriteString("  # ork deploy updates this automatically — do not edit\n")
	b.WriteString("  image: \"\"\n")
	b.WriteString("  # ───────────────────────────────────────────────────────────────\n\n")

	b.WriteString("  # Application port\n")
	fmt.Fprintf(&b, "  port: \"%s\"\n\n", info.Port)

	b.WriteString("  # How many copies of your app to run normally\n")
	fmt.Fprintf(&b, "  replicas: \"%s\"\n\n", replicas)

	b.WriteString("  # How much CPU and memory your app should get. Choose a profile:\n")
	b.WriteString("  # docs.orkestra.sh/concepts/resource-profiles\n")
	b.WriteString("  resourceProfile: \"burst\"\n\n")

	if !opts.NoHA {
		b.WriteString("  # Maximum copies of your app to run when traffic increases\n")
		b.WriteString("  maxReplicas: \"10\"\n\n")
	}

	if info.HasFrontend || opts.AddIngress {
		b.WriteString("  # This app's public hostname (e.g. myapp.example.com)\n")
		b.WriteString("  host: \"\"\n\n")
	}

	b.WriteString("  # Orkestra Control Center hostname (e.g. control.mycompany.com)\n")
	b.WriteString("  controlCenterHost: \"\"\n\n")

	return b.String()
}

// ReadCRName reads the ConfigMap name from .orkestra/app.yaml.
// Returns the name (e.g. "my-app-orkestra") so callers don't need to re-derive it.
func ReadCRName(appYAML string) (string, error) {
	data, err := os.ReadFile(appYAML)
	if err != nil {
		return "", err
	}
	// metadata.name is always at exactly 2-space indent in the generated file.
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "  name: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "  name: ")), nil
		}
	}
	return "", fmt.Errorf("name field not found in %s", appYAML)
}

// buildOrkestraValues returns a commented-out values.yaml template for the
// Orkestra Helm chart. The most actionable setting (Control Center ingress)
// is at the top so the user can immediately expose the CC URL after install.
func buildOrkestraValues(name string, notifyMe bool) string {
	extraEnvFrom := ""
	if notifyMe {
		extraEnvFrom = `
# ── Notifications ─────────────────────────────────────────────────────────────
# orkestra-notification Secret is created by ork deploy when SMTP/Slack env
# vars are present. It injects credentials into the Orkestra runtime so
# pkg/konfig reads them as normal env vars.
runtime:
  extraEnvFrom:
    - secretRef:
        name: orkestra-notification
`
	}
	_ = name
	return `# .orkestra/values.yaml — Orkestra cluster configuration
# Apply changes with: ork deploy --upgrade-orkestra
#
# ── Control Center ────────────────────────────────────────────────────────────
# Expose the Control Center externally so 'ork deploy' can show its URL.
# After enabling, set controlCenterHost in .orkestra/app.yaml to the same host.
#
# controlCenter:
#   ingress:
#     enabled: true
#     className: nginx          # nginx | traefik | kong | etc.
#     hosts:
#       - host: control.mycompany.com
#         paths:
#           - path: /
#             pathType: Prefix
#     tls:
#       - secretName: control-center-tls
#         hosts:
#           - control.mycompany.com

# ── Runtime ───────────────────────────────────────────────────────────────────
# runtime:
#   replicaCount: 2
#   resources:
#     requests:
#       cpu: 100m
#       memory: 128Mi
#     limits:
#       cpu: 500m
#       memory: 512Mi
#   image:
#     tag: ""                   # pin to a specific Orkestra version
` + extraEnvFrom
}

// func updateGitignore(dir string) error {
// 	giPath := filepath.Join(dir, ".gitignore")
// 	entry := "\n# Added by ork doctor init\n.orkestra/bundle/\n"

// 	data, err := os.ReadFile(giPath)
// 	if err != nil && !os.IsNotExist(err) {
// 		return err
// 	}
// 	if strings.Contains(string(data), ".orkestra/bundle/") {
// 		return nil
// 	}
// 	f, err := os.OpenFile(giPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	_, err = f.WriteString(entry)
// 	return err
// }

func updateGitignore(dir string) error {
	giPath := filepath.Join(dir, ".gitignore")

	required := []string{
		".orkestra/bundle/",
		".orkestra/init.ork",
	}

	// Read existing .gitignore (if any)
	data, err := os.ReadFile(giPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existing := string(data)

	// Build the block to append
	var toAppend []string
	for _, entry := range required {
		if !strings.Contains(existing, entry) {
			toAppend = append(toAppend, entry)
		}
	}

	// Nothing to add
	if len(toAppend) == 0 {
		return nil
	}

	// Open for append or create
	f, err := os.OpenFile(giPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header + missing entries
	_, err = f.WriteString("\n# Added by ork doctor init\n" + strings.Join(toAppend, "\n") + "\n")
	return err
}
