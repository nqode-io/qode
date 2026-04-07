package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

const cursorCommandsDir = ".cursor/commands"

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(root string) error {
	if err := os.MkdirAll(filepath.Join(root, cursorCommandsDir), 0755); err != nil {
		return err
	}

	name := filepath.Base(root)
	cmds := cursorSlashCommands(name)
	for cmdName, content := range cmds {
		p := filepath.Join(root, cursorCommandsDir, cmdName+".mdc")
		if err := writeFile(p, content); err != nil {
			return err
		}
	}

	fmt.Printf("  Cursor: .cursor/commands/ (%d commands)\n", len(cmds))
	return nil
}

func cursorSlashCommands(name string) map[string]string {
	m := make(map[string]string, len(allCommands))
	for _, cmd := range allCommands {
		desc := fmt.Sprintf(cmd.Description, name)
		m[cmd.Name] = fmt.Sprintf("---\ndescription: %s\n---\n\n%s", desc, cmd.Body)
	}
	return m
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
