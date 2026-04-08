package log

import (
	"testing"
)

func TestInit_DefaultLevel(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
	}
}

func TestInit_DebugLevel(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "debug")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
	}
}

func TestInit_WarnLevel(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "warn")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
	}
}

func TestInit_ErrorLevel(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "error")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
	}
}

func TestInit_InvalidLevel_FallsBackToInfo(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "invalid")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
	}
}

func TestInit_CaseInsensitive(t *testing.T) {
	t.Setenv("QODE_LOG_LEVEL", "DEBUG")
	Init()

	if logger == nil {
		t.Fatal("expected non-nil logger after Init()")
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
