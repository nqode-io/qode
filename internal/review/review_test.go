package review

import (
	"os"
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

func TestBuildCodePrompt_PctConstraints(t *testing.T) {
	root := t.TempDir()
	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// Custom 10-point rubric: two dimensions × weight 5.
	// Expected pct thresholds: 50%→5.0, 75%→7.5, 80%→8.0, 100%→10.0.
	cfg := &config.Config{
		Scoring: config.ScoringConfig{
			Rubrics: map[string]config.RubricConfig{
				"review": {
					Dimensions: []config.DimensionConfig{
						{Name: "Dimension A", Weight: 5, Description: "first"},
						{Name: "Dimension B", Weight: 5, Description: "second"},
					},
				},
			},
		},
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
	}

	got, err := BuildCodePrompt(engine, cfg, ctx, "")
	if err != nil {
		t.Fatalf("BuildCodePrompt: %v", err)
	}

	for _, want := range []string{"5.0", "7.5", "8.0", "10.0"} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt must contain pct threshold %q", want)
		}
	}
	if !strings.Contains(got, "10") {
		t.Error("prompt must show rubric total 10")
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

func TestBuildCodePrompt_ContainsProjectName(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "myproject")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	engine, err := prompt.NewEngine(root)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	ctx := &context.Context{
		Branch:     "test-branch",
		ContextDir: filepath.Join(root, ".qode", "branches", "test-branch"),
	}

	got, err := BuildCodePrompt(engine, &config.Config{}, ctx, "")
	if err != nil {
		t.Fatalf("BuildCodePrompt: %v", err)
	}

	if !strings.Contains(got, "myproject") {
		t.Errorf("prompt must contain project name %q derived from root dir", "myproject")
	}
}

func TestBuildSecurityPrompt_ContainsProjectName(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "myproject")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

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

	if !strings.Contains(got, "myproject") {
		t.Errorf("prompt must contain project name %q derived from root dir", "myproject")
	}
}
