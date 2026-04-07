package log

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is the package-level structured logger.
// Initialised to slog.Default() so callers never encounter a nil Logger before Init() is called.
var Logger = slog.Default()

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
	Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// Info logs at info level.
func Info(msg string, args ...any) { Logger.Info(msg, args...) }

// Warn logs at warn level.
func Warn(msg string, args ...any) { Logger.Warn(msg, args...) }

// Error logs at error level.
func Error(msg string, args ...any) { Logger.Error(msg, args...) }

// Debug logs at debug level.
func Debug(msg string, args ...any) { Logger.Debug(msg, args...) }
