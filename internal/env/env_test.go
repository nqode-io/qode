package env

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, dotEnvFile), []byte(content), 0600); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := Load(dir); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestLoad_SetsUnsetVar(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "TEST_QODE_FOO=bar\n")
	os.Unsetenv("TEST_QODE_FOO")
	t.Cleanup(func() { os.Unsetenv("TEST_QODE_FOO") })

	if err := Load(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := os.Getenv("TEST_QODE_FOO"); got != "bar" {
		t.Errorf("expected %q, got %q", "bar", got)
	}
}

func TestLoad_DoesNotOverrideExistingVar(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "TEST_QODE_BAZ=from_env_file\n")
	t.Setenv("TEST_QODE_BAZ", "existing")

	if err := Load(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := os.Getenv("TEST_QODE_BAZ"); got != "existing" {
		t.Errorf("expected %q, got %q", "existing", got)
	}
}

func TestLoad_DoesNotOverrideEmptyVar(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "TEST_QODE_EMPTY=from_env_file\n")
	t.Setenv("TEST_QODE_EMPTY", "")

	if err := Load(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := os.Getenv("TEST_QODE_EMPTY"); got != "" {
		t.Errorf("expected empty string (existing), got %q", got)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "")

	if err := Load(dir); err != nil {
		t.Errorf("expected nil for empty file, got %v", err)
	}
}

func TestLoad_MultipleVars(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, "TEST_QODE_A=1\nTEST_QODE_B=2\n")
	os.Unsetenv("TEST_QODE_A")
	os.Unsetenv("TEST_QODE_B")
	t.Cleanup(func() {
		os.Unsetenv("TEST_QODE_A")
		os.Unsetenv("TEST_QODE_B")
	})

	if err := Load(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := os.Getenv("TEST_QODE_A"); got != "1" {
		t.Errorf("TEST_QODE_A: expected %q, got %q", "1", got)
	}
	if got := os.Getenv("TEST_QODE_B"); got != "2" {
		t.Errorf("TEST_QODE_B: expected %q, got %q", "2", got)
	}
}

func TestLoad_MalformedFile(t *testing.T) {
	dir := t.TempDir()
	// Unclosed quoted value is a parse error in godotenv.
	writeEnvFile(t, dir, "FOO='unclosed\n")
	if err := Load(dir); err == nil {
		t.Error("expected error for malformed .env, got nil")
	}
}

func TestLoad_EmptyProjectRoot_UsesCwd(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir) // temp dir has no .env
	if err := Load(""); err != nil {
		t.Errorf("expected nil for missing .env in cwd, got %v", err)
	}
}
