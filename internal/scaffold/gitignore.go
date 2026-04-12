package scaffold

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/iokit"
)

// GitignoreMarker is the section header written above qode's gitignore rules.
const GitignoreMarker = "# qode temp files"

// GitignoreRules lists the gitignore patterns that qode adds to .gitignore.
// Callers must not modify this slice.
var GitignoreRules = []string{
	".qode/contexts/*",
	".qode/prompts/scaffold/",
}

// AppendGitignoreRules adds qode-specific patterns to .gitignore in root.
// Each rule is checked individually; only missing rules are appended.
// Prints a confirmation to out when rules are written; silent on no-op.
func AppendGitignoreRules(ctx context.Context, out io.Writer, root string) error {
	path := filepath.Join(root, ".gitignore")

	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}
	existing := string(data)

	markerPresent := strings.Contains(existing, GitignoreMarker)
	var missing []string
	for _, rule := range GitignoreRules {
		if !strings.Contains(existing, rule) {
			missing = append(missing, rule)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	var sb strings.Builder
	if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
		sb.WriteString("\n")
	}
	if !markerPresent {
		sb.WriteString(GitignoreMarker)
		sb.WriteString("\n")
	}
	for _, rule := range missing {
		sb.WriteString(rule)
		sb.WriteString("\n")
	}

	if err := iokit.WriteFileCtx(ctx, path, []byte(existing+sb.String()), 0644); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	_, _ = fmt.Fprintln(out, "Appended qode ignore rules to .gitignore")
	return nil
}
