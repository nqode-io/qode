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

	if cfg.Review.MinCodeScore != 10.0 {
		t.Errorf("expected MinCodeScore 10.0, got %.1f", cfg.Review.MinCodeScore)
	}
	if cfg.Scoring.TargetScore != 0 {
		t.Errorf("expected TargetScore 0 (use rubric max), got %d", cfg.Scoring.TargetScore)
	}
	if !cfg.IDE.Cursor.Enabled {
		t.Error("expected Cursor enabled by default")
	}
	if !cfg.IDE.ClaudeCode.Enabled {
		t.Error("expected ClaudeCode enabled by default")
	}
	if len(cfg.Scoring.Rubrics) != 3 {
		t.Errorf("expected 3 default rubrics, got %d", len(cfg.Scoring.Rubrics))
	}
	reviewRubric, ok := cfg.Scoring.Rubrics["review"]
	if !ok {
		t.Fatal("expected review rubric in defaults")
	}
	if len(reviewRubric.Dimensions) != 6 {
		t.Errorf("expected 6 review dimensions (including Performance), got %d", len(reviewRubric.Dimensions))
	}
}

func TestSave_Load(t *testing.T) {
	dir := t.TempDir()

	cfg := DefaultConfig()

	if err := Save(dir, &cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// QodeVersion is set by qode init (cli layer), not by DefaultConfig.
	// A round-tripped DefaultConfig has no version set.
	if loaded.QodeVersion != "" {
		t.Errorf("expected empty QodeVersion from DefaultConfig round-trip, got %q", loaded.QodeVersion)
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
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("qode_version: \"0.1\"\n"), 0644); err != nil {
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
