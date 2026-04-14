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
		// ── Unified security configuration ───────────────────────────────────
		// ENV vars populate SecurityConfig as defaults.
		// Katalog YAML values are merged on top in KomposeKatalogFromYaml.
		//
		// ENV → SecurityConfig mapping:
		//   ENABLE_DELETION_PROTECTION  → security.DeletionProtection.Enabled
		//   DELETION_PROTECTION_POLICY  → security.DeletionProtection.FailurePolicy
		//   ENABLE_ADMISSION_WEBHOOK    → security.Webhooks.Admission.Enabled
		//   ENABLE_CONVERSION           → security.Conversion.Enabled
		//   WEBHOOK_FAILURE_POLICY      → security.Webhooks.FailurePolicy
		//   ORKESTRA_SERVICE_NAME       → security.Webhooks.ServiceName
		//                               → security.DeletionProtection.ServiceName
		//   RBAC_AUTO_APPLY             → security.RBAC.Enabled
		//   RBAC_CLEANUP_ON_SHUTDOWN    → security.RBAC.CleanupOnShutdown
		//   CONVERSION_WINDOW           → security.Conversion.ConversionWindow
		//   TLS_CERT / TLS_KEY          → security.Webhooks.TLSCert / TLSKey (initial
		//                                 values; overwritten by ensureSecurity() when
		//                                 Orkestra generates its own certificates)
		security: func() SecurityConfig {
			svcName := GetStrEnv("ORKESTRA_SERVICE_NAME", "orkestra")
			var s SecurityConfig
			s.DeletionProtection.Enabled = GetBoolEnv("ENABLE_DELETION_PROTECTION", false)
			s.DeletionProtection.FailurePolicy = GetStrEnv("DELETION_PROTECTION_POLICY", "Fail")
			s.DeletionProtection.ServiceName = svcName
			s.Webhooks.Admission.Enabled = GetBoolEnv("ENABLE_ADMISSION_WEBHOOK", false)
			s.Conversion.Enabled = GetBoolEnv("ENABLE_CONVERSION", false)
			s.Webhooks.FailurePolicy = GetStrEnv("WEBHOOK_FAILURE_POLICY", "Ignore")
			s.Webhooks.ServiceName = svcName
			s.Conversion.ConversionWindow = GetIntEnv("CONVERSION_WINDOW", 100)
			s.Webhooks.TLSCert = GetStrEnv("TLS_CERT", "")
			s.Webhooks.TLSKey = GetStrEnv("TLS_KEY", "")
			s.RBAC.Enabled = GetBoolEnv("RBAC_AUTO_APPLY", false)
			s.RBAC.CleanupOnShutdown = GetBoolEnv("RBAC_CLEANUP_ON_SHUTDOWN", false)
			return s
		}(),

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
