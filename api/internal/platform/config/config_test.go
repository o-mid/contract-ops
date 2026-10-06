package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PORT", "LOG_LEVEL", "CORS_ORIGIN", "SHUTDOWN_TIMEOUT", "DEMO_GENERATOR"} {
		t.Setenv(key, "")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.CORSOrigin != "http://localhost:5173" {
		t.Fatalf("CORSOrigin = %q", cfg.CORSOrigin)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if !cfg.DemoGenerator {
		t.Fatal("expected demo generator to stay on by default")
	}
}

func TestLoadOverrides(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("PORT", "09090")
	t.Setenv("LOG_LEVEL", "WARN")
	t.Setenv("CORS_ORIGIN", "https://ops.example.com/")
	t.Setenv("SHUTDOWN_TIMEOUT", "2s")
	t.Setenv("DEMO_GENERATOR", "off")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "9090" {
		t.Fatalf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelWarn {
		t.Fatalf("LogLevel = %v, want warn", cfg.LogLevel)
	}
	if cfg.CORSOrigin != "https://ops.example.com" {
		t.Fatalf("CORSOrigin = %q", cfg.CORSOrigin)
	}
	if cfg.ShutdownTimeout != 2*time.Second {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if cfg.DemoGenerator {
		t.Fatal("expected demo generator to be off")
	}
}

func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}

	t.Setenv("DATABASE_URL", "http://localhost/db")
	if _, err := DatabaseURL(); err == nil {
		t.Fatal("expected non-postgres DATABASE_URL to fail")
	}

	t.Setenv("DATABASE_URL", "postgres://contract_ops:contract_ops@localhost:5432/contract_ops?sslmode=disable")
	got, err := DatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected url")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{name: "port", key: "PORT", val: "0", want: "PORT"},
		{name: "level", key: "LOG_LEVEL", val: "verbose", want: "LOG_LEVEL"},
		{name: "origin", key: "CORS_ORIGIN", val: "localhost:5173", want: "CORS_ORIGIN"},
		{name: "origin path", key: "CORS_ORIGIN", val: "http://localhost:5173/app", want: "CORS_ORIGIN"},
		{name: "timeout", key: "SHUTDOWN_TIMEOUT", val: "soon", want: "SHUTDOWN_TIMEOUT"},
		{name: "generator", key: "DEMO_GENERATOR", val: "maybe", want: "DEMO_GENERATOR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(test.key, test.val)

			_, err := Load()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error %q does not mention %s", err, test.want)
			}
		})
	}
}

func TestBootstrapAPIKey(t *testing.T) {
	t.Setenv("BOOTSTRAP_API_KEY", "")
	key, err := BootstrapAPIKey()
	if err != nil || key != "" {
		t.Fatalf("empty key = %q %v", key, err)
	}

	t.Setenv("BOOTSTRAP_API_KEY", "short")
	if _, err := BootstrapAPIKey(); err == nil {
		t.Fatal("expected a short key to fail")
	}

	t.Setenv("BOOTSTRAP_API_KEY", "co_local_dev_key_not_for_production")
	key, err = BootstrapAPIKey()
	if err != nil || key != "co_local_dev_key_not_for_production" {
		t.Fatalf("key = %q %v", key, err)
	}
}
