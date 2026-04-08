package log

import (
	"context"
	"log/slog"
	"testing"
)

func TestInit_LogLevel(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		wantLevel slog.Level
	}{
		{"default is info", "", slog.LevelInfo},
		{"debug", "debug", slog.LevelDebug},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"invalid falls back to info", "invalid", slog.LevelInfo},
		{"case insensitive", "DEBUG", slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("QODE_LOG_LEVEL", tt.envValue)
			Init()

			ctx := context.Background()
			if !logger.Handler().Enabled(ctx, tt.wantLevel) {
				t.Errorf("expected level %v to be enabled", tt.wantLevel)
			}
			// Verify messages below the configured level are suppressed.
			if tt.wantLevel > slog.LevelDebug {
				belowLevel := tt.wantLevel - 4 // one severity step lower
				if logger.Handler().Enabled(ctx, belowLevel) {
					t.Errorf("expected level %v to be disabled (below %v)", belowLevel, tt.wantLevel)
				}
			}
		})
	}
}

func TestLogFunctions_NoPanic(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "debug")
	Init()

	Info("test info", "key", "value")
	Warn("test warn", "key", "value")
	Error("test error", "key", "value")
	Debug("test debug", "key", "value")
}
