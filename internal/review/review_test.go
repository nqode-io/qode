package review

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/context"
	"github.com/nqode/qode/internal/prompt"
)

func TestBuildCodePrompt_OmitsDiffAndSpec(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
		Spec:       "spec sentinel content here",
	}

	got, err := BuildCodePrompt(engine, &config.Config{}, ctx, "")
	if err != nil {
		t.Fatalf("BuildCodePrompt: %v", err)
	}

	if strings.Contains(got, "spec sentinel content here") {
		t.Error("prompt must not inline spec content")
	}
	if !strings.Contains(got, "spec.md") {
		t.Error("prompt must reference spec.md")
	}
	if !strings.Contains(got, "diff.md") {
		t.Error("prompt must reference diff.md")
	}
}

func TestBuildSecurityPrompt_OmitsDiff(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
	}

	got, err := BuildSecurityPrompt(engine, &config.Config{}, ctx, "")
	if err != nil {
		t.Fatalf("BuildSecurityPrompt: %v", err)
	}

	if !strings.Contains(got, "diff.md") {
		t.Error("prompt must reference diff.md")
	}
}
