package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

var claudeCommands = []string{
	"qode-plan-refine",
	"qode-plan-spec",
	"qode-review-code",
	"qode-review-security",
	"qode-check",
	"qode-start",
	"qode-ticket-fetch",
	"qode-knowledge-add-context",
	"qode-knowledge-add-branch",
	"qode-pr-create",
}

// SetupClaudeCode generates Claude Code configuration files.
func SetupClaudeCode(out io.Writer, root string) error {
	commandsDir := filepath.Join(root, ".claude", "commands")
	if err := iokit.EnsureDir(commandsDir); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root), "").
		WithIDE("claude").
		Build()

	for _, cmd := range claudeCommands {
		content, err := engine.Render("scaffold/"+cmd, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", cmd, err)
		}
		if err := iokit.WriteFile(filepath.Join(commandsDir, cmd+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  Claude Code: %d slash commands\n", len(claudeCommands))
	return nil
}
