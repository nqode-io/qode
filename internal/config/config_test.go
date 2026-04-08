package config

import (
	"errors"
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
	wantRubrics := DefaultRubricConfigs()
	if len(cfg.Scoring.Rubrics) != len(wantRubrics) {
		t.Errorf("expected %d default rubrics, got %d", len(wantRubrics), len(cfg.Scoring.Rubrics))
	}
	reviewRubric, ok := cfg.Scoring.Rubrics["review"]
	if !ok {
		t.Fatal("expected review rubric in defaults")
	}
	wantReviewDims := len(wantRubrics["review"].Dimensions)
	if len(reviewRubric.Dimensions) != wantReviewDims {
		t.Errorf("expected %d review dimensions, got %d", wantReviewDims, len(reviewRubric.Dimensions))
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
	if loaded.QodeVersion != "" {
		t.Errorf("expected empty QodeVersion from DefaultConfig round-trip, got %q", loaded.QodeVersion)
	}
	if loaded.Review.MinCodeScore != cfg.Review.MinCodeScore {
		t.Errorf("MinCodeScore: got %.1f, want %.1f", loaded.Review.MinCodeScore, cfg.Review.MinCodeScore)
	}
	if loaded.Review.MinSecurityScore != cfg.Review.MinSecurityScore {
		t.Errorf("MinSecurityScore: got %.1f, want %.1f", loaded.Review.MinSecurityScore, cfg.Review.MinSecurityScore)
	}
	if loaded.Scoring.Strict != cfg.Scoring.Strict {
		t.Errorf("Strict: got %v, want %v", loaded.Scoring.Strict, cfg.Scoring.Strict)
	}
	if loaded.Scoring.TargetScore != cfg.Scoring.TargetScore {
		t.Errorf("TargetScore: got %d, want %d", loaded.Scoring.TargetScore, cfg.Scoring.TargetScore)
	}
	if loaded.IDE.Cursor.Enabled != cfg.IDE.Cursor.Enabled {
		t.Errorf("Cursor.Enabled: got %v, want %v", loaded.IDE.Cursor.Enabled, cfg.IDE.Cursor.Enabled)
	}
	if loaded.IDE.ClaudeCode.Enabled != cfg.IDE.ClaudeCode.Enabled {
		t.Errorf("ClaudeCode.Enabled: got %v, want %v", loaded.IDE.ClaudeCode.Enabled, cfg.IDE.ClaudeCode.Enabled)
	}
	if len(loaded.Scoring.Rubrics) != len(cfg.Scoring.Rubrics) {
		t.Errorf("Rubrics count: got %d, want %d", len(loaded.Scoring.Rubrics), len(cfg.Scoring.Rubrics))
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
	if loaded.Review.MinCodeScore != cfg.Review.MinCodeScore {
		t.Errorf("MinCodeScore: got %.1f, want %.1f", loaded.Review.MinCodeScore, cfg.Review.MinCodeScore)
	}
	if len(loaded.Scoring.Rubrics) != len(cfg.Scoring.Rubrics) {
		t.Errorf("Rubrics count: got %d, want %d", len(loaded.Scoring.Rubrics), len(cfg.Scoring.Rubrics))
	}
	if loaded.IDE.Cursor.Enabled != cfg.IDE.Cursor.Enabled {
		t.Errorf("Cursor.Enabled: got %v, want %v", loaded.IDE.Cursor.Enabled, cfg.IDE.Cursor.Enabled)
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

	// No qode.yaml → ErrConfigNotFound.
	_, err := FindRoot(subDir)
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
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

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("invalid: [unterminated"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "loading") {
		t.Errorf("expected error to mention 'loading', got: %v", err)
	}
}

func TestLoad_InvalidScoringYAML(t *testing.T) {
	dir := t.TempDir()
	// Valid qode.yaml.
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("qode_version: \"1\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Broken scoring.yaml.
	scoringDir := filepath.Join(dir, QodeDir)
	if err := os.MkdirAll(scoringDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scoringDir, ScoringFileName), []byte("rubrics: [bad"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid scoring YAML")
	}
	if !strings.Contains(err.Error(), "scoring") {
		t.Errorf("expected error to mention 'scoring', got: %v", err)
	}
}
