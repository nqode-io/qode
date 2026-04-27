package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

var codexCommands = []string{
	"qode-plan-refine",
	"qode-plan-spec",
	"qode-review-code",
	"qode-review-security",
	"qode-check",
	"qode-start",
	"qode-ticket-fetch",
	"qode-knowledge-add-context",
	"qode-pr-create",
	"qode-pr-resolve",
}

// SetupCodex generates Codex slash command files under <root>/.codex/commands/.
// It writes one .md file per command and reports the count to out.
func SetupCodex(out io.Writer, root string) error {
	commandsDir := filepath.Join(root, ".codex", "commands")
	if err := iokit.EnsureDir(commandsDir); err != nil {
		return fmt.Errorf("codex setup: %w", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root)).
		WithIDE("codex").
		Build()

	for _, cmd := range codexCommands {
		content, err := engine.Render("scaffold/"+cmd, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", cmd, err)
		}
		if err := iokit.WriteFile(filepath.Join(commandsDir, cmd+".md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  Codex: %d slash commands\n", len(codexCommands))
	return nil
}
