package config

import (
	"os"
	"strconv"
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
	S3Endpoint         string
	S3Region           string
	S3Bucket           string
	S3AccessKeyID      string
	S3SecretAccessKey  string
	S3ForcePathStyle   bool
	S3PublicBaseURL    string
}

// Load reads configuration from the process environment, applying
// development-friendly defaults when variables are unset.
func Load() Config {
	return Config{
		Port:              getenv("PORT", "8080"),
		LogLevel:          getenv("LOG_LEVEL", "info"),
		DatabaseURL:       getenv("DATABASE_URL", ""),
		PublicBaseURL:     getenv("PUBLIC_BASE_URL", "http://localhost:8080"),
		S3Endpoint:        getenv("S3_ENDPOINT", ""),
		S3Region:          getenv("S3_REGION", "us-east-1"),
		S3Bucket:          getenv("S3_BUCKET", ""),
		S3AccessKeyID:     getenv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey: getenv("S3_SECRET_ACCESS_KEY", ""),
		S3ForcePathStyle:  getenvBool("S3_FORCE_PATH_STYLE", true),
		S3PublicBaseURL:   getenv("S3_PUBLIC_BASE_URL", ""),
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
