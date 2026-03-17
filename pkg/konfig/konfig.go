package konfig

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func Init(filenames ...string) (*Konfig, error) {
	err := godotenv.Load(filenames...)
	if err != nil {
		log.Printf("failed to load env from file: %v", err)
		log.Print("Defaulting to system defined variables...")
	}

	kfg := &Konfig{
		app: appKonfig{
			Name:        GetStrEnv("APP_NAME", "orkestra"),
			Version:     GetStrEnv("APP_VERSION", "1.0.0"),
			Environment: GetStrEnv("APP_ENV", "development"),
			LogLevel:    GetStrEnv("LOG_LEVEL", "info"),
		},
		cluster: clusterKonfig{
			KubekonfigPath:   GetStrEnv("KUBEKONFIG", ""),
			MasterURL:        GetStrEnv("MASTER_URL", ""),
			InCluster:        GetBoolEnv("IN_CLUSTER", false),
			Name:             GetStrEnv("CLUSTER_NAME", "kubernetes-crd-example"),
			DefaultNamespace: GetStrEnv("DEFAULT_NAMESPACE", "default"),

			// Workload
			DefaultResync:  GetDurEnvSeconds("DEFAULT_RESYNC", 15),
			Finalizer:      GetStrEnv("FINALIZER", "alexia.ai/finalizer"),
			LabelSelector:  GetStrEnv("LABEL_SELECTOR", "app=alexia"),
			DefaultWorkers: GetIntEnv("DEFAULT_WORKERS", 3),
		},
		healthServer: healthServer{
			Port:         GetStrEnv("PORT", "5000"),
			ReadTimeout:  GetDurEnvSeconds("SRV_READ_TIMEOUT", 5),
			WriteTimeout: GetDurEnvSeconds("SRV_WRITE_TIMEOUT", 20),
		},
		konductor: konductorElection{
			LeaseDuration: GetDurEnvSeconds("LEASE_DURATION", 60),
			RenewDeadline: GetDurEnvSeconds("RENEW_DEADLINE", 40),
			RetryPeriod:   GetDurEnvSeconds("RETRY_PERIOD", 10),
		},
		katalog: katalogKonfig{
			DefaultMaxQueueDepth:    GetIntEnv("MAX_QUEUE_DEPTH", 2000),
			DefaultDegradeThreshold: GetIntEnv("DEGRADE_THRESHOLD", 5),
			Mode:                    GetStrEnv("KATALOG_MODE", "go"),
			Path:                    GetStrEnv("KATALOG_PATH", ""),
		},
	}

	// normalize environment
	kfg.normalizeEnvironment()

	// validate katalog konfig
	if err = kfg.validateKatalogKonfig(); err != nil {
		return nil, err
	}

	// validate struct
	if err = Validate().Struct(kfg); err != nil {
		return nil, err
	}

	return kfg, nil
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
