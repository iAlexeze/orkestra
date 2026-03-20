package konfig

import (
	// "log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func Init(filenames ...string) (*Konfig, error) {
	err := godotenv.Load(filenames...)
	if err != nil {
		// log.Print("No '.env' file found. Defaulting to system defined variables...")
	}

	kfg := &Konfig{
		ork: orkKonfig{
			Name:        orkestra,
			ShortName:   ork,
			Version:     GetStrEnv("ORK_VERSION", "1.0.0"),
			Environment: GetStrEnv("ORK_ENV", "development"),
			LogLevel:    GetStrEnv("LOG_LEVEL", "info"),
		},
		cluster: clusterKonfig{
			KubekonfigPath:   GetStrEnv("KUBEKONFIG", ""),
			MasterURL:        GetStrEnv("MASTER_URL", ""),
			InCluster:        GetBoolEnv("IN_CLUSTER", false),
			Name:             GetStrEnv("CLUSTER_NAME", "kubernetes-crd-example"),
			DefaultNamespace: GetStrEnv("NAMESPACE", "default"),

			// Workload
			DefaultResync:  GetDurEnvSeconds("DEFAULT_RESYNC", 15),
			Finalizer:      GetStrEnv("FINALIZER", "konduktor.orkestra.io/finalizer"),
			LabelSelector:  GetStrEnv("LABEL_SELECTOR", "ork=estra"),
			DefaultWorkers: GetIntEnv("DEFAULT_WORKERS", 3),
		},
		healthServer: healthServer{
			Port:         GetStrEnv("HEALTH_PORT", "5000"),
			ReadTimeout:  GetDurEnvSeconds("SRV_READ_TIMEOUT", 5),
			WriteTimeout: GetDurEnvSeconds("SRV_WRITE_TIMEOUT", 20),
		},
		konductor: konductorElection{
			ElectionNamespace: GetStrEnv("LEADER_ELECTION_NAMESPACE", "default"),
			LeaseDuration:     GetDurEnvSeconds("LEASE_DURATION", 60),
			RenewDeadline:     GetDurEnvSeconds("RENEW_DEADLINE", 40),
			RetryPeriod:       GetDurEnvSeconds("RETRY_PERIOD", 10),
		},
		katalog: katalogKonfig{
			DefaultMaxQueueDepth:    GetIntEnv("MAX_QUEUE_DEPTH", 2000),
			DefaultDegradeThreshold: GetIntEnv("DEGRADE_THRESHOLD", 5),
			Paths:                   GetStrSliceEnv("KATALOG_PATH", []string{}),
		},
	}

	// normalize environment
	kfg.normalizeEnvironment()

	// validate struct
	if err = Validate().Struct(kfg); err != nil {
		return nil, err
	}

	return kfg, nil
}

// -----------------------------------------------------------------------------

// GetStrEnv returns the string value of an env
func GetStrEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

// GetStrSliceEnv returns the slice value of an env
func GetStrSliceEnv(key string, def []string) []string {
	if val, ok := os.LookupEnv(key); ok {
		return []string{val}
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
