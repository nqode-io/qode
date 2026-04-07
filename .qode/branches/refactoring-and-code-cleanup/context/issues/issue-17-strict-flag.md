# Issue #17: `--strict` CLI Flag to Complement `scoring.strict: true` in Config

## Summary

Strict mode can only be enabled via `qode.yaml` (`scoring.strict: true`). CI/CD pipelines that want to enforce hard stops on quality gate failures must modify the project config, which forces all developers into strict mode. A `--strict` CLI flag would let a pre-merge hook run `qode review code --strict` while the local developer experience remains lenient. The `config.Load()` function already has a comment noting "CLI flags override all of these at call site" — this is the intended extension point.

## Affected Files

**Config (where strict is defined):**

- `internal/config/schema.go:35` — `ScoringConfig.Strict bool yaml:"strict"`
- `internal/config/defaults.go:11` — default `Strict: false`
- `internal/config/config.go:25` — comment: "CLI flags override all of these at call site"

**Where strict is checked (3 files, 3 locations):**

- `internal/cli/review.go:75` — `if cfg.Scoring.Strict { return fmt.Errorf(...) }` (empty diff check)
- `internal/cli/plan.go:204` — `if cfg.Scoring.Strict { return fmt.Errorf(...) }` (spec prerequisite)
- `internal/cli/start.go:50` — `if cfg.Scoring.Strict { return fmt.Errorf(...) }` (start prerequisite)

**Root command (where the flag should be registered):**

- `internal/cli/root.go:60` — PersistentFlags section (only `--root` exists today)

**Tests (existing strict behavior coverage):**

- `internal/cli/review_test.go` — `TestRunReview_StrictEmptyDiff_Code/Security`
- `internal/cli/plan_test.go` — `TestRunPlanSpec_GuardBlocked_NoAnalysis_Strict`
- `internal/cli/start_test.go` — `TestRunStart_GuardBlocked_NoSpec_Strict`

## Current State

All three strict checks follow the same pattern — strict is read from config, not from any CLI flag:

```go
// review.go:73–79
if diff == "" && !force {
    if cfg.Scoring.Strict {
        return fmt.Errorf("no changes detected: commit code first before running a review")
    }
    fmt.Fprintln(os.Stderr, "No changes detected. Commit some code first.")
    return nil
}

// plan.go:202–208
if result := workflow.CheckStep("spec", ctx, cfg); result.Blocked {
    if cfg.Scoring.Strict {
        return fmt.Errorf("%s", result.Message)
    }
    fmt.Printf("STOP. ...")
}
```

Non-strict: gate violations print a STOP message and return nil (exit 0).  
Strict: gate violations return an error (exit non-zero).

## Proposed Fix

### Step 1: Register `--strict` as a persistent flag in `root.go`

```go
// internal/cli/root.go
var flagStrict bool

// in init() or wherever PersistentFlags are registered:
rootCmd.PersistentFlags().BoolVar(&flagStrict, "strict", false, "enforce strict mode: gate violations cause non-zero exit")
```

### Step 2: Apply the override after `config.Load()` in each command

The same pattern applies in all three affected run functions:

```go
cfg, err := config.Load(root)
if err != nil {
    return err
}
if flagStrict {
    cfg.Scoring.Strict = true  // CLI flag overrides config file
}
```

Apply this to:

- `runReview()` in `review.go` (after line 61)
- `runPlanSpec()` in `plan.go` (after line 189)
- `newStartCmd` RunE in `start.go` (after line 34)

No changes to the strict-checking logic itself — it already works correctly once `cfg.Scoring.Strict` is set.

### Step 3: Add test coverage

For each of the three commands, add a test that verifies the flag overrides a non-strict config:

```go
func TestRunReview_StrictFlag_EmptyDiff(t *testing.T) {
    flagStrict = true
    defer func() { flagStrict = false }()
    // set up non-strict config, empty diff
    // expect error returned (not nil)
}
```

## Usage after implementation

```bash
# CI pre-merge hook — enforce hard stops without changing qode.yaml
qode review code --strict
qode review security --strict
qode plan spec --strict
qode start --strict

# Combine with other flags
qode review code --strict --to-file
```

## Impact

- **CI/CD**: pipelines can enforce strict without repo config changes
- **Developer UX**: local workflow stays lenient by default
- **Backward compatible**: `--strict` defaults to `false`; existing behavior unchanged
- **Minimal change**: ~4 lines per affected file; no new abstractions needed
