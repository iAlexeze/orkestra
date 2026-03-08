package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func Init(filenames ...string) (*Config, error) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Printf("failed to load env from file: %v", err)
		log.Print("Defaulting to system defined variables...")
	}

	cfg := &Config{
		app: appConfig{
			Name:        GetStrEnv("APP_NAME", "multi-crd-controller"),
			Version:     GetStrEnv("APP_VERSION", "1.0.0"),
			Environment: GetStrEnv("APP_ENV", "development"),
		},
		cluster: clusterConfig{
			KubeconfigPath: GetStrEnv("KUBECONFIG", ""),
			MasterURL:      GetStrEnv("MASTER_URL", ""),
			InCluster:      GetBoolEnv("IN_CLUSTER", false),
			Name:           GetStrEnv("CLUSTER_NAME", "kubernetes-crd-example"),
			Namespace:      GetStrEnv("NAMESPACE", "default"),

			// Workload
			DefaultResync: GetDurEnvSeconds("DEFAULT_RESYNC", 15),
			Finalizer:     GetStrEnv("FINALIZER", "alexia.ai/finalizer"),
			LabelSelector: GetStrEnv("LABEL_SELECTOR", "app=alexia"),
			Workers:       GetIntEnv("WORKERS", 3),
		},
		healthServer: healthServer{
			Port:         GetStrEnv("PORT", "5000"),
			ReadTimeout:  GetDurEnvSeconds("SRV_READ_TIMEOUT", 5),
			WriteTimeout: GetDurEnvSeconds("SRV_WRITE_TIMEOUT", 20),
		},
		leader: leaderElection{
			LeaseDuration: GetDurEnvSeconds("LEASE_DURATION", 60),
			RenewDeadline: GetDurEnvSeconds("RENEW_DEADLINE", 40),
			RetryPeriod:   GetDurEnvSeconds("RETRY_PERIOD", 10),
		},
	}

	// Normalize app environment
	switch strings.ToLower(cfg.app.Environment) {
	case "dev", "development":
		cfg.app.Environment = "development"
	case "uat", "staging":
		cfg.app.Environment = "staging"
	case "live", "prod", "production":
		cfg.app.Environment = "production"
	default:
		cfg.app.Environment = "development"

	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetStrEnv returns the string value of an env
func GetStrEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

// GetBoolEnv returns the boolean value of an env
func GetBoolEnv(key string, def bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		valBool, _ := strconv.ParseBool(val)
		return valBool
	}
	return def
}

// GetDurEnvSeconds returns the time.duration value of an env
func GetDurEnvSeconds(key string, def int) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		valInt, _ := strconv.Atoi(val)
		return time.Duration(valInt) * time.Second
	}
	return time.Duration(def) * time.Second
}

// GetIntEnv returns the int value of an env
func GetIntEnv(key string, def int) int {
	if val, ok := os.LookupEnv(key); ok {
		valInt, _ := strconv.Atoi(val)
		return valInt
	}
	return def
}

// validate validates required values:
// To be replaced by more advaced validation framework when needed
func (c *Config) validate() error {
	required := map[string]string{
		c.App().Name:               "APP_NAME",
		c.Cluster().KubeconfigPath: "KUBECONFIG",
	}

	for k, v := range required {
		if v == "" {
			return fmt.Errorf("%s is not set", k)
		}
	}
	return nil
}
