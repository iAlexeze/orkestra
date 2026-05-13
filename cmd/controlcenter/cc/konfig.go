package controlcenter

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type ControlCenterKonfig struct {
	Port                 string        `json:"port"`
	IgnoreDefault        bool          `json:"ignoreDefault"`
	RefreshInterval      time.Duration `json:"refreshInterval"`
	LogLevel             string        `json:"logLevel"`
	URLs                 []string      `json:"urls"`
	EnableRuntimeManager bool          `json:"enableRuntimeManager"`
}

func NewControlCenterKonfig() *ControlCenterKonfig {
	return handleEnvVars()
}

// Handle user defined variables
func handleEnvVars() *ControlCenterKonfig {
	port := getStrEnv("PORT", "8081")
	ignoreDefault := getBoolEnv("IGNORE_DEFAULT", false)
	runtimeManager := getBoolEnv("ENABLE_RUNTIME_MANAGER", true)
	loglevel := getStrEnv("LOG_LEVEL", "info")
	refreshInterval := getDurEnv("REFRESH_INTERVAL", 15)
	urls := splitEnv("ORKESTRA_URLS", []string{})

	return &ControlCenterKonfig{
		Port:                 port,
		IgnoreDefault:        ignoreDefault,
		RefreshInterval:      refreshInterval,
		LogLevel:             loglevel,
		URLs:                 urls,
		EnableRuntimeManager: runtimeManager,
	}
}

func getStrEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		result = append(result, p)
	}

	return result
}

func splitEnvUpper(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p == "" {
			continue
		}
		result = append(result, p)
	}

	return result
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		valueBool, _ := strconv.ParseBool(value)
		return valueBool
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int64 {
	if value := os.Getenv(key); value != "" {
		valueInt, _ := strconv.ParseInt(value, 10, 64)
		return valueInt
	}
	return int64(defaultValue)
}

// Duration in seconds
func getDurEnv(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		valueInt, _ := strconv.Atoi(value)
		return time.Duration(valueInt) * time.Second
	}

	return time.Duration(defaultValue) * time.Second
}
