package git

import (
	"fmt"
	"os/exec"
	"strings"
)

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

// DiffFromBase returns the unified diff of changes since the merge-base with
// baseBranch (defaults to "main" then "master").
// When the merge-base equals HEAD (e.g. you are on main), it falls back to
// the diff of the last commit (HEAD~1..HEAD).
func DiffFromBase(root, baseBranch string) (string, error) {
	base, err := resolveBase(root, baseBranch)
	if err != nil {
		// Fall back to staged + unstaged changes.
		out, err2 := run(root, "diff", "HEAD")
		if err2 != nil {
			return "", err
		}
		return out, nil
	}
	out, err := run(root, "diff", base+"...HEAD")
	if err != nil {
		return "", err
	}
	// When merge-base == HEAD the branch hasn't diverged (e.g. we're on main).
	// Fall back to the last commit so the review still has something to analyse.
	if strings.TrimSpace(out) == "" {
		out, err = run(root, "diff", "HEAD~1..HEAD")
		if err != nil {
			return "", err
		}
	}
	return out, nil
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
	return "", fmt.Errorf("could not find base branch")
}

func run(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
