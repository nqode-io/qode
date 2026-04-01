# Code Review — reimplement-init-command

## Pre-read Incident Report

This code ships. Two weeks later, a CI pipeline initialises a project with a locked-down `.qode/` directory (read-only parent, group-owned). `qode init` succeeds — but silently skips writing `.qode/scoring.yaml` because the `os.Stat` call returns a permission error, not a NotExist error. The guard only enters the write block when the error is NotExist. The pipeline runs with no custom rubrics; all AI reviews score against built-in defaults. Nobody notices until someone wonders why rubric thresholds look wrong. The file was never written and no error was surfaced.

---

## Files Reviewed

- `internal/cli/init.go`
- `internal/cli/init_test.go`
- `internal/config/config.go` (mergeScoringFromFile, Load)
- `internal/config/config_test.go`
- `internal/config/schema.go`
- `internal/config/defaults.go`
- `internal/scaffold/claudecode.go`
- `internal/scaffold/cursor.go`
- `internal/scaffold/scaffold.go`
- `internal/scaffold/scaffold_test.go`
- `internal/cli/knowledge_cmd.go`
- `internal/cli/plan_test.go`
- `internal/cli/root.go`
- `docs/scoring-yaml-reference.md`
- `docs/qode-yaml-reference.md`
- `docs/how-to-customise-prompts.md`
- `README.md`

---

## Issues

---

### M1 — Silent swallowing of non-NotExist stat errors

**Severity:** Medium  
**File:** `internal/cli/init.go:71`  
**Issue:** The guard that decides whether to write `.qode/scoring.yaml` passes only when `os.IsNotExist(statErr)` is true. If `os.Stat` returns any other error (e.g. `EACCES`, `ENOTDIR`), the condition is false, the write block is skipped, and no error is returned. `scoring.yaml` is silently absent. The user sees a clean exit but gets no rubrics.

**Suggestion:**

```go
_, statErr := os.Stat(scoringPath)
if statErr != nil {
    if !os.IsNotExist(statErr) {
        return fmt.Errorf("checking %s: %w", scoringPath, statErr)
    }
    // File does not exist — first run. Write defaults.
    scoringFile := config.ScoringFileConfig{Rubrics: config.DefaultRubricConfigs()}
    scoringData, err := yaml.Marshal(&scoringFile)
    if err != nil {
        return fmt.Errorf("marshaling scoring config: %w", err)
    }
    if err := os.WriteFile(scoringPath, scoringData, 0644); err != nil {
        return fmt.Errorf("writing %s: %w", scoringPath, err)
    }
    fmt.Printf("Generated: %s\n", scoringPath)
}
```

---

### M2 — New `mergeScoringFromFile` logic has no unit test

**Severity:** Medium  
**File:** `internal/config/config.go:54–65` (mergeScoringFromFile), `internal/config/config_test.go`  
**Issue:** `mergeScoringFromFile` is new logic that merges `.qode/scoring.yaml` rubrics into the loaded config. It has no direct unit test. The only coverage is indirect — `init_test.go` verifies that `.qode/scoring.yaml` is written and not overwritten, but nothing tests that `config.Load` actually reads rubrics from an existing `scoring.yaml` and makes them available on `cfg.Scoring.Rubrics`. If this merge path were silently broken (e.g. wrong YAML key, wrong merge condition), all downstream scoring behaviour would regress without a failing test.

**Suggestion:** Add to `internal/config/config_test.go`:

```go
func TestLoad_MergesRubricsFromScoringFile(t *testing.T) {
    dir := t.TempDir()
    if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("qode_version: \"0.1\"\n"), 0644); err != nil {
        t.Fatal(err)
    }
    scoring := "rubrics:\n  refine:\n    dimensions:\n      - name: Custom\n        weight: 99\n"
    qodeDir := filepath.Join(dir, QodeDir)
    if err := os.MkdirAll(qodeDir, 0755); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(qodeDir, ScoringFileName), []byte(scoring), 0644); err != nil {
        t.Fatal(err)
    }
    cfg, err := Load(dir)
    if err != nil {
        t.Fatalf("Load: %v", err)
    }
    refine, ok := cfg.Scoring.Rubrics["refine"]
    if !ok || len(refine.Dimensions) == 0 || refine.Dimensions[0].Name != "Custom" {
        t.Errorf("expected rubric from scoring.yaml to be merged, got %+v", cfg.Scoring.Rubrics)
    }
}
```

Also add a test for the malformed-file error path.

---

### L1 — Dead `cfg *config.Config` parameter in public scaffold API

**Severity:** Low  
**File:** `internal/scaffold/claudecode.go:12`, `internal/scaffold/cursor.go:14`  
**Issue:** `SetupClaudeCode(root string, cfg *config.Config)` and `SetupCursor(root string, cfg *config.Config)` never read `cfg`. The parameter was preserved per the spec's interface-stability requirement, but every caller passes a real config it believes is being used. A future engineer adding IDE-specific behaviour (e.g. toggling commands based on `cfg.IDE.Cursor.Enabled`) would likely add it inside these functions and be surprised to find the parameter already present but ignored.

**Note:** The spec explicitly preserved this signature for external-caller stability. Flag as a debt item; no immediate fix required to merge.

---

### L2 — Dead `tmplCount` variable in test

**Severity:** Low  
**File:** `internal/cli/init_test.go:883`  
**Issue:** `tmplCount` is computed in a first loop, then immediately discarded with `_ = tmplCount`. Only `total` (from `filepath.Walk`) is checked. Looks like an incomplete refactor.

**Suggestion:** Remove the first loop and `tmplCount` entirely.

---

### Nit1 — `cfgForYaml.Scoring.Rubrics = nil` protection is correct but its purpose is implicit

**Severity:** Nit  
**File:** `internal/cli/init.go:736`  
**Issue:** After the nil assignment, `cfg` (with rubrics intact) is passed to `scaffold.Setup`, which never reads rubrics. The protection is correct but its purpose is not obvious — a reader might wonder what downstream call requires rubrics on `cfg`.

**Suggestion:** Add a brief comment: `// Nil rubrics here because .qode/scoring.yaml owns them; scaffold.Setup does not read them.`

---

## Positive Findings (verified, not assumed)

**`cfgForYaml` value-copy semantics:** `cfgForYaml := cfg` copies the struct. `cfgForYaml.Scoring.Rubrics = nil` assigns nil to the copy's field only — does not mutate `cfg.Scoring.Rubrics`. Go map reference is replaced, not the underlying map. Verified safe.

**`rootCmd.Version` fallback:** In tests `rootCmd.Version` is `""` (no ldflags), so `"dev"` is always written. `TestRunInitExisting_WritesQodeVersion` asserts exactly `"dev"`. Verified consistent.

**`os.IsNotExist` guard for scoring.yaml on re-run (happy path):** `TestRunInitExisting_RerunPreservesScoringYaml` reads file bytes after both runs and asserts equality. Guard is correct for the file-exists case.

**`ide` subcommand removal:** `newIDECmd()` removed from `root.go` AddCommand list. `TestRootCmd_NoIDESubcommand` calls `rootCmd.Find([]string{"ide"})` and asserts it returns `rootCmd` itself — Cobra's signal for not-found. Verified clean.

**Package rename `internal/ide` → `internal/scaffold`:** All import paths updated. No residual references to `internal/ide`. `scaffold_test.go` is in `package scaffold`. `init.go` imports `github.com/nqode/qode/internal/scaffold`. Verified complete.

**`StackDefaults` deletion:** Entire map removed from `defaults.go`. All detector `DefaultConfig()` methods that referenced it are gone with the deleted `internal/detect/` package. No remaining references. Verified complete.

**`slashCommands`/`claudeSlashCommands` project name derivation:** Both internal functions take `name string` derived from `filepath.Base(root)` in the public Setup functions. `TestSetupClaudeCode_CommandsContainRootName` and `TestSetupCursor_CommandsContainRootName` confirm the project name appears in command content when root is a named subdirectory. Verified correct.

**`buildBranchLessonData` signature change:** `cfg *config.Config` replaced with `engine *prompt.Engine`. Call site in `runKnowledgeAddBranch` no longer loads config separately. No dead config-loading code remains. Verified consistent.

**Backward compat:** `gopkg.in/yaml.v3` ignores unknown fields; existing `qode.yaml` files with `project:` or `workspace:` sections load cleanly. Spec assumption confirmed by library documentation.

**`filepath.Base(root)` injection safety:** Flows only into markdown template text (command descriptions, headings). Never passed to `os/exec`, shell interpolation, or `text/template` execution path that could cause harm. Verified across both scaffold files.

---

## Summary

| Severity | Count |
| -------- | ----- |
| Critical | 0     |
| High     | 0     |
| Medium   | 2     |
| Low      | 2     |
| Nit      | 1     |

**Overall assessment:** The implementation is correct on the happy path and the simplification from a detection-heavy two-step flow to a single idempotent command is well-executed. The scoring.yaml separation solves the rubric-overwrite problem cleanly at the config layer. The two Medium findings are both real: one is an error-handling gap that silently misbehaves under non-standard filesystem conditions; the other is a coverage gap for the new merge path that is otherwise central to the feature.

**Top 3 most important things to fix before merging:**

1. **M1** (`internal/cli/init.go:71`) — Fix the `os.Stat` guard to surface non-NotExist errors instead of silently skipping the scoring.yaml write.
2. **M2** (`internal/config/config_test.go`) — Add a unit test that proves `config.Load` merges rubrics from `.qode/scoring.yaml` into the loaded config.
3. **L2** (`internal/cli/init_test.go:883`) — Remove the dead `tmplCount` variable.

---

## Rating

| Dimension | Score | What you verified (not what you assumed) |
|-----------|-------|------------------------------------------|
| Correctness (0–3) | 2.5 | Core behaviour correct; cfgForYaml copy semantics verified safe; rootCmd.Version fallback exercised by test. One real bug: non-NotExist stat errors silently skip scoring.yaml write (M1). |
| CLI Contract (0–2) | 2.0 | `qode ide` removed from root.go AddCommand list; TestRootCmd_NoIDESubcommand verifies via Cobra Find. `qode init` produces qode.yaml + .qode/ dirs + scoring.yaml + IDE configs in one invocation, verified by seven integration tests. Re-run asymmetry (qode.yaml overwritten, scoring.yaml preserved) documented and tested. |
| Go Idioms & Code Quality (0–2) | 1.5 | Dead `cfg *config.Config` parameter in public scaffold API never read (L1). Dead `tmplCount` variable in test (L2). Otherwise idiomatic: explicit errors with %w, no magic numbers, clear names, function sizes within standard. |
| Error Handling & UX (0–2) | 1.5 | Non-NotExist stat errors silently swallowed in init.go:71 (M1). All other error paths are explicit with %w wrapping and include file paths. mergeScoringFromFile propagates parse errors correctly. |
| Test Coverage (0–2) | 1.5 | mergeScoringFromFile and the new Load merge path have no direct unit test (M2). All seven specified init behaviours have integration tests. Both scaffold Setup functions verified with root-name assertions. |
| Template Safety (0–1) | 1.0 | filepath.Base(root) flows only into markdown text in claudecode.go and cursor.go, verified never reaches os/exec or shell interpolation. Embedded FS template names are internal — no user-controlled path traversal possible. |

**Total Score: 10.0/12**  
**Minimum passing score: 10/12**
