package cli

import (
	"testing"
)

// TestParseBranchArgs_SingleDot verifies that "." is silently dropped because
// it would resolve to the branches root directory rather than a named branch.
func TestParseBranchArgs_SingleDot(t *testing.T) {
	result := parseBranchArgs([]string{"."})
	if len(result) != 0 {
		t.Errorf("expected empty result for \".\", got %v", result)
	}
}

// TestParseBranchArgs_DotInList verifies that "." is filtered out while valid
// branch names in the same list are preserved.
func TestParseBranchArgs_DotInList(t *testing.T) {
	result := parseBranchArgs([]string{"feat/my-feature,.,main"})
	for _, b := range result {
		if b == "." {
			t.Errorf("\".\", should have been filtered, got %v", result)
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 branches, got %v", result)
	}
}

// TestParseBranchArgs_DotDot verifies that ".." is still blocked.
func TestParseBranchArgs_DotDot(t *testing.T) {
	result := parseBranchArgs([]string{".."})
	if len(result) != 0 {
		t.Errorf("expected empty result for \"..\", got %v", result)
	}
}

// TestParseBranchArgs_SlashedBranch verifies that slashed branch names are
// accepted (sanitization happens downstream in context.Load).
func TestParseBranchArgs_SlashedBranch(t *testing.T) {
	result := parseBranchArgs([]string{"feat/jira-123"})
	if len(result) != 1 || result[0] != "feat/jira-123" {
		t.Errorf("expected [\"feat/jira-123\"], got %v", result)
	}
}
