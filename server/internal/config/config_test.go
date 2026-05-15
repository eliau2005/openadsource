package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("S3_ENDPOINT", "")
	t.Setenv("S3_BUCKET", "")
	t.Setenv("S3_ACCESS_KEY_ID", "")
	t.Setenv("S3_SECRET_ACCESS_KEY", "")
	t.Setenv("S3_FORCE_PATH_STYLE", "")
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port: want 8080, got %q", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: want info, got %q", cfg.LogLevel)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL: want empty, got %q", cfg.DatabaseURL)
	}
	if cfg.PublicBaseURL != "http://localhost:8080" {
		t.Errorf("PublicBaseURL default wrong: got %q", cfg.PublicBaseURL)
	}
	if cfg.S3Region != "us-east-1" {
		t.Errorf("S3Region default wrong: got %q", cfg.S3Region)
	}
	if !cfg.S3ForcePathStyle {
		t.Errorf("S3ForcePathStyle default: want true, got false")
	}
	if cfg.S3Configured() {
		t.Errorf("S3Configured: should be false when no S3 vars set")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_URL", "postgres://example/db")
	t.Setenv("PUBLIC_BASE_URL", "https://ads.example.com")
	t.Setenv("S3_ENDPOINT", "http://minio:9000")
	t.Setenv("S3_BUCKET", "openadsource")
	t.Setenv("S3_ACCESS_KEY_ID", "abc")
	t.Setenv("S3_SECRET_ACCESS_KEY", "def")
	t.Setenv("S3_FORCE_PATH_STYLE", "false")
	t.Setenv("S3_PUBLIC_BASE_URL", "http://localhost:9000/openadsource")
	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("Port override failed: got %q", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel override failed: got %q", cfg.LogLevel)
	}
	if cfg.DatabaseURL != "postgres://example/db" {
		t.Errorf("DatabaseURL override failed: got %q", cfg.DatabaseURL)
	}
	if cfg.PublicBaseURL != "https://ads.example.com" {
		t.Errorf("PublicBaseURL override failed: got %q", cfg.PublicBaseURL)
	}
	if cfg.S3ForcePathStyle {
		t.Errorf("S3ForcePathStyle override failed: want false, got true")
	}
	if !cfg.S3Configured() {
		t.Errorf("S3Configured: should be true when all S3 vars set")
	}
}
