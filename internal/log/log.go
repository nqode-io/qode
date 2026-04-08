// Package log provides structured logging built on slog.
package log

import (
	"log/slog"
	"os"
	"strings"
)

// logger is the package-level structured logger.
// Initialised to slog.Default() so callers never encounter a nil logger before Init() is called.
// Init() must be called before any goroutines are started; after that, only the
// package-level functions (Info, Warn, Error, Debug) should be used.
var logger = slog.Default()

// Init configures the package-level logger.
// It respects the QODE_LOG_LEVEL environment variable (debug, info, warn, error).
// Default level is info.
func Init() {
	level := slog.LevelInfo
	if env := os.Getenv("QODE_LOG_LEVEL"); env != "" {
		switch strings.ToLower(env) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// Info logs at info level.
func Info(msg string, args ...any) { logger.Info(msg, args...) }

// Warn logs at warn level.
func Warn(msg string, args ...any) { logger.Warn(msg, args...) }

// Error logs at error level.
func Error(msg string, args ...any) { logger.Error(msg, args...) }

// Debug logs at debug level.
func Debug(msg string, args ...any) { logger.Debug(msg, args...) }
