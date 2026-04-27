package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/nqode/qode/internal/iokit"
	"github.com/nqode/qode/internal/prompt"
)

const cursorCommandsDir = ".cursor/commands"

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(out io.Writer, root string) error {
	if err := iokit.EnsureDir(filepath.Join(root, cursorCommandsDir)); err != nil {
		return err
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		return err
	}

	data := prompt.NewTemplateData(filepath.Base(root)).
		WithIDE("cursor").
		Build()

	for _, workflow := range qodeWorkflows {
		content, err := engine.Render("scaffold/"+workflow.Name, data)
		if err != nil {
			return fmt.Errorf("render %s: %w", workflow.Name, err)
		}
		p := filepath.Join(root, cursorCommandsDir, workflow.Name+".mdc")
		if err := iokit.WriteFile(p, []byte(content), 0644); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintf(out, "  Cursor: .cursor/commands/ (%d commands)\n", len(qodeWorkflows))
	return nil
}
