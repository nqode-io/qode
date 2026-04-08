// Package iokit provides file I/O utilities including atomic writes and directory helpers.
package iokit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ReadFileOrString reads a file and returns defaultVal on any error.
// Any error (including permission denied) returns defaultVal.
func ReadFileOrString(path, defaultVal string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultVal
	}
	return string(data)
}

// WriteFile creates parent directories and writes data to path.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return WriteFileCtx(context.Background(), path, data, perm)
}

// WriteFileCtx is like WriteFile but respects context cancellation.
func WriteFileCtx(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// AtomicWrite writes via a temp file + rename to avoid partial writes.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	return AtomicWriteCtx(context.Background(), path, data, perm)
}

// AtomicWriteCtx is like AtomicWrite but respects context cancellation.
func AtomicWriteCtx(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("ensure dir %s: %w", path, err)
	}
	return nil
}
