// Package log builds the process logger.
package log

import (
	"log/slog"
	"os"
)

// New returns a JSON logger. One object per line stays readable when the
// API runs under Compose alongside other services.
func New(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
