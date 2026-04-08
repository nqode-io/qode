package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRoot initialises a real git repo on the given branch, sets flagRoot,
// creates the .qode/branches/<branch>/context/ directory, and returns root.
func setupTestRoot(t *testing.T, branch string) string {
	t.Helper()
	root := t.TempDir()
	flagRoot = root

	gitCmds := [][]string{
		{"init", "-b", branch},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range gitCmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	branchDir := filepath.Join(root, ".qode", "branches", branch)
	if err := os.MkdirAll(filepath.Join(branchDir, "context"), 0755); err != nil {
		t.Fatalf("MkdirAll branch dir: %v", err)
	}

	t.Cleanup(func() { flagRoot = "" })
	return root
}

// setupTestRootWithConfig creates a test root with git repo, branch context dir,
// and a qode.yaml config file, returning the root path.
func setupTestRootWithConfig(t *testing.T, branch, yamlContent string) string {
	t.Helper()
	root := setupTestRoot(t, branch)
	if err := os.WriteFile(filepath.Join(root, "qode.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("WriteFile qode.yaml: %v", err)
	}
	return root
}

// writeBranchFile writes content to a file in the branch context dir.
func writeBranchFile(t *testing.T, root, branch, name, content string) {
	t.Helper()
	path := filepath.Join(root, ".qode", "branches", branch, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}
