package scaffold

import (
	"fmt"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

const cursorCommandsDir = ".cursor/commands"

var cursorCommands = []string{
	"qode-plan-refine",
	"qode-plan-spec",
	"qode-review-code",
	"qode-review-security",
	"qode-check",
	"qode-start",
	"qode-ticket-fetch",
	"qode-knowledge-add-context",
	"qode-knowledge-add-branch",
}

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(root string) error {
	if err := iokit.EnsureDir(filepath.Join(root, cursorCommandsDir)); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.TemplateData{Project: prompt.TemplateProject{Name: filepath.Base(root)}}

	for _, cmd := range cursorCommands {
		content, err := engine.Render("scaffold/"+cmd+".cursor", data)
		if err != nil {
			return fmt.Errorf("render %s: %w", cmd, err)
		}
		p := filepath.Join(root, cursorCommandsDir, cmd+".mdc")
		if err := iokit.WriteFile(p, []byte(content), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("  Cursor: .cursor/commands/ (%d commands)\n", len(cursorCommands))
	return nil
}
