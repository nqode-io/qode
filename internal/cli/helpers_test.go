package cli

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStdout redirects os.Stdout to a pipe for the duration of fn, then
// returns everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("w.Close: %v", err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("r.Close: %v", err)
	}
	return buf.String()
}
