// pkg/doktor/compose.go
//
// Reads docker-compose.yaml and extracts service definitions.
// Classifies services as stateless (Deployments) or stateful (Motif candidates).
package doktor

import (
	"fmt"
	"os"
	"strings"

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
	Version  string                    `yaml:"version,omitempty"`
	Services map[string]ComposeService `yaml:"services"`
}

// ComposeService is one service entry in a docker compose file.
type ComposeService struct {
	Image       interface{} `yaml:"image,omitempty"` // string
	Build       interface{} `yaml:"build,omitempty"` // string or object
	Ports       interface{} `yaml:"ports,omitempty"` // []string or []int
	Environment interface{} `yaml:"environment,omitempty"`
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
		AppYAMLKeys: []string{"kafkaImage"},
		AdminUI:     "Kafka UI",
	},
	{
		Image:       "bitnami/kafka",
		MotifRef:    "kafka",
		AppYAMLKeys: []string{"kafkaImage"},
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
