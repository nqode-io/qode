package qodecontext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupContextsDir creates a temp root with .qode/contexts/ and returns root.
func setupContextsDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, ".qode", "contexts")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

// TestLoad_NoCurrentContext verifies that Load returns ErrNoCurrentContext when
// no symlink exists.
func TestLoad_NoCurrentContext(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	_, err := Load(context.Background(), root)
	if !errors.Is(err, ErrNoCurrentContext) {
		t.Errorf("want ErrNoCurrentContext, got: %v", err)
	}
}

// TestLoad_AfterInitAndSwitch verifies that Load succeeds after Init + Switch.
func TestLoad_AfterInitAndSwitch(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"my-feature"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Switch(context.Background(), root,"my-feature"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	ctx, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ctx.ContextName != "my-feature" {
		t.Errorf("ContextName = %q, want %q", ctx.ContextName, "my-feature")
	}
}

// TestInit_CreatesStubFiles verifies Init creates .ctx-name.md, ticket.md, notes.md.
func TestInit_CreatesStubFiles(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"feat-login"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	dir := filepath.Join(root, ".qode", "contexts", "feat-login")
	for _, f := range []string{ctxNameFile, "ticket.md", "notes.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
	}

	// .ctx-name.md must contain the context name.
	data, _ := os.ReadFile(filepath.Join(dir, ctxNameFile))
	if string(data) != "feat-login" {
		t.Errorf(".ctx-name.md = %q, want %q", string(data), "feat-login")
	}
}

// TestInit_AlreadyExists verifies Init returns an error when context dir exists.
func TestInit_AlreadyExists(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"dup"); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	err := Init(context.Background(), root,"dup")
	if err == nil {
		t.Fatal("expected error for duplicate context")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %v", err)
	}
}

// TestSwitch_ReplacesExistingSymlink verifies Switch updates an existing symlink.
func TestSwitch_ReplacesExistingSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"feat-a"); err != nil {
		t.Fatalf("Init feat-a: %v", err)
	}
	if err := Init(context.Background(), root,"feat-b"); err != nil {
		t.Fatalf("Init feat-b: %v", err)
	}
	if err := Switch(context.Background(), root,"feat-a"); err != nil {
		t.Fatalf("Switch feat-a: %v", err)
	}

	name, err := CurrentName(context.Background(), root)
	if err != nil {
		t.Fatalf("CurrentName after first switch: %v", err)
	}
	if name != "feat-a" {
		t.Errorf("want feat-a, got %q", name)
	}

	if err := Switch(context.Background(), root,"feat-b"); err != nil {
		t.Fatalf("Switch feat-b: %v", err)
	}

	name, err = CurrentName(context.Background(), root)
	if err != nil {
		t.Fatalf("CurrentName after second switch: %v", err)
	}
	if name != "feat-b" {
		t.Errorf("want feat-b, got %q", name)
	}
}

// TestSwitch_NonExistentContext returns an error.
func TestSwitch_NonExistentContext(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	err := Switch(context.Background(), root,"does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent context")
	}
}

// TestClear_PreservesCtxNameFile verifies that Clear keeps .ctx-name.md.
func TestClear_PreservesCtxNameFile(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"clear-test"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	dir := filepath.Join(root, ".qode", "contexts", "clear-test")
	writeFile(t, filepath.Join(dir, "spec.md"), "# Spec")
	writeFile(t, filepath.Join(dir, "refined-analysis.md"), "# Analysis")

	if err := Clear(context.Background(), root,"clear-test"); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ctxNameFile)); err != nil {
		t.Errorf(".ctx-name.md should be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "spec.md")); !os.IsNotExist(err) {
		t.Error("spec.md should have been removed")
	}

	// Stub files should be reinitialised.
	if _, err := os.Stat(filepath.Join(dir, "ticket.md")); err != nil {
		t.Errorf("ticket.md should be reinitialised: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.md")); err != nil {
		t.Errorf("notes.md should be reinitialised: %v", err)
	}
}

// TestClear_InvalidName verifies Clear rejects invalid explicit names.
func TestClear_InvalidName(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	cases := []struct {
		name    string
		wantMsg string
	}{
		{".", "reserved"},
		{"..", "reserved"},
		{"a/b", "path separators"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Clear(context.Background(), root,tc.name)
			if err == nil {
				t.Fatalf("expected error for name %q", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("want %q in error, got: %v", tc.wantMsg, err)
			}
		})
	}
}

// TestRemove_InvalidName verifies Remove rejects invalid explicit names.
func TestRemove_InvalidName(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	cases := []struct {
		name    string
		wantMsg string
	}{
		{".", "reserved"},
		{"..", "reserved"},
		{"a/b", "path separators"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Remove(context.Background(), root,tc.name)
			if err == nil {
				t.Fatalf("expected error for name %q", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("want %q in error, got: %v", tc.wantMsg, err)
			}
		})
	}
}

// TestRemove_DeletesDirAndSymlink verifies that Remove deletes the context and
// the "current" symlink when it points to the target.
func TestRemove_DeletesDirAndSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"rm-test"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Switch(context.Background(), root,"rm-test"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if err := Remove(context.Background(), root,"rm-test"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	dir := filepath.Join(root, ".qode", "contexts", "rm-test")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("context directory should be removed")
	}
	link := filepath.Join(root, ".qode", "contexts", currentLink)
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("current symlink should be removed")
	}
}

// TestRemove_PreservesSymlinkWhenPointingElsewhere verifies that Remove does
// not remove the "current" symlink when it points to a different context.
func TestRemove_PreservesSymlinkWhenPointingElsewhere(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"keep"); err != nil {
		t.Fatalf("Init keep: %v", err)
	}
	if err := Init(context.Background(), root,"gone"); err != nil {
		t.Fatalf("Init gone: %v", err)
	}
	if err := Switch(context.Background(), root,"keep"); err != nil {
		t.Fatalf("Switch keep: %v", err)
	}

	if err := Remove(context.Background(), root,"gone"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// "current" should still point to "keep".
	name, err := CurrentName(context.Background(), root)
	if err != nil {
		t.Fatalf("CurrentName: %v", err)
	}
	if name != "keep" {
		t.Errorf("want 'keep', got %q", name)
	}
}

// TestReset_RemovesSymlink verifies Reset removes the current symlink.
func TestReset_RemovesSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"resetme"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Switch(context.Background(), root,"resetme"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if err := Reset(context.Background(), root); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	link := filepath.Join(root, ".qode", "contexts", currentLink)
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Error("current symlink should be removed after Reset")
	}
}

// TestReset_NoopWhenNoSymlink verifies Reset is idempotent.
func TestReset_NoopWhenNoSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Reset(context.Background(), root); err != nil {
		t.Errorf("Reset with no symlink should not error: %v", err)
	}
}

// TestValidateContextName covers valid and invalid names.
func TestValidateContextName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"feat-login", false},
		{"my_feature", false},
		{"", true},
		{".", true},
		{"..", true},
		{"a/b", true},
		{"a\\b", true},
		{strings.Repeat("a", 256), true},
		{strings.Repeat("a", 255), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateContextName(tc.name)
			if tc.wantErr && err == nil {
				t.Errorf("name %q: expected error, got nil", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("name %q: unexpected error: %v", tc.name, err)
			}
		})
	}
}

// TestSanitizeContextName verifies "/" is replaced with "--".
func TestSanitizeContextName(t *testing.T) {
	t.Parallel()
	if got := SanitizeContextName("feat/jira-123"); got != "feat--jira-123" {
		t.Errorf("got %q, want %q", got, "feat--jira-123")
	}
}

// TestSafeContextDir_PathTraversal verifies path traversal is rejected.
func TestSafeContextDir_PathTraversal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := safeContextDir(root, "../../etc/passwd")
	if err == nil {
		t.Error("expected path traversal error")
	}
}

// TestCurrentName_DanglingSymlink verifies a dangling symlink returns an error.
func TestCurrentName_DanglingSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)
	base := filepath.Join(root, ".qode", "contexts")
	// Create a symlink pointing to a non-existent directory.
	link := filepath.Join(base, currentLink)
	if err := os.Symlink("nonexistent", link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := CurrentName(context.Background(), root)
	if err == nil {
		t.Fatal("expected error for dangling symlink")
	}
	if !errors.Is(err, ErrDanglingSymlink) {
		t.Errorf("want ErrDanglingSymlink, got: %v", err)
	}
}

// TestCurrentName_RegularFileNotSymlink verifies that a regular file at "current"
// produces an error about reading the symlink, not a dangling-symlink error.
func TestCurrentName_RegularFileNotSymlink(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)
	link := filepath.Join(root, ".qode", "contexts", currentLink)
	writeFile(t, link, "not-a-symlink")

	_, err := CurrentName(context.Background(), root)
	if err == nil {
		t.Fatal("expected error when current is a regular file")
	}
	if errors.Is(err, ErrDanglingSymlink) {
		t.Errorf("should not report dangling for non-symlink: %v", err)
	}
	if !strings.Contains(err.Error(), "reading current symlink") {
		t.Errorf("expected 'reading current symlink' error, got: %v", err)
	}
}

// TestValidateContextName_UnicodeByteBoundary verifies that the 255-character
// limit is enforced on byte length, matching filesystem limits.
func TestValidateContextName_UnicodeByteBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		desc    string
		name    string
		wantErr bool
	}{
		{
			desc:    "85 three-byte runes exactly 255 bytes",
			name:    strings.Repeat("\u4e2d", 85), // 85 * 3 = 255 bytes
			wantErr: false,
		},
		{
			desc:    "86 three-byte runes exceeds 255 bytes",
			name:    strings.Repeat("\u4e2d", 86), // 86 * 3 = 258 bytes
			wantErr: true,
		},
		{
			desc:    "254 ASCII bytes plus one two-byte rune equals 256 bytes",
			name:    strings.Repeat("a", 254) + "\u00e9", // 254 + 2 = 256 bytes
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			err := ValidateContextName(tc.name)
			if tc.wantErr && err == nil {
				t.Errorf("%s: expected error, got nil", tc.desc)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("%s: unexpected error: %v", tc.desc, err)
			}
		})
	}
}

// TestLoad_ScanMockupFiles verifies that image files are loaded as mockups.
func TestLoad_ScanMockupFiles(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"scan-test"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Switch(context.Background(), root,"scan-test"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	dir := filepath.Join(root, ".qode", "contexts", "scan-test")
	writeFile(t, filepath.Join(dir, "mockup.png"), "fake-png-bytes")

	ctx, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(ctx.Mockups) != 1 {
		t.Errorf("want 1 mockup, got %d", len(ctx.Mockups))
	}
}

// TestLoad_IterationParsing verifies iteration loading from numbered files.
func TestLoad_IterationParsing(t *testing.T) {
	t.Parallel()
	root := setupContextsDir(t)

	if err := Init(context.Background(), root,"iter-test"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := Switch(context.Background(), root,"iter-test"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	dir := filepath.Join(root, ".qode", "contexts", "iter-test")
	writeFile(t, filepath.Join(dir, "refined-analysis-1-score-20.md"), "iter 1")
	writeFile(t, filepath.Join(dir, "refined-analysis.md"), "<!-- qode:iteration=2 score=23/25 -->\n# Analysis")

	ctx, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(ctx.Iterations) != 2 {
		t.Fatalf("want 2 iterations, got %d", len(ctx.Iterations))
	}
}
