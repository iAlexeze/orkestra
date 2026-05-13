// pkg/doctor/compose.go
//
// Reads docker-compose.yaml and extracts service definitions.
// Classifies services as stateless (Deployments) or stateful (Motif candidates).
package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"gopkg.in/yaml.v3"
)

// ComposeFileVariants stores common docker compose variants
var ComposeFileVariants = []string{
	"docker-compose.yaml",
	"docker-compose.yml",
	"compose.yaml",
	"compose.yml",
}

// ComposeFile is the parsed docker-compose.yaml.
type ComposeFile struct {
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService is one service entry in a docker compose file.
type ComposeService struct {
	Image       interface{} `yaml:"image,omitempty"` // string
	Build       interface{} `yaml:"build,omitempty"` // string or object
	Ports       interface{} `yaml:"ports,omitempty"` // []string or []int
	Environment interface{} `yaml:"environment,omitempty"`
	EnvFile     interface{} `yaml:"env_file,omitempty"` // string or []string or [{path,required}]
	Volumes     interface{} `yaml:"volumes,omitempty"`
	DependsOn   interface{} `yaml:"depends_on,omitempty"`
}

// imageString extracts the image field as a string.
func (s ComposeService) imageString() string {
	if s.Image == nil {
		return ""
	}
	if str, ok := s.Image.(string); ok {
		return str
	}
	return ""
}

// BuildContext extracts the build context directory and optional Dockerfile path
// from the service's build: field. Returns ("", "") when build: is not set.
//
// String form: build: ./app  → context="./app", dockerfile=""
// Map form:    build: {context: ./app, dockerfile: ./Dockerfile.prod}
func (s ComposeService) BuildContext() (context, dockerfile string) {
	if s.Build == nil {
		return "", ""
	}
	switch v := s.Build.(type) {
	case string:
		return v, ""
	case map[string]interface{}:
		if ctx, ok := v["context"].(string); ok {
			context = ctx
		}
		if df, ok := v["dockerfile"].(string); ok {
			dockerfile = df
		}
		return context, dockerfile
	}
	return "", ""
}

// KnownMotif maps a detected infrastructure image to its Motif reference.
type KnownMotif struct {
	Image       string   // detected image prefix (may include registry like confluentinc/)
	MotifRef    string   // short name in orkestra-motifs
	AppYAMLKeys []string // keys to add to app.yaml
	AdminUI     string   // companion admin UI name
}

// knownMotifs is the catalog of detectable infrastructure services.
var knownMotifs = []KnownMotif{
	{
		Image:       "confluentinc/cp-kafka",
		MotifRef:    "kafka",
		AppYAMLKeys: []string{"kafkaImage", "kafkaVolumeSize"},
		AdminUI:     "Kafka UI",
	},
	{
		Image:       "bitnami/kafka",
		MotifRef:    "kafka",
		AppYAMLKeys: []string{"kafkaImage", "kafkaVolumeSize"},
		AdminUI:     "Kafka UI",
	},
	{
		Image:       "apache/kafka",
		MotifRef:    "kafka",
		AppYAMLKeys: []string{"kafkaImage", "kafkaVolumeSize"},
		AdminUI:     "Kafka UI",
	},
	{
		Image:       "kafka",
		MotifRef:    "kafka",
		AppYAMLKeys: []string{"kafkaImage", "kafkaVolumeSize"},
		AdminUI:     "Kafka UI",
	},
	{
		Image:       "postgres",
		MotifRef:    "postgres",
		AppYAMLKeys: []string{"postgresImage", "postgresVolumeSize", "postgresUser"},
		AdminUI:     "pgAdmin",
	},
	{
		Image:       "postgresql",
		MotifRef:    "postgres",
		AppYAMLKeys: []string{"postgresImage", "postgresVolumeSize", "postgresUser"},
		AdminUI:     "pgAdmin",
	},
	{
		Image:       "mysql",
		MotifRef:    "mysql",
		AppYAMLKeys: []string{"mysqlImage", "mysqlVolumeSize", "mysqlUser"},
		AdminUI:     "phpMyAdmin",
	},
	{
		Image:       "mariadb",
		MotifRef:    "mysql",
		AppYAMLKeys: []string{"mysqlImage", "mysqlVolumeSize", "mysqlUser"},
		AdminUI:     "phpMyAdmin",
	},
	{
		Image:       "mongo",
		MotifRef:    "mongodb",
		AppYAMLKeys: []string{"mongoImage", "mongoVolumeSize"},
		AdminUI:     "mongo-express",
	},
	{
		Image:       "rabbitmq",
		MotifRef:    "rabbitmq",
		AppYAMLKeys: []string{"rabbitmqImage"},
		AdminUI:     "RabbitMQ Management",
	},
	{
		Image:       "redis",
		MotifRef:    "redis",
		AppYAMLKeys: []string{"redisImage", "redisVolumeSize"},
		AdminUI:     "RedisInsight",
	},
}

// StatefulService describes a detected infrastructure service.
type StatefulService struct {
	Name  string
	Motif KnownMotif
	Image string
}

// ParseCompose reads and parses a docker-compose.yaml file.
func ParseCompose(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &cf, nil
}

// DependsOnNames returns the service names this service depends on.
// Handles both the list form (depends_on: [a, b]) and the map/condition form
// (depends_on: {a: {condition: service_healthy}}).
func (s ComposeService) DependsOnNames() []string {
	if s.DependsOn == nil {
		return nil
	}
	switch v := s.DependsOn.(type) {
	case []interface{}:
		var names []string
		for _, item := range v {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
		return names
	case map[string]interface{}:
		var names []string
		for k := range v {
			names = append(names, k)
		}
		return names
	}
	return nil
}

// StatefulDepsPerApp maps each buildable app name to the stateful services whose
// Motif declaration belongs in that app's katalog.
//
// Each stateful service is declared exactly once — in the katalog of the first
// app (in appNames order) that lists it in depends_on. If no app declares a
// depends_on relationship with a given stateful service, it is assigned to
// appNames[0] as a fallback so it still gets deployed exactly once.
//
// Example: three apps all depend on postgres and two depend on redis.
// Result: postgres and redis each appear in one katalog only (the earliest app
// in appNames that depends on each), not repeated across every depending app.
func StatefulDepsPerApp(cf *ComposeFile, appNames []string, stateful []StatefulService) map[string][]StatefulService {
	result := make(map[string][]StatefulService, len(appNames))

	statefulByName := make(map[string]StatefulService, len(stateful))
	for _, ss := range stateful {
		statefulByName[ss.Name] = ss
	}

	// claimed tracks which stateful services have already been assigned to an app.
	// Once claimed, later apps that also depend on the service are skipped —
	// the Motif is a cluster-level resource and only needs one declaration.
	claimed := make(map[string]bool)

	for _, appName := range appNames {
		svc, ok := cf.Services[appName]
		if !ok {
			continue
		}
		for _, dep := range svc.DependsOnNames() {
			ss, isStateful := statefulByName[dep]
			if isStateful && !claimed[dep] {
				result[appName] = append(result[appName], ss)
				claimed[dep] = true
			}
		}
	}

	// Any stateful service with no depends_on reference goes to the first app.
	if len(appNames) > 0 {
		for _, ss := range stateful {
			if !claimed[ss.Name] {
				result[appNames[0]] = append(result[appNames[0]], ss)
			}
		}
	}

	return result
}

// ClassifyServices separates compose services into stateless (Deployments)
// and stateful (Motif-backed) based on known infrastructure images.
func ClassifyServices(cf *ComposeFile) (stateless []string, stateful []StatefulService) {
	for name, svc := range cf.Services {
		img := svc.imageString()
		if km, ok := detectMotif(img); ok {
			stateful = append(stateful, StatefulService{
				Name:  name,
				Motif: km,
				Image: img,
			})
		} else {
			stateless = append(stateless, name)
		}
	}
	return
}

// DetectComposeFile looks for docker-compose.yaml in dir and returns the path.
// Returns empty string when no compose file is found.
func DetectComposeFile(dir string) string {
	for _, name := range ComposeFileVariants {
		p := strings.Join([]string{dir, name}, "/")
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// ContainerPort returns the container (right-hand) port from the first ports: entry.
// Handles all compose port formats:
//   - "8080"                  → "8080"
//   - "3001:3000"             → "3000"
//   - "127.0.0.1:8000:8080"  → "8080"
//   - integer 8080            → "8080"
//   - long form {target: 8080} → "8080"
func (s ComposeService) ContainerPort() string {
	if s.Ports == nil {
		return ""
	}
	entries, ok := s.Ports.([]interface{})
	if !ok || len(entries) == 0 {
		return ""
	}
	return extractContainerPort(entries[0])
}

func extractContainerPort(entry interface{}) string {
	switch v := entry.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		// "127.0.0.1:8080:8080" or "3000:3000" or "8080" — take rightmost segment
		parts := strings.Split(v, ":")
		return parts[len(parts)-1]
	case map[string]interface{}:
		if target, ok := v["target"]; ok {
			return fmt.Sprintf("%v", target)
		}
	}
	return ""
}

// EnvFileList returns the env_file paths declared on this service.
// Handles string, []string, and long form [{path: "...", required: false}].
func (s ComposeService) EnvFileList() []string {
	if s.EnvFile == nil {
		return nil
	}
	switch v := s.EnvFile.(type) {
	case string:
		return []string{v}
	case []interface{}:
		var result []string
		for _, item := range v {
			switch it := item.(type) {
			case string:
				result = append(result, it)
			case map[string]interface{}:
				if path, ok := it["path"].(string); ok {
					result = append(result, path)
				}
			}
		}
		return result
	}
	return nil
}

// parseServiceEnvironment parses the environment: field into a flat key→value map.
// Handles map form (KEY: value) and list form (- KEY=value or - KEY).
func parseServiceEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}
	switch v := env.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if val == nil {
				result[k] = ""
			} else {
				result[k] = fmt.Sprintf("%v", val)
			}
		}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				if idx := strings.IndexByte(str, '='); idx >= 0 {
					result[str[:idx]] = str[idx+1:]
				} else {
					result[str] = ""
				}
			}
		}
	}
	return result
}

// resolveServiceEnvVars merges env vars for a compose service in priority order:
//  1. root .env in baseDir (lowest) — skipped if env_file: is declared
//  2. each env_file entry (in order)
//  3. environment: block (highest)
//
// IsCfg tagging from ParseEnvFile is preserved for all file-sourced vars.
// environment: block vars inherit IsCfg from any already-loaded entry for the same key.
func resolveServiceEnvVars(svc ComposeService, baseDir string) []orktypes.EnvVar {
	merged := make(map[string]orktypes.EnvVar)

	envFiles := svc.EnvFileList()

	// Root .env is used as base only when no env_file: is declared.
	if len(envFiles) == 0 {
		rootPath := filepath.Join(baseDir, ".env")
		if fileExists(rootPath) {
			if vars, err := ParseEnvFile(rootPath); err == nil {
				for _, v := range vars {
					merged[v.Key] = v
				}
			}
		}
	}

	// env_file entries override root .env, applied in order.
	for _, ef := range envFiles {
		if !filepath.IsAbs(ef) {
			ef = filepath.Join(baseDir, ef)
		}
		if fileExists(ef) {
			if vars, err := ParseEnvFile(ef); err == nil {
				for _, v := range vars {
					merged[v.Key] = v
				}
			}
		}
	}

	// environment: block wins over all file sources.
	// Preserve IsCfg from the already-loaded entry when overriding the value.
	for k, val := range parseServiceEnvironment(svc.Environment) {
		if existing, ok := merged[k]; ok {
			existing.Value = val
			merged[k] = existing
		} else {
			merged[k] = orktypes.EnvVar{Key: k, Value: val, IsCfg: false}
		}
	}

	result := make([]orktypes.EnvVar, 0, len(merged))
	for _, v := range merged {
		result = append(result, v)
	}
	return result
}

// detectMotif returns the KnownMotif for an image if it matches, and true.
func detectMotif(image string) (KnownMotif, bool) {
	if image == "" {
		return KnownMotif{}, false
	}
	// Strip tag
	base := strings.Split(image, ":")[0]

	// Match multi-segment names first (e.g., confluentinc/cp-kafka, bitnami/kafka)
	for _, km := range knownMotifs {
		if strings.Contains(km.Image, "/") && strings.HasPrefix(base, km.Image) {
			return km, true
		}
	}

	// Match on last path segment
	name := base
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		name = base[idx+1:]
	}
	for _, km := range knownMotifs {
		if !strings.Contains(km.Image, "/") && strings.HasPrefix(name, km.Image) {
			return km, true
		}
	}
	return KnownMotif{}, false
}

// ParseBuildContext resolves the build: field for a service to absolute paths.
// Returns the build context directory and the Dockerfile path.
//
//	build: ./frontend          → context=<projectDir>/frontend, dockerfile=<context>/Dockerfile
//	build: {context: ./api, dockerfile: Dockerfile.prod}
//	                           → context=<projectDir>/api, dockerfile=<context>/Dockerfile.prod
//	build: nil                 → "", "" (service has no build: — uses a pre-built image)
func ParseBuildContext(svc ComposeService, projectDir string) (buildCtx, dockerfile string) {
	rawCtx, rawDF := svc.BuildContext()
	if rawCtx == "" && rawDF == "" {
		return "", ""
	}

	if rawCtx == "" {
		rawCtx = "."
	}

	if filepath.IsAbs(rawCtx) {
		buildCtx = rawCtx
	} else {
		buildCtx = filepath.Join(projectDir, rawCtx)
	}

	if rawDF == "" {
		dockerfile = filepath.Join(buildCtx, "Dockerfile")
	} else if filepath.IsAbs(rawDF) {
		dockerfile = rawDF
	} else {
		dockerfile = filepath.Join(buildCtx, rawDF)
	}

	return buildCtx, dockerfile
}

// ServiceEnvVars is the public alias for resolveServiceEnvVars.
// Returns merged env vars for a single compose service in priority order:
//  1. root .env in projectDir (lowest priority)
//  2. each env_file entry (in declaration order)
//  3. environment: block (highest priority)
func ServiceEnvVars(svc ComposeService, projectDir string) []orktypes.EnvVar {
	return resolveServiceEnvVars(svc, projectDir)
}
