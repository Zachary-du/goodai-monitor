package config

import (
	"os"
	"strconv"
)

// Config holds the runtime configuration parameters for GoodAI Monitor.
type Config struct {
	Port              string
	Host              string
	StoragePath       string
	LogLevel          string
	Timezone          string
	DataRetentionDays int
}

// Load reads configuration values from environment variables or applies defaults.
func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8090"),
		Host:              getEnv("HOST", "0.0.0.0"),
		StoragePath:       getEnv("STORAGE_PATH", "./data/monitor.db"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		Timezone:          getEnv("TZ", "Pacific/Auckland"),
		DataRetentionDays: getEnvInt("DATA_RETENTION_DAYS", 14),
	}
}

// Address returns the combined host:port listening address for the HTTP server.
func (c *Config) Address() string {
	if c.Host == "" || c.Host == "0.0.0.0" {
		return ":" + c.Port
	}
	return c.Host + ":" + c.Port
}

// getEnv retrieves an environment variable or returns the default value if unset or empty.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt retrieves an integer environment variable or returns the default value if unset or invalid.
func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
