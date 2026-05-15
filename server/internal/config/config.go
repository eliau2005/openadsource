package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the runtime configuration for the adserver and worker
// binaries. Later phases extend this with Redis, GeoIP, and signing-key
// fields.
type Config struct {
	Port          string
	LogLevel      string
	DatabaseURL   string
	PublicBaseURL string

	// S3-compatible object storage (optional — only consumed when
	// S3Configured() returns true).
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3ForcePathStyle  bool
	S3PublicBaseURL   string

	// Phase 3 — decision engine.
	RedisURL                string        // optional; budget enforcement falls back to a no-op stub when empty
	GeoIPDBPath             string        // optional; geo resolver stubs gracefully when missing
	TrustedProxies          string        // comma-separated CIDR list; controls X-Forwarded-For trust
	RegistryRefreshInterval time.Duration // TTL between snapshot reloads
}

// Load reads configuration from the process environment, applying
// development-friendly defaults when variables are unset.
func Load() Config {
	return Config{
		Port:                    getenv("PORT", "8080"),
		LogLevel:                getenv("LOG_LEVEL", "info"),
		DatabaseURL:             getenv("DATABASE_URL", ""),
		PublicBaseURL:           getenv("PUBLIC_BASE_URL", "http://localhost:8080"),
		S3Endpoint:              getenv("S3_ENDPOINT", ""),
		S3Region:                getenv("S3_REGION", "us-east-1"),
		S3Bucket:                getenv("S3_BUCKET", ""),
		S3AccessKeyID:           getenv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey:       getenv("S3_SECRET_ACCESS_KEY", ""),
		S3ForcePathStyle:        getenvBool("S3_FORCE_PATH_STYLE", true),
		S3PublicBaseURL:         getenv("S3_PUBLIC_BASE_URL", ""),
		RedisURL:                getenv("REDIS_URL", ""),
		GeoIPDBPath:             getenv("GEOIP_DB_PATH", ""),
		TrustedProxies:          getenv("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"),
		RegistryRefreshInterval: getenvDuration("REGISTRY_REFRESH_INTERVAL", 30*time.Second),
	}
}

// S3Configured reports whether enough S3 env vars are present to construct
// an S3 client. Resolver / seed code should consult this before assuming
// internal_s3 ads can be served via presigning or uploads can happen.
func (c Config) S3Configured() bool {
	return c.S3Endpoint != "" && c.S3Bucket != "" && c.S3AccessKeyID != "" && c.S3SecretAccessKey != ""
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
