// Package git provides thin wrappers around git CLI operations.
package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNoBaseBranch is returned when no base branch (main, master, develop) can be found.
var ErrNoBaseBranch = errors.New("could not find base branch")

// SanitizeBranchName returns a filesystem-safe directory name for the branch,
// replacing "/" with "--" so slashed branch names (e.g. feat/jira-123) map to
// a single flat directory rather than nested subdirectories.
func SanitizeBranchName(name string) string {
	return strings.ReplaceAll(name, "/", "--")
}

// CurrentBranch returns the current git branch name.
func CurrentBranch(root string) (string, error) {
	out, err := run(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// CreateBranch creates a new git branch from base (or HEAD if base is empty).
func CreateBranch(root, name, base string) error {
	if base == "" {
		_, err := run(root, "checkout", "-b", name)
		return err
	}
	_, err := run(root, "checkout", "-b", name, base)
	return err
}

// DeleteBranch deletes a git branch locally.
func DeleteBranch(root, name string) error {
	_, err := run(root, "branch", "-d", name)
	return err
}

// DiffFromBase returns the unified diff of all changes on the current branch
// since the merge-base with baseBranch (defaults to "main" then "master"),
// including both committed and uncommitted (staged + unstaged) changes.
func DiffFromBase(root, baseBranch string) (string, error) {
	uncommitted, err := run(root, "diff", "HEAD", "--", ":(exclude).qode/")
	if err != nil {
		uncommitted = ""
	}

	base, err := resolveBase(root, baseBranch)
	if err != nil {
		// No base branch found; return uncommitted changes only.
		return strings.TrimRight(uncommitted, "\n"), nil
	}

	committed, err := run(root, "diff", base+"...HEAD", "--", ":(exclude).qode/")
	if err != nil {
		return "", err
	}

	result := strings.TrimRight(committed, "\n")
	if u := strings.TrimSpace(uncommitted); u != "" {
		if result != "" {
			result += "\n"
		}
		result += u
	}
	return result, nil
}

// ChangedFiles returns files changed since the merge-base.
func ChangedFiles(root, baseBranch string) ([]string, error) {
	base, err := resolveBase(root, baseBranch)
	if err != nil {
		return nil, err
	}
	out, err := run(root, "diff", "--name-only", base+"...HEAD")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func resolveBase(root, preferred string) (string, error) {
	if preferred != "" {
		if _, err := run(root, "rev-parse", "--verify", preferred); err == nil {
			out, err := run(root, "merge-base", "HEAD", preferred)
			if err == nil {
				return strings.TrimSpace(out), nil
			}
		}
	}
	for _, candidate := range []string{"main", "master", "develop"} {
		if _, err := run(root, "rev-parse", "--verify", candidate); err == nil {
			out, err := run(root, "merge-base", "HEAD", candidate)
			if err == nil {
				return strings.TrimSpace(out), nil
			}
		}
	}
	return "", ErrNoBaseBranch
}

func run(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), string(exitErr.Stderr), exitErr)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
