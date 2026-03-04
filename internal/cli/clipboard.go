package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

// copyToClipboard copies text to the system clipboard.
// It silently does nothing if no clipboard tool is available.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		}
	case "windows":
		cmd = exec.Command("clip")
	}

	if cmd == nil {
		return fmt.Errorf("no clipboard tool found")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := fmt.Fprint(stdin, text); err != nil {
		return err
	}
	_ = stdin.Close()
	return cmd.Wait()
}
