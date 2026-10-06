// Package config reads process settings from the environment.
// Invalid values fail at startup, before the process binds a port.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort       = "8080"
	defaultCORSOrigin = "http://localhost:5173"
)

const defaultShutdownTimeout = 5 * time.Second

// Config is the validated runtime configuration for one process.
type Config struct {
	Port            string
	LogLevel        slog.Level
	CORSOrigin      string
	ShutdownTimeout time.Duration
	DemoGenerator   bool
}

// Load reads configuration from the environment. Empty values fall back to
// the defaults the API used before this package existed.
func Load() (Config, error) {
	port, err := portFromEnv()
	if err != nil {
		return Config{}, err
	}

	level, err := levelFromEnv()
	if err != nil {
		return Config{}, err
	}

	origin, err := originFromEnv()
	if err != nil {
		return Config{}, err
	}

	timeout, err := timeoutFromEnv()
	if err != nil {
		return Config{}, err
	}

	demo, err := boolFromEnv("DEMO_GENERATOR", true)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:            port,
		LogLevel:        level,
		CORSOrigin:      origin,
		ShutdownTimeout: timeout,
		DemoGenerator:   demo,
	}, nil
}

func portFromEnv() (string, error) {
	raw, ok := os.LookupEnv("PORT")
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultPort, nil
	}

	raw = strings.TrimSpace(raw)
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("PORT must be a number from 1 to 65535")
	}

	return strconv.Itoa(n), nil
}

func levelFromEnv() (slog.Level, error) {
	raw, ok := os.LookupEnv("LOG_LEVEL")
	if !ok || strings.TrimSpace(raw) == "" {
		return slog.LevelInfo, nil
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

func originFromEnv() (string, error) {
	raw, ok := os.LookupEnv("CORS_ORIGIN")
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultCORSOrigin, nil
	}

	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("CORS_ORIGIN must be an origin like http://localhost:5173")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("CORS_ORIGIN must be an origin like http://localhost:5173")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("CORS_ORIGIN must use http or https")
	}

	return parsed.Scheme + "://" + parsed.Host, nil
}

func timeoutFromEnv() (time.Duration, error) {
	raw, ok := os.LookupEnv("SHUTDOWN_TIMEOUT")
	if !ok || strings.TrimSpace(raw) == "" {
		return defaultShutdownTimeout, nil
	}

	timeout, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || timeout <= 0 {
		return 0, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration such as 5s")
	}

	return timeout, nil
}

// BootstrapAPIKey is the local workspace key. Empty means no key is seeded.
// It is a development credential, in the same class as the local database password.
func BootstrapAPIKey() (string, error) {
	raw := strings.TrimSpace(os.Getenv("BOOTSTRAP_API_KEY"))
	if raw == "" {
		return "", nil
	}
	if len(raw) < 20 || strings.ContainsAny(raw, " \t\r\n") {
		return "", fmt.Errorf("BOOTSTRAP_API_KEY must be at least 20 characters and contain no whitespace")
	}
	return raw, nil
}

// DatabaseURL returns the Postgres URL. The API and migrate command both
// refuse to start without one, because the event feed lives in the database.
func DatabaseURL() (string, error) {
	raw, ok := os.LookupEnv("DATABASE_URL")
	if !ok || strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("DATABASE_URL is required")
	}

	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return "", fmt.Errorf("DATABASE_URL must be a postgres URL")
	}

	return raw, nil
}

func boolFromEnv(key string, fallback bool) (bool, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", key)
	}
}
