package config

import "os"

// Config holds the runtime configuration for both the adserver and worker
// binaries. Phase 0 keeps this intentionally minimal; later phases extend
// it with Postgres, Redis, S3, GeoIP, and signing-key fields.
type Config struct {
	Port     string
	LogLevel string
}

// Load reads configuration from the process environment, applying
// development-friendly defaults when variables are unset.
func Load() Config {
	return Config{
		Port:     getenv("PORT", "8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
