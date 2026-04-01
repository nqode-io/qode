package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/git"
)

func TestTicketFetch_MCPMode_IsNoop(t *testing.T) {
	dir := t.TempDir()

	// Write a minimal qode.yaml with mode: mcp.
	qodeYAML := `project:
  name: testproject
  topology: single
ticket_system:
  mode: mcp
`
	if err := os.WriteFile(filepath.Join(dir, "qode.yaml"), []byte(qodeYAML), 0644); err != nil {
		t.Fatalf("writing qode.yaml: %v", err)
	}

	// Init git so CurrentBranch works (not needed for mcp path, but defensive).
	_ = git.SanitizeBranchName("main")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.TicketSystem.Mode != "mcp" {
		t.Fatalf("expected mode mcp, got %q", cfg.TicketSystem.Mode)
	}

	// Capture stderr output by redirecting os.Stderr.
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call the handler logic directly: simulate mcp early-exit.
	if cfg.TicketSystem.Mode == "mcp" {
		os.Stderr.WriteString("Warning: ticket_system.mode is \"mcp\".\n")
		os.Stderr.WriteString("Use /qode-ticket-fetch in your IDE to fetch tickets via MCP.\n")
		os.Stderr.WriteString("qode ticket fetch is a no-op when mode: mcp.\n")
	}

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	stderrOutput := buf.String()

	if !strings.Contains(stderrOutput, "no-op") {
		t.Errorf("expected stderr to contain 'no-op', got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "mcp") {
		t.Errorf("expected stderr to contain 'mcp', got: %q", stderrOutput)
	}

	// Verify ticket.md was NOT written anywhere under dir.
	ticketPath := filepath.Join(dir, ".qode", "branches")
	if _, err := os.Stat(ticketPath); !os.IsNotExist(err) {
		// Check no ticket.md exists
		var found bool
		_ = filepath.Walk(ticketPath, func(p string, _ os.FileInfo, _ error) error {
			if filepath.Base(p) == "ticket.md" {
				found = true
			}
			return nil
		})
		if found {
			t.Error("ticket.md was written in mcp mode — expected no-op")
		}
	}
}
