package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	if !cfg.Agents.Cursor.Enabled {
		t.Error("expected Cursor enabled by default")
	}
	if !cfg.Agents.ClaudeCode.Enabled {
		t.Error("expected ClaudeCode enabled by default")
	}
	if !cfg.Agents.Codex.Enabled {
		t.Error("expected Codex enabled by default")
	}
	if !cfg.Agents.OpenCode.Enabled {
		t.Error("expected OpenCode enabled by default")
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
	// config.Load merges ~/.qode/config.yaml, so HOME must be isolated.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
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
	if loaded.Agents.Cursor.Enabled != cfg.Agents.Cursor.Enabled {
		t.Errorf("Cursor.Enabled: got %v, want %v", loaded.Agents.Cursor.Enabled, cfg.Agents.Cursor.Enabled)
	}
	if loaded.Agents.ClaudeCode.Enabled != cfg.Agents.ClaudeCode.Enabled {
		t.Errorf("ClaudeCode.Enabled: got %v, want %v", loaded.Agents.ClaudeCode.Enabled, cfg.Agents.ClaudeCode.Enabled)
	}
	if len(loaded.Scoring.Rubrics) != len(cfg.Scoring.Rubrics) {
		t.Errorf("Rubrics count: got %d, want %d", len(loaded.Scoring.Rubrics), len(cfg.Scoring.Rubrics))
	}
}

func TestLoad_ConfigWithoutOpenCode_DefaultsEnabled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	legacy := `qode_version: "0.3.0"
agents:
    cursor:
        enabled: true
    claude_code:
        enabled: true
    codex:
        enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.Agents.OpenCode.Enabled {
		t.Error("expected OpenCode enabled when agents.opencode is absent from qode.yaml")
	}
}

func TestDefaultConfig_DiffCommand(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Diff.Command == "" {
		t.Error("expected non-empty default Diff.Command")
	}
}

func TestDiffConfig_YAMLRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Diff.Command = "git diff HEAD"

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.Diff.Command != "git diff HEAD" {
		t.Errorf("Diff.Command: got %q, want %q", loaded.Diff.Command, "git diff HEAD")
	}
	if loaded.Review.MinCodeScore != cfg.Review.MinCodeScore {
		t.Errorf("MinCodeScore: got %.1f, want %.1f", loaded.Review.MinCodeScore, cfg.Review.MinCodeScore)
	}
	if len(loaded.Scoring.Rubrics) != len(cfg.Scoring.Rubrics) {
		t.Errorf("Rubrics count: got %d, want %d", len(loaded.Scoring.Rubrics), len(cfg.Scoring.Rubrics))
	}
	if loaded.Agents.Cursor.Enabled != cfg.Agents.Cursor.Enabled {
		t.Errorf("Cursor.Enabled: got %v, want %v", loaded.Agents.Cursor.Enabled, cfg.Agents.Cursor.Enabled)
	}
}

func TestDiffConfig_OmitEmpty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Diff.Command = ""
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "command:") && !strings.Contains(string(data), "diff:") {
		t.Error("unexpected command key when empty")
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

// --- deprecated ide: key ---

// seedLegacyHome isolates HOME so Load's user-level merge is deterministic and
// returns the isolated home directory.
func seedLegacyHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// writeProjectConfig writes body as the project qode.yaml and returns its path.
func writeProjectConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// writeUserConfig writes body as ~/.qode/config.yaml and returns its path.
func writeUserConfig(t *testing.T, home, body string) string {
	t.Helper()
	path := filepath.Join(home, QodeDir, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// assertAgents compares the four toggles in cursor, claude_code, codex, opencode
// order and fails with the whole vector, which is what makes a dropped setting
// readable rather than a single boolean mismatch.
func assertAgents(t *testing.T, cfg *Config, want [4]bool) {
	t.Helper()
	got := [4]bool{
		cfg.Agents.Cursor.Enabled,
		cfg.Agents.ClaudeCode.Enabled,
		cfg.Agents.Codex.Enabled,
		cfg.Agents.OpenCode.Enabled,
	}
	if got != want {
		t.Errorf("agents = %v, want %v", got, want)
	}
	if cfg.IDE.Kind != 0 {
		t.Errorf("cfg.IDE.Kind = %d, want 0: the legacy node must never survive a load", cfg.IDE.Kind)
	}
}

func TestLoad_LegacyIDEPartialBlock_KeepsUnmentionedAgentDefaults(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "ide:\n  cursor:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{false, true, true, true})
}

func TestLoad_LegacyIDEPartialBlockInUserConfig_KeepsUnmentionedAgentDefaults(t *testing.T) {
	// t.Setenv forbids t.Parallel; the legacy block lives under HOME.
	home := seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "qode_version: 0.4.0-beta\n")
	writeUserConfig(t, home, "ide:\n  codex:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{true, true, false, true})
}

func TestLoad_LegacyIDEThroughMergeKey_KeepsUnmentionedAgentDefaults(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "base: &base\n  ide:\n    cursor:\n      enabled: false\n<<: *base\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{false, true, true, true})
}

func TestLoad_LegacyIDEInProjectAndUserConfig_AppliesBoth(t *testing.T) {
	// t.Setenv forbids t.Parallel; one of the two legacy blocks lives under HOME.
	home := seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "ide:\n  cursor:\n    enabled: false\n")
	writeUserConfig(t, home, "ide:\n  codex:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Promotion must run per file: the user file's ide: node replaces the project
	// file's in cfg.IDE, so a single promotion at the end loses cursor: false.
	assertAgents(t, cfg, [4]bool{false, true, false, true})
}

func TestLoad_LegacyIDENullSection_KeepsAllAgentsEnabled(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "ide:\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{true, true, true, true})
}

func TestLoad_LegacyIDEWithAgents_AgentsWins(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir,
		"agents:\n  cursor:\n    enabled: true\nide:\n  cursor:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{true, true, true, true})
}

func TestLoad_LegacyIDEWithAgentsInUserConfig_AgentsWins(t *testing.T) {
	// t.Setenv forbids t.Parallel; both keys live in ~/.qode/config.yaml.
	home := seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "qode_version: 0.4.0-beta\n")
	writeUserConfig(t, home,
		"agents:\n  codex:\n    enabled: true\nide:\n  codex:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{true, true, true, true})
}

func TestLoad_LegacyIDEWrongType_IsRefused(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	tests := []struct {
		name string
		body string
	}{
		{name: "sequence", body: "qode_version: 0.4.0-beta\nide: [cursor]\n"},
		{name: "scalar", body: "qode_version: 0.4.0-beta\nide: cursor\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seedLegacyHome(t)
			dir := t.TempDir()
			path := writeProjectConfig(t, dir, tc.body)

			_, err := Load(dir)
			if err == nil {
				t.Fatalf("Load accepted an ide: key that is not a mapping")
			}
			if !strings.Contains(err.Error(), "cannot unmarshal") {
				t.Errorf("error = %v, want it to say what could not be unmarshalled", err)
			}
			// The path belongs in the message once. Load adds it; the probe
			// unmarshal must not add it a second time.
			if got := strings.Count(err.Error(), path); got != 1 {
				t.Errorf("the config path appears %d times in %v, want 1", got, err)
			}

			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("reading %s: %v", path, readErr)
			}
			if string(after) != tc.body {
				t.Errorf("a refused load rewrote the file:\ngot\n%s\nwant\n%s", after, tc.body)
			}
		})
	}
}

func TestLoad_LegacyKeyNotices(t *testing.T) {
	// t.Setenv forbids t.Parallel; every case isolates HOME.
	tests := []struct {
		name    string
		project string
		user    string
		want    func(projectPath, userPath string) []string
	}{
		{
			name:    "legacy key alone",
			project: "ide:\n  cursor:\n    enabled: false\n",
			want: func(p, _ string) []string {
				return []string{fmt.Sprintf(legacyKeyNotice, p)}
			},
		},
		{
			name:    "canonical key alone",
			project: "agents:\n  cursor:\n    enabled: false\n",
			want:    func(string, string) []string { return nil },
		},
		{
			name:    "both keys",
			project: "agents:\n  cursor:\n    enabled: true\nide:\n  cursor:\n    enabled: false\n",
			want: func(p, _ string) []string {
				return []string{fmt.Sprintf(bothKeysNotice, p)}
			},
		},
		{
			// The ordinary "rename the key" advice would produce a duplicate
			// agents: here, which the next qode init refuses, so this shape gets
			// the two-step wording instead.
			name:    "null agents key beside a legacy block",
			project: "agents:\nide:\n  cursor:\n    enabled: false\n",
			want: func(p, _ string) []string {
				return []string{fmt.Sprintf(emptyAgentsNotice, p)}
			},
		},
		{
			name:    "both keys in the user config",
			project: "qode_version: 0.4.0-beta\n",
			user:    "agents:\n  codex:\n    enabled: true\nide:\n  codex:\n    enabled: false\n",
			want: func(_, u string) []string {
				return []string{fmt.Sprintf(bothKeysNotice, u)}
			},
		},
		{
			name:    "legacy key in the user config",
			project: "qode_version: 0.4.0-beta\n",
			user:    "ide:\n  codex:\n    enabled: false\n",
			want: func(_, u string) []string {
				return []string{fmt.Sprintf(legacyKeyNotice, u)}
			},
		},
		{
			name:    "neither key",
			project: "qode_version: 0.4.0-beta\n",
			want:    func(string, string) []string { return nil },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := seedLegacyHome(t)
			dir := t.TempDir()
			projectPath := writeProjectConfig(t, dir, tc.project)
			userPath := filepath.Join(home, QodeDir, "config.yaml")
			if tc.user != "" {
				userPath = writeUserConfig(t, home, tc.user)
			}

			cfg, err := Load(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			got := cfg.Notices()
			want := tc.want(projectPath, userPath)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Notices() =\n%#v\nwant\n%#v", got, want)
			}
		})
	}
}

func TestUpgrade_AcceptsWhatTheEmptyAgentsNoticeTellsTheUserToDo(t *testing.T) {
	t.Parallel()

	// A valueless agents: beside an ide: block is the one shape where the
	// ordinary deprecation advice — "rename the key to 'agents:'" — leaves the
	// user worse off than before. emptyAgentsNotice sends them down the second
	// row instead; this is what makes the two rows different.
	tests := []struct {
		name    string
		edited  string
		wantErr error
	}{
		{
			name:    "renaming the key, as the ordinary notice says",
			edited:  "qode_version: 0.4.0-beta\nagents:\nagents:\n  cursor:\n    enabled: false\n",
			wantErr: ErrConfigInvalid,
		},
		{
			name:   "deleting the empty line first, as the empty-agents notice says",
			edited: "qode_version: 0.4.0-beta\nagents:\n  cursor:\n    enabled: false\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := seedConfig(t, tc.edited)

			_, err := Upgrade(context.Background(), dir, "0.4.0-beta")
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Upgrade: %v", err)
			}
			var got Config
			if err := yaml.Unmarshal(readConfig(t, dir), &got); err != nil {
				t.Fatalf("unmarshalling upgraded config: %v", err)
			}
			if got.Agents.Cursor.Enabled {
				t.Error("the remedy the notice recommends re-enabled cursor")
			}
		})
	}
}

func TestLoad_LegacyKeyNotices_BothFilesOrderedProjectFirst(t *testing.T) {
	// t.Setenv forbids t.Parallel; one of the two legacy blocks lives under HOME.
	home := seedLegacyHome(t)
	dir := t.TempDir()
	projectPath := writeProjectConfig(t, dir, "ide:\n  cursor:\n    enabled: false\n")
	userPath := writeUserConfig(t, home, "ide:\n  codex:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := []string{
		fmt.Sprintf(legacyKeyNotice, projectPath),
		fmt.Sprintf(legacyKeyNotice, userPath),
	}
	if !reflect.DeepEqual(cfg.Notices(), want) {
		t.Errorf("Notices() =\n%#v\nwant\n%#v", cfg.Notices(), want)
	}
}

func TestSave_NeverEmitsLegacyKeys(t *testing.T) {
	// t.Setenv forbids t.Parallel; Load merges ~/.qode/config.yaml.
	seedLegacyHome(t)
	dir := t.TempDir()
	writeProjectConfig(t, dir, "ide:\n  cursor:\n    enabled: false\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	assertAgents(t, cfg, [4]bool{false, true, true, true})

	out := t.TempDir()
	if err := Save(out, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(out, ConfigFileName))
	if err != nil {
		t.Fatalf("reading saved config: %v", err)
	}
	for _, banned := range []string{"ide:", "legacy:"} {
		if strings.Contains(string(data), banned) {
			t.Errorf("saved config contains %q:\n%s", banned, data)
		}
	}
	if !strings.Contains(string(data), "agents:") {
		t.Errorf("saved config has no agents: block:\n%s", data)
	}
}
