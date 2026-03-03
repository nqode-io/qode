package dispatch

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// clipboardDispatcher copies the prompt to the system clipboard and returns
// ErrManualDispatch — the user must paste it into their IDE.
type clipboardDispatcher struct{}

func (c *clipboardDispatcher) Name() string      { return "clipboard" }
func (c *clipboardDispatcher) Available() bool   { return true }

func (c *clipboardDispatcher) Run(_ context.Context, prompt string, _ Options) (string, error) {
	if err := copyToClipboard(prompt); err != nil {
		return "", fmt.Errorf("clipboard: %w", err)
	}
	return "", ErrManualDispatch
}

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
	stdin.Close()
	return cmd.Wait()
}

// ClipboardInstruction returns the message to show when falling back to
// clipboard mode, given the command that would have been used.
func ClipboardInstruction(slashCmd string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"Prompt copied to clipboard.\nPaste into your IDE or use: /%s", slashCmd,
	))
}
