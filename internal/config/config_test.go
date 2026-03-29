package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Review.MinCodeScore != 8.0 {
		t.Errorf("expected MinCodeScore 8.0, got %.1f", cfg.Review.MinCodeScore)
	}
	if cfg.Scoring.RefineTargetScore != 25 {
		t.Errorf("expected RefineTargetScore 25, got %d", cfg.Scoring.RefineTargetScore)
	}
	if !cfg.IDE.Cursor.Enabled {
		t.Error("expected Cursor enabled by default")
	}
	if !cfg.IDE.ClaudeCode.Enabled {
		t.Error("expected ClaudeCode enabled by default")
	}
}

func TestConfigLayers_Shorthand(t *testing.T) {
	cfg := Config{
		Project: ProjectConfig{
			Stack: "nextjs",
			Test:  TestConfig{Unit: "npm test"},
		},
	}
	layers := cfg.Layers()
	if len(layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(layers))
	}
	if layers[0].Stack != "nextjs" {
		t.Errorf("expected nextjs, got %s", layers[0].Stack)
	}
}

func TestConfigLayers_Composite(t *testing.T) {
	cfg := Config{
		Project: ProjectConfig{
			Layers: []LayerConfig{
				{Name: "frontend", Stack: "react", Path: "./frontend"},
				{Name: "backend", Stack: "dotnet", Path: "./backend"},
			},
		},
	}
	layers := cfg.Layers()
	if len(layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(layers))
	}
}

func TestSave_Load(t *testing.T) {
	dir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Project.Name = "test-project"
	cfg.Project.Layers = []LayerConfig{
		{Name: "frontend", Stack: "react", Path: "./frontend"},
	}

	if err := Save(dir, &cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.Project.Name != "test-project" {
		t.Errorf("expected test-project, got %s", loaded.Project.Name)
	}
	if len(loaded.Project.Layers) != 1 {
		t.Errorf("expected 1 layer, got %d", len(loaded.Project.Layers))
	}
}

func TestDefaultConfig_BranchKeepBranchContext(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Branch.KeepBranchContext != false {
		t.Error("expected Branch.KeepBranchContext false by default")
	}
}

func TestBranchConfig_YAMLRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Branch.KeepBranchContext = true

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !loaded.Branch.KeepBranchContext {
		t.Error("expected Branch.KeepBranchContext true after round-trip")
	}
}

func TestBranchConfig_OmitEmpty(t *testing.T) {
	cfg := DefaultConfig()
	// KeepBranchContext is false — field is omitempty, so it should not appear in YAML.
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "keep_branch_context") {
		t.Error("expected keep_branch_context to be omitted when false")
	}
}

func TestFindRoot(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "src", "components")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No qode.yaml → error.
	_, err := FindRoot(subDir)
	if err == nil {
		t.Error("expected error when no qode.yaml exists")
	}

	// Write qode.yaml at root.
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindRoot(subDir)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	if found != dir {
		t.Errorf("expected %s, got %s", dir, found)
	}
}
