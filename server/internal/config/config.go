package config

import "os"

// Config holds the runtime configuration for both the adserver and worker
// binaries. Later phases extend this with Redis, GeoIP, and signing-key
// fields.
type Config struct {
	Port        string
	LogLevel    string
	DatabaseURL string
}

// Load reads configuration from the process environment, applying
// development-friendly defaults when variables are unset.
func Load() Config {
	return Config{
		Port:        getenv("PORT", "8080"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
