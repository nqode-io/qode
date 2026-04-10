package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSanitizeBranchName verifies that slashes are replaced with "--" and
// names without slashes pass through unchanged.
func TestSanitizeBranchName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"main", "main"},
		{"feat/my-feature", "feat--my-feature"},
		{"feat/jira-123/description", "feat--jira-123--description"},
		{"fix/bug-456", "fix--bug-456"},
		{"", ""},
		// Trailing slash edge case.
		{"feat/", "feat--"},
		// Double slash edge case.
		{"feat//bar", "feat----bar"},
		// Backslash is not replaced (not a path separator on Unix/Mac).
		{`feat\bar`, `feat\bar`},
	}
	for _, c := range cases {
		if got := SanitizeBranchName(c.input); got != c.want {
			t.Errorf("SanitizeBranchName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

// initGitRepo creates a temp directory with a git repo initialised on the
// given branch and one empty root commit, mirroring the pattern used in the
// CLI tests.
func initGitRepo(t *testing.T, branch string) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", branch},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

// gitRun executes a git command in root, failing the test on error.
func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writeFile creates a file at path with content and stages it.
func writeAndStage(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
	gitRun(t, root, "add", name)
}

func TestDefaultBranch_FallsBackToMain(t *testing.T) {
	t.Parallel()
	// A local-only repo has no origin/HEAD, so DefaultBranch must return "main".
	root := initGitRepo(t, "main")
	got, err := DefaultBranch(context.Background(), root)
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if got != "main" {
		t.Errorf("DefaultBranch() = %q, want %q", got, "main")
	}
}

func TestDefaultBranch_StripsOriginPrefix(t *testing.T) {
	t.Parallel()
	// Create a local repo and manually set the symbolic-ref so origin/HEAD points to "develop".
	root := initGitRepo(t, "develop")
	// Add a fake remote entry and set origin/HEAD directly without needing a real remote.
	for _, args := range [][]string{
		{"remote", "add", "origin", "https://example.com/fake-repo.git"},
		{"symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	got, err := DefaultBranch(context.Background(), root)
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if got != "develop" {
		t.Errorf("DefaultBranch() = %q, want %q", got, "develop")
	}
}

func TestCurrentBranch(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "main")
	}

	// Switch to a new branch and verify.
	gitRun(t, root, "checkout", "-b", "feature")
	branch, err = CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch after checkout: %v", err)
	}
	if branch != "feature" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "feature")
	}
}

func TestCurrentBranch_NotGitRepo(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := CurrentBranch(root)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestCreateBranch_FromHEAD(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	if err := CreateBranch(root, "new-feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "new-feature" {
		t.Errorf("after CreateBranch, CurrentBranch() = %q, want %q", branch, "new-feature")
	}
}

func TestCreateBranch_FromBase(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	writeAndStage(t, root, "base.go", "package main\n")
	gitRun(t, root, "commit", "-m", "add base")

	if err := CreateBranch(root, "from-main", "main"); err != nil {
		t.Fatalf("CreateBranch from main: %v", err)
	}
	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "from-main" {
		t.Errorf("CurrentBranch() = %q, want %q", branch, "from-main")
	}
}

func TestDeleteBranch(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	gitRun(t, root, "checkout", "-b", "to-delete")
	gitRun(t, root, "checkout", "main")

	if err := DeleteBranch(root, "to-delete"); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	// Verify branch no longer exists.
	out, _ := run(root, "branch", "--list", "to-delete")
	if strings.TrimSpace(out) != "" {
		t.Errorf("branch 'to-delete' still exists after deletion")
	}
}

func TestDeleteBranch_NonExistent(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	err := DeleteBranch(root, "no-such-branch")
	if err == nil {
		t.Fatal("expected error for deleting non-existent branch")
	}
}

// TestDiffFromBase_CommittedOnly verifies that committed changes on a feature
// branch are returned when there are no uncommitted changes.
func TestDiffFromBase_CommittedOnly(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")
	writeAndStage(t, root, "hello.go", "package main\n")
	gitRun(t, root, "commit", "-m", "add hello")

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	if !strings.Contains(diff, "hello.go") {
		t.Errorf("expected hello.go in diff, got:\n%s", diff)
	}
}

// TestDiffFromBase_UncommittedOnly verifies that uncommitted (staged) changes
// are captured even when no commits exist on the feature branch yet.
func TestDiffFromBase_UncommittedOnly(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")
	// Stage a file but do not commit.
	writeAndStage(t, root, "staged.go", "package main\n")

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	if !strings.Contains(diff, "staged.go") {
		t.Errorf("expected staged.go in diff, got:\n%s", diff)
	}
}

// TestDiffFromBase_CommittedAndUncommitted verifies that both committed and
// staged uncommitted changes appear in the diff.
func TestDiffFromBase_CommittedAndUncommitted(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")

	// Committed change.
	writeAndStage(t, root, "committed.go", "package main\n")
	gitRun(t, root, "commit", "-m", "committed")

	// Staged but not committed.
	writeAndStage(t, root, "staged.go", "package main\n")

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	if !strings.Contains(diff, "committed.go") {
		t.Errorf("expected committed.go in diff, got:\n%s", diff)
	}
	if !strings.Contains(diff, "staged.go") {
		t.Errorf("expected staged.go in diff, got:\n%s", diff)
	}
}

// TestDiffFromBase_NoChanges verifies that an empty string is returned when
// the branch has no committed or uncommitted changes relative to the base.
func TestDiffFromBase_NoChanges(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	if strings.TrimSpace(diff) != "" {
		t.Errorf("expected empty diff, got:\n%s", diff)
	}
}

// TestDiffFromBase_MultipleCommits verifies that changes from all commits on
// the branch (not just the latest) are present in the diff.
func TestDiffFromBase_MultipleCommits(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		writeAndStage(t, root, name, "package main\n")
		gitRun(t, root, "commit", "-m", "add "+name)
	}

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if !strings.Contains(diff, name) {
			t.Errorf("expected %s in diff, got:\n%s", name, diff)
		}
	}
}

// TestDiffFromBase_ExcludesQodeDir verifies that files inside .qode/ are not
// included in the diff regardless of changes.
func TestDiffFromBase_ExcludesQodeDir(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")

	// A regular file that should appear.
	writeAndStage(t, root, "app.go", "package main\n")
	gitRun(t, root, "commit", "-m", "add app.go")

	// A file inside .qode/ that should NOT appear.
	qodeDir := filepath.Join(root, ".qode")
	if err := os.MkdirAll(qodeDir, 0755); err != nil {
		t.Fatalf("MkdirAll .qode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(qodeDir, "secret.md"), []byte("secret"), 0644); err != nil {
		t.Fatalf("WriteFile .qode/secret.md: %v", err)
	}
	gitRun(t, root, "add", ".qode/secret.md")
	gitRun(t, root, "commit", "-m", "add qode file")

	diff, err := DiffFromBase(root, "main")
	if err != nil {
		t.Fatalf("DiffFromBase: %v", err)
	}
	if !strings.Contains(diff, "app.go") {
		t.Errorf("expected app.go in diff, got:\n%s", diff)
	}
	if strings.Contains(diff, "secret.md") {
		t.Errorf("expected .qode/secret.md to be excluded from diff, got:\n%s", diff)
	}
}

// TestChangedFiles_ReturnsModifiedFiles verifies that ChangedFiles returns the
// names of all files modified on the branch.
func TestChangedFiles_ReturnsModifiedFiles(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")

	gitRun(t, root, "checkout", "-b", "feature")
	for _, name := range []string{"foo.go", "bar.go"} {
		writeAndStage(t, root, name, "package main\n")
	}
	gitRun(t, root, "commit", "-m", "add files")

	files, err := ChangedFiles(root, "main")
	if err != nil {
		t.Fatalf("ChangedFiles: %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}
	for _, want := range []string{"foo.go", "bar.go"} {
		if !fileSet[want] {
			t.Errorf("expected %s in changed files, got: %v", want, files)
		}
	}
}

func TestCurrentBranchCtx_CancelledContext(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := CurrentBranchCtx(ctx, root)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRunCtx_DeadlineExceeded(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t, "main")
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	_, err := runCtx(ctx, root, "status")
	if err == nil {
		t.Fatal("expected error for expired deadline")
	}
}
