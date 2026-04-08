package iokit_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nqode/qode/internal/iokit"
)

func TestReadFileOrString_FileExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	want := "hello world"
	if err := os.WriteFile(path, []byte(want), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	got := iokit.ReadFileOrString(path, "default")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReadFileOrString_FileMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")
	want := "fallback"
	got := iokit.ReadFileOrString(path, want)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "file.txt")
	data := []byte("content")
	if err := iokit.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestWriteFile_VerifyPermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	const perm os.FileMode = 0644
	if err := iokit.WriteFile(path, []byte("data"), perm); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != perm {
		t.Errorf("permissions: got %o, want %o", got, perm)
	}
}

func TestAtomicWrite_ContentCorrect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	want := "atomic content"
	if err := iokit.AtomicWrite(path, []byte(want), 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAtomicWrite_VerifyPermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	const perm os.FileMode = 0644
	if err := iokit.AtomicWrite(path, []byte("data"), perm); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != perm {
		t.Errorf("permissions: got %o, want %o", got, perm)
	}
}

func TestAtomicWrite_NoTempFileLeft(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	if err := iokit.AtomicWrite(path, []byte("data"), 0644); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "out.txt" {
			t.Errorf("unexpected file left in dir: %s", e.Name())
		}
	}
}

func TestEnsureDir_CreatesNestedDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "x", "y", "z")
	if err := iokit.EnsureDir(target); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected a directory at %s", target)
	}
}

func TestEnsureDir_Idempotent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "existing")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := iokit.EnsureDir(target); err != nil {
		t.Errorf("second EnsureDir: %v", err)
	}
}

func TestAtomicWrite_ReadOnlyDir(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root")
	}
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(roDir, 0555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	err := iokit.AtomicWrite(filepath.Join(roDir, "file.txt"), []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error writing to read-only directory")
	}
}

func TestWriteFile_ReadOnlyParent(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root")
	}
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(roDir, 0555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(roDir, 0755) })

	err := iokit.WriteFile(filepath.Join(roDir, "sub", "file.txt"), []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error writing under read-only parent")
	}
}

func TestEnsureDir_PathIsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Trying to create a dir at a path that is already a file should fail.
	err := iokit.EnsureDir(filepath.Join(filePath, "sub"))
	if err == nil {
		t.Fatal("expected error when path component is a file")
	}
}

func TestReadFileOrString_PermissionDenied(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte("secret"), 0000); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	got := iokit.ReadFileOrString(path, "fallback")
	if got != "fallback" {
		t.Errorf("expected fallback on permission denied, got %q", got)
	}
}
