package konfig

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func Init(filenames ...string) (*Konfig, error) {
	// load .env files for tesing...
	_ = godotenv.Load(filenames...)

	kfg := &Konfig{
		ork: orkKonfig{
			Name:        Orkestra,
			ShortName:   Ork,
			Environment: GetStrEnv("ORK_ENV", "development"),
		},
		cluster: clusterKonfig{
			// KubekonfigPath:   GetStrEnv("KUBEKONFIG", ""),
			MasterURL: GetStrEnv("MASTER_URL", ""),
			Name:      GetStrEnv("CLUSTER_NAME", "orkestra-cluster"),
			Namespace: GetStrEnv("ORKESTRA_NAMESPACE", "default"),

			// Workload
			DefaultResync:       GetDurEnvSeconds("DEFAULT_RESYNC", 15),
			DefaultWorkers:      GetIntEnv("DEFAULT_WORKERS", 3),
			ShutdownTimeout:     GetDurEnvSeconds("SHUTDOWN_TIMEOUT", 30),
			ShutdownGracePeriod: GetDurEnvSeconds("SHUTDOWN_GRACE_PERIOD", 60),
		},
		webhook: webhookConfig{
			EnableWebhooks:   GetBoolEnv("ENABLE_ADMISSION_WEBHOOK", false),
			EnableConversion: GetBoolEnv("ENABLE_CONVERSION", false),
			TLSCert:          GetStrEnv("TLS_CERT", ""),
			TLSKey:           GetStrEnv("TLS_KEY", ""),
			ConversionWindow: GetIntEnv("CONVERSION_WINDOW", 100),

			// Registration
			WebhookRegistration: webhookRegistration{
				FailurePolicy:    GetStrEnv("FAILURE_POLICY", "Ignore"),
				ServiceName:      GetStrEnv("ORKESTRA_SERVICE_NAME", "orkestra"),
				ServiceNamespace: GetStrEnv("ORKESTRA_NAMESPACE", "default"),
				TLSCert:          GetStrEnv("TLS_CERT", ""),
			},
		},
		registry: registryConfig{
			RegistryURL: GetStrEnv("ORK_REGISTRY", ""),
		},
		healthServer: healthServer{
			Port:         GetStrEnv("ORK_PORT", "8080"),
			ReadTimeout:  GetDurEnvSeconds("SRV_READ_TIMEOUT", 5),
			WriteTimeout: GetDurEnvSeconds("SRV_WRITE_TIMEOUT", 20),
		},
		konductor: konductorElection{
			Namespace:     GetStrEnv("ORKESTRA_NAMESPACE", "default"),
			LeaseDuration: GetDurEnvSeconds("LEASE_DURATION", 60),
			RenewDeadline: GetDurEnvSeconds("RENEW_DEADLINE", 40),
			RetryPeriod:   GetDurEnvSeconds("RETRY_PERIOD", 10),
		},
		katalog: katalogKonfig{
			DefaultMaxQueueDepth:    GetIntEnv("MAX_QUEUE_DEPTH", 100),
			DefaultDegradeThreshold: GetIntEnv("DEGRADE_THRESHOLD", 5),
			Paths:                   GetStrSliceEnv("KATALOG_PATH", []string{}),
		},
	}

	// normalize environment
	kfg.normalizeEnvironment()

	// validate struct
	if err := Validate().Struct(kfg); err != nil {
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
