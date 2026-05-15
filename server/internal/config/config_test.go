package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port: want 8080, got %q", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: want info, got %q", cfg.LogLevel)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("Port override failed: got %q", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel override failed: got %q", cfg.LogLevel)
	}
}
