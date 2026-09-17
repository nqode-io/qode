//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/prompt"
	"github.com/nqode/qode/internal/scaffold"
	"gopkg.in/yaml.v3"
)

func TestRunInitExisting_WritesQodeVersion(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "qode.yaml"))
	if err != nil {
		t.Fatalf("reading qode.yaml: %v", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshaling qode.yaml: %v", err)
	}

	if raw["qode_version"] != "dev" {
		t.Errorf("expected qode_version \"dev\", got %v", raw["qode_version"])
	}
	if _, ok := raw["project"]; ok {
		t.Error("qode.yaml must not contain a 'project' key")
	}
}

func TestRunInitExisting_CreatesDirs(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	for _, sub := range []string{"contexts", "knowledge", "prompts"} {
		path := filepath.Join(dir, ".qode", sub)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf(".qode/%s/ not created", sub)
		}
	}
}

func TestRunInitExisting_CopiesTemplates(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	promptsDir := filepath.Join(dir, ".qode", "prompts")
	var total int
	if err := filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md.tmpl") {
			total++
		}
		return nil
	}); err != nil {
		t.Fatalf("walking .qode/prompts/: %v", err)
	}
	embedded, _ := prompt.EmbeddedTemplates()
	// Only non-scaffold templates are copied to .qode/prompts/.
	wantTemplates := 0
	for name := range embedded {
		if !strings.HasPrefix(name, "scaffold/") {
			wantTemplates++
		}
	}
	if total != wantTemplates {
		t.Errorf("expected %d .md.tmpl files under .qode/prompts/, got %d", wantTemplates, total)
	}
}

func TestRunInitExisting_CreatesAgentConfigs(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	claudePath := filepath.Join(dir, ".claude", "commands", "qode-plan-refine.md")
	if _, err := os.Stat(claudePath); os.IsNotExist(err) {
		t.Error(".claude/commands/qode-plan-refine.md not created")
	}

	cursorPath := filepath.Join(dir, ".cursor", "commands", "qode-plan-refine.mdc")
	if _, err := os.Stat(cursorPath); os.IsNotExist(err) {
		t.Error(".cursor/commands/qode-plan-refine.mdc not created")
	}

	codexPath := filepath.Join(dir, ".agents", "skills", "qode-plan-refine", "SKILL.md")
	if _, err := os.Stat(codexPath); os.IsNotExist(err) {
		t.Error(".agents/skills/qode-plan-refine/SKILL.md not created")
	}

	for _, path := range []string{
		filepath.Join(dir, ".claude", "commands", "qode-note-add.md"),
		filepath.Join(dir, ".cursor", "commands", "qode-note-add.mdc"),
		filepath.Join(dir, ".agents", "skills", "qode-note-add", "SKILL.md"),
		filepath.Join(dir, ".opencode", "commands", "qode-plan-refine.md"),
		filepath.Join(dir, ".opencode", "commands", "qode-note-add.md"),
	} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s not created", path)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".codex", "commands")); !os.IsNotExist(err) {
		t.Error("legacy .codex/commands directory should not be created")
	}
}

func TestRunInitExisting_NoCursorRules(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	// Positive: .qode/prompts/ must exist.
	if _, err := os.Stat(filepath.Join(dir, ".qode", "prompts")); err != nil {
		t.Errorf("expected .qode/prompts/ to exist: %v", err)
	}
	// Negative: legacy .cursorrules/ must not be created.
	if _, err := os.Stat(filepath.Join(dir, ".cursorrules")); !os.IsNotExist(err) {
		t.Error("runInitExisting must not create .cursorrules/ directory")
	}
}

func TestRunInitExisting_NoDetectionOutput(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	var buf bytes.Buffer
	if err := runInitExisting(context.Background(), &buf, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	// The writer must not contain legacy detection phrases.
	out := buf.String()
	for _, forbidden := range []string{"Detected", "Scanning", "qode ide setup"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("output must not contain %q, got: %s", forbidden, out)
		}
	}

	// Sanity-check that the writer received expected content.
	if !strings.Contains(out, "Generated:") {
		t.Errorf("expected 'Generated:' in output, got: %s", out)
	}
}

func TestRunInitExisting_CreatesScoringYaml(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	if _, err := os.Stat(scoringPath); os.IsNotExist(err) {
		t.Error(".qode/scoring.yaml not created on first run")
	}
}

func TestRunInitExisting_RerunPreservesScoringYaml(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}

	// Record scoring.yaml content after first run.
	scoringPath := filepath.Join(dir, ".qode", "scoring.yaml")
	firstScoring, err := os.ReadFile(scoringPath)
	if err != nil {
		t.Fatalf("reading scoring.yaml after first run: %v", err)
	}

	// Second run must succeed and must not overwrite .qode/scoring.yaml.
	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("second runInitExisting: %v", err)
	}

	secondScoring, err := os.ReadFile(scoringPath)
	if err != nil {
		t.Fatalf("reading scoring.yaml after second run: %v", err)
	}

	if string(firstScoring) != string(secondScoring) {
		t.Error(".qode/scoring.yaml was overwritten on re-run")
	}
}

func TestRunInitExisting_AppendsGitignoreRules(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)

	for _, rule := range scaffold.GitignoreRules {
		if !strings.Contains(content, rule) {
			t.Errorf(".gitignore missing rule %q", rule)
		}
	}
}

func TestRunInitExisting_GitignoreIsIdempotent(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}
	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("second runInitExisting: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("reading .gitignore: %v", err)
	}
	content := string(data)

	for _, rule := range scaffold.GitignoreRules {
		if count := strings.Count(content, rule); count != 1 {
			t.Errorf("rule %q appears %d times, want 1", rule, count)
		}
	}
}

func TestRootCmd_NoAgentsSubcommand(t *testing.T) {
	// Confirm 'agents' is not a registered subcommand.
	agentsCmd, _, findErr := rootCmd.Find([]string{"agents"})
	if findErr == nil && agentsCmd != rootCmd {
		t.Error("'agents' subcommand must not be registered on rootCmd")
	}
}

// seedProjectConfig writes body as the project qode.yaml and returns its path.
func seedProjectConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("seeding %s: %v", config.ConfigFileName, err)
	}
	return path
}

func readProjectConfig(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, config.ConfigFileName))
	if err != nil {
		t.Fatalf("reading %s: %v", config.ConfigFileName, err)
	}
	return string(data)
}

func TestRunInitExisting_ConfigOnly_WritesOnlyConfig(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", true, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != config.ConfigFileName {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("--config-only created %v, want only %s", names, config.ConfigFileName)
	}
	if !strings.Contains(readProjectConfig(t, dir), "# Minimum scores a review must reach.") {
		t.Error("generated config is not the commented default document")
	}
}

func TestRunInitExisting_ConfigOnly_RefusesExisting(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	const existing = "qode_version: 0.1.0\n"
	seedProjectConfig(t, dir, existing)

	err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", true, false)
	if !errors.Is(err, config.ErrConfigExists) {
		t.Fatalf("error = %v, want ErrConfigExists", err)
	}
	if got := readProjectConfig(t, dir); got != existing {
		t.Errorf("existing config was modified: %q", got)
	}
}

func TestRunInitExisting_ConfigOnly_ForceOverwrites(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, "qode_version: 0.1.0\n")

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", true, true); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}
	if !strings.Contains(readProjectConfig(t, dir), "# Minimum scores a review must reach.") {
		t.Error("config was not replaced by the commented defaults")
	}
}

// customisedConfig is a hand-tuned project config: three non-default values, a
// hand-written comment, and no agents.opencode block.
const customisedConfig = `# keep me
qode_version: 0.1.0
review:
  min_security_score: 10
scoring:
  strict: true
agents:
  cursor:
    enabled: false
  claude_code:
    enabled: true
  codex:
    enabled: true
`

func TestRunInitExisting_UpgradesExistingConfig(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, customisedConfig)

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	rendered := readProjectConfig(t, dir)
	if !strings.Contains(rendered, "# keep me") {
		t.Errorf("hand-written comment was dropped:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  opencode:\n    enabled: true") {
		t.Errorf("opencode was not appended under the existing agents mapping:\n%s", rendered)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if cfg.Review.MinSecurityScore != 10 {
		t.Errorf("min_security_score = %v, want the user's 10", cfg.Review.MinSecurityScore)
	}
	if !cfg.Scoring.Strict {
		t.Error("scoring.strict was reset to false")
	}
	if cfg.Agents.Cursor.Enabled {
		t.Error("agents.cursor.enabled was reset to true")
	}
	if cfg.QodeVersion != "0.4.0-beta" {
		t.Errorf("qode_version = %q, want it re-stamped to 0.4.0-beta", cfg.QodeVersion)
	}
}

func TestRunInitExisting_SecondRunIsNoOp(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, customisedConfig)

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("first runInitExisting: %v", err)
	}
	before := readProjectConfig(t, dir)
	path := filepath.Join(dir, config.ConfigFileName)
	statBefore, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("second runInitExisting: %v", err)
	}
	if got := readProjectConfig(t, dir); got != before {
		t.Errorf("second run rewrote qode.yaml:\n%s", got)
	}
	statAfter, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !statAfter.ModTime().Equal(statBefore.ModTime()) {
		t.Error("second run touched qode.yaml's mtime")
	}
}

func TestRunInitExisting_SkipsDisabledAgents(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		skipped  string
		expected []string
	}{
		{
			name: "cursor", key: "cursor", skipped: ".cursor",
			expected: []string{".claude/commands/qode-plan-refine.md", ".agents/skills/qode-plan-refine/SKILL.md", ".opencode/commands/qode-plan-refine.md"},
		},
		{
			name: "claude_code", key: "claude_code", skipped: ".claude",
			expected: []string{".cursor/commands/qode-plan-refine.mdc", ".agents/skills/qode-plan-refine/SKILL.md", ".opencode/commands/qode-plan-refine.md"},
		},
		{
			name: "codex", key: "codex", skipped: ".agents",
			expected: []string{".cursor/commands/qode-plan-refine.mdc", ".claude/commands/qode-plan-refine.md", ".opencode/commands/qode-plan-refine.md"},
		},
		{
			name: "opencode", key: "opencode", skipped: ".opencode",
			expected: []string{".cursor/commands/qode-plan-refine.mdc", ".claude/commands/qode-plan-refine.md", ".agents/skills/qode-plan-refine/SKILL.md"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv forbids t.Parallel; config.Load reads os.UserHomeDir().
			isolateHome(t)
			dir := t.TempDir()
			seedProjectConfig(t, dir, "agents:\n  "+tc.key+":\n    enabled: false\n")

			if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
				t.Fatalf("runInitExisting: %v", err)
			}

			if _, err := os.Stat(filepath.Join(dir, tc.skipped)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s was generated for a disabled agent", tc.skipped)
			}
			for _, rel := range tc.expected {
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
					t.Errorf("%s was not generated: %v", rel, err)
				}
			}
		})
	}
}

func TestRunInitExisting_NoAgentsEnabled(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, `agents:
  cursor:
    enabled: false
  claude_code:
    enabled: false
  codex:
    enabled: false
  opencode:
    enabled: false
`)

	var buf bytes.Buffer
	if err := runInitExisting(context.Background(), &buf, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}
	if !strings.Contains(buf.String(), "No IDEs enabled") {
		t.Errorf("output does not guide a user with every agent disabled:\n%s", buf.String())
	}
}

func TestRunInitExisting_BrokenConfigIsUntouched(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "parse error", body: "scoring: [\n"},
		{name: "validate failure", body: "review:\n  min_code_score: -1\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv forbids t.Parallel; config.Load reads os.UserHomeDir().
			isolateHome(t)
			dir := t.TempDir()
			seedProjectConfig(t, dir, tc.body)

			err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false)
			if !errors.Is(err, config.ErrConfigInvalid) {
				t.Fatalf("error = %v, want ErrConfigInvalid", err)
			}
			if !strings.Contains(err.Error(), "--config-only --force") {
				t.Errorf("error does not name the overwrite command: %v", err)
			}
			if got := readProjectConfig(t, dir); got != tc.body {
				t.Errorf("broken config was modified:\ngot  %q\nwant %q", got, tc.body)
			}
		})
	}
}

func TestRunInitExisting_BrokenScoringYamlHasNoOverwriteHint(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantNamed string
	}{
		// A parse failure is wrapped by Load with the offending path.
		{name: "parse error", body: "rubrics: [\n", wantNamed: config.ScoringFileName},
		// A file that parses but fails Validate is reported by the rubric key it broke.
		{name: "invalid rubric", body: "rubrics:\n  bogus:\n    dimensions: []\n", wantNamed: "scoring.rubrics"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv forbids t.Parallel; config.Load reads os.UserHomeDir().
			isolateHome(t)
			dir := t.TempDir()
			const valid = "qode_version: 0.1.0\n"
			seedProjectConfig(t, dir, valid)

			scoringPath := filepath.Join(dir, config.QodeDir, config.ScoringFileName)
			if err := os.MkdirAll(filepath.Dir(scoringPath), 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(scoringPath, []byte(tc.body), 0644); err != nil {
				t.Fatalf("writing %s: %v", config.ScoringFileName, err)
			}

			err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false)
			if err == nil {
				t.Fatal("expected an error for a broken scoring.yaml")
			}
			if !strings.Contains(err.Error(), tc.wantNamed) {
				t.Errorf("error does not name %q: %v", tc.wantNamed, err)
			}
			if strings.Contains(err.Error(), "--config-only --force") {
				t.Errorf("a broken scoring.yaml must not suggest overwriting qode.yaml: %v", err)
			}
			// qode.yaml is valid here, so ensureConfig upgrades it before Load reaches
			// the broken rubric file; what must survive is the user's own value.
			if got := readProjectConfig(t, dir); !strings.Contains(got, "qode_version: 0.1.0") {
				t.Errorf("the user's qode_version did not survive: %q", got)
			}
		})
	}
}

func TestRunInitExisting_EmptyConfigIsFilled(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, "")

	if err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	if !strings.Contains(readProjectConfig(t, dir), "# Minimum scores a review must reach.") {
		t.Error("empty config was not filled with the commented defaults")
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if cfg.Review.MinCodeScore != config.DefaultConfig().Review.MinCodeScore {
		t.Errorf("min_code_score = %v, want the default", cfg.Review.MinCodeScore)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "commands", "qode-plan-refine.md")); err != nil {
		t.Errorf("scaffolding did not proceed after filling the config: %v", err)
	}
}

func TestRunInitExisting_UpgradeOutputLine(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, "qode_version: 0.1.0\n")

	var buf bytes.Buffer
	if err := runInitExisting(context.Background(), &buf, io.Discard, dir, "", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Updated: ") {
		t.Errorf("upgrade path did not report the update:\n%s", out)
	}
	for _, banned := range []string{"Detected", "Scanning", "qode ide setup"} {
		if strings.Contains(out, banned) {
			t.Errorf("output reintroduced %q:\n%s", banned, out)
		}
	}
}

func TestRunInitExisting_UnreadableConfigHasNoOverwriteHint(t *testing.T) {
	isolateHome(t)
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := t.TempDir()
	// An unreadable qode.yaml makes Upgrade fail on the read, which is not
	// ErrConfigInvalid, so the overwrite hint must not be attached.
	path := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(path, []byte("qode_version: 0.1.0\n"), 0000); err != nil {
		t.Fatalf("seeding: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	err := runInitExisting(context.Background(), &bytes.Buffer{}, io.Discard, dir, "", false, false)
	if err == nil {
		t.Fatal("expected an error when qode.yaml cannot be read")
	}
	if errors.Is(err, config.ErrConfigInvalid) {
		t.Errorf("an unreadable file is not an invalid one: %v", err)
	}
	if strings.Contains(err.Error(), "--config-only --force") {
		t.Errorf("hint attached to an error that overwriting will not fix: %v", err)
	}
}

// --- deprecated ide: key ---

const legacyRenameLine = "qode.yaml: 'ide:' has been renamed to 'agents:' — updated in place."

func TestRunInitExisting_RenamesLegacyIDEKey(t *testing.T) {
	// t.Setenv forbids t.Parallel; config.Load reads os.UserHomeDir().
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir, "qode_version: 0.3.4-beta\nide:\n  cursor:\n    enabled: false\n")

	var buf bytes.Buffer
	if err := runInitExisting(context.Background(), &buf, io.Discard, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	if got := strings.Count(buf.String(), legacyRenameLine); got != 1 {
		t.Errorf("rename line printed %d times, want 1:\n%s", got, buf.String())
	}
	rendered := readProjectConfig(t, dir)
	if !strings.Contains(rendered, "\nagents:\n") {
		t.Errorf("qode.yaml has no agents: block:\n%s", rendered)
	}
	if strings.Contains(rendered, "\nide:\n") {
		t.Errorf("qode.yaml still carries the deprecated ide: block:\n%s", rendered)
	}
	if _, err := os.Stat(filepath.Join(dir, ".cursor")); !errors.Is(err, fs.ErrNotExist) {
		t.Error(".cursor was generated for an agent the legacy config disabled")
	}
}

func TestRunInitExisting_BothKeys_DoesNotPrintRenameLine(t *testing.T) {
	// t.Setenv forbids t.Parallel; config.Load reads os.UserHomeDir().
	isolateHome(t)
	dir := t.TempDir()
	const body = "qode_version: 0.3.4-beta\nagents:\n  cursor:\n    enabled: true\nide:\n  cursor:\n    enabled: false\n"
	seedProjectConfig(t, dir, body)

	var buf bytes.Buffer
	if err := runInitExisting(context.Background(), &buf, io.Discard, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	if strings.Contains(buf.String(), legacyRenameLine) {
		t.Errorf("rename line printed for a file carrying both keys:\n%s", buf.String())
	}
	rendered := readProjectConfig(t, dir)
	if !strings.Contains(rendered, "ide:\n  cursor:\n    enabled: false\n") {
		t.Errorf("the deprecated block was rewritten:\n%s", rendered)
	}
}

func TestInitCmd_LongHelpUsesAgentVocabulary(t *testing.T) {
	t.Parallel()

	long := newInitCmd().Long
	if !strings.Contains(long, "agent workflow assets") {
		t.Errorf("init --help does not describe agent workflow assets:\n%s", long)
	}
	if strings.Contains(long, "IDE") {
		t.Errorf("init --help still says IDE:\n%s", long)
	}
}

func TestRunInitExisting_BothKeysConfig_WarnsOnce(t *testing.T) {
	// t.Setenv (via isolateHome) forbids t.Parallel.
	isolateHome(t)
	dir := t.TempDir()
	seedProjectConfig(t, dir,
		"qode_version: 0.3.4-beta\nagents:\n  cursor:\n    enabled: true\nide:\n  cursor:\n    enabled: false\n")

	var out, errOut bytes.Buffer
	if err := runInitExisting(context.Background(), &out, &errOut, dir, "0.4.0-beta", false, false); err != nil {
		t.Fatalf("runInitExisting: %v", err)
	}

	if got := strings.Count(errOut.String(), "both 'agents:' and 'ide:' are set"); got != 1 {
		t.Errorf("warning printed %d times, want 1:\n%s", got, errOut.String())
	}
	// The two notices are mutually exclusive: a file carrying both keys is never
	// renamed, so the rename line must not appear beside the warning.
	if strings.Contains(out.String(), legacyRenameLine) {
		t.Errorf("rename line printed alongside the both-keys warning:\n%s", out.String())
	}
}
