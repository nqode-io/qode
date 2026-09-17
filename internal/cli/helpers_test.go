package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nqode/qode/internal/qodecontext"
)

// Shared YAML config constants for tests — eliminates scattered inline YAML.
const (
	testYAMLMinimal       = "project:\n  name: test\n"
	testYAMLWithStack     = "project:\n  name: test\n  stack: go\n"
	testYAMLStrictMode    = "scoring:\n  strict: true\n"
	testYAMLNonStrict     = "scoring:\n  strict: false\n"
	testYAMLFullNonStrict = "project:\n  name: test\n  stack: go\nscoring:\n  strict: false\n"
)

// isolateHome redirects the user home directory so config.Load cannot merge the
// developer's own ~/.qode/config.yaml into a test's configuration. t.Setenv
// forbids t.Parallel, which is why every test reaching config.Load runs serially.
func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

// setupTestRoot creates a temp dir with .qode/contexts/<context>/ and an
// active "current" symlink, sets flagRoot, and returns root.
// contextName is also used as the display name for the context.
func setupTestRoot(t *testing.T, contextName string) string {
	t.Helper()
	isolateHome(t)
	root := t.TempDir()
	flagRoot = root

	if err := qodecontext.Init(context.Background(), root, contextName); err != nil {
		t.Fatalf("qodecontext.Init: %v", err)
	}
	if err := qodecontext.Switch(context.Background(), root, contextName); err != nil {
		t.Fatalf("qodecontext.Switch: %v", err)
	}

	t.Cleanup(func() { flagRoot = "" })
	return root
}

// setupTestRootWithConfig creates a test root with context layout and a
// qode.yaml config file, returning the root path.
func setupTestRootWithConfig(t *testing.T, yamlContent string) string {
	t.Helper()
	root := setupTestRoot(t, "test-context")
	writeConfigFile(t, root, yamlContent)
	return root
}

// writeConfigFile writes content to qode.yaml in root.
func writeConfigFile(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}
}

// writeContextFile writes content to a file in the active context directory.
func writeContextFile(t *testing.T, root, name, content string) {
	t.Helper()
	contextDir := filepath.Join(root, ".qode", "contexts", "test-context")
	path := filepath.Join(contextDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}
