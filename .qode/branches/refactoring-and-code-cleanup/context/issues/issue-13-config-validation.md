# Issue #13: `Config.Validate()` at Load Time

## Summary

Bad config values — negative score thresholds, zero-weight dimensions, unknown rubric keys, target scores exceeding the rubric maximum — are only discovered at runtime when a command runs, often producing confusing failures. A `Validate()` call inside `config.Load()` would surface these early with clear, actionable messages.

## Affected Files

| File | Role | Relevant lines |
|------|------|----------------|
| `internal/config/config.go` | `Load()` — needs `Validate()` call | 20–50 |
| `internal/config/schema.go` | Config structs | 1–71 |
| `internal/config/defaults.go` | Default values (reference bounds) | 1–119 |
| `internal/scoring/rubric.go` | `BuildRubric()`, `Rubric.Total()` | 20–27, 132–155 |
| `internal/workflow/guard.go` | `RefineMinScore()` uses `TargetScore` | 71–82 |
| `internal/review/review.go` | Uses `MinCodeScore`/`MinSecurityScore` | 12–36 |
| `internal/prompt/engine.go` | Passes rubric totals to templates | 59–74 |

## Current State

`config.Load()` (lines 20–50) applies no validation — it merges defaults, `qode.yaml`, `.qode/scoring.yaml`, and `~/.qode/config.yaml` and returns immediately:

```go
func Load(root string) (*Config, error) {
    cfg := defaultConfig()
    // merge from files...
    return cfg, nil  // no Validate() call
}
```

**Silent failure examples:**

**Zero-weight dimension** — user adds `weight: 0` to a custom review dimension:

```yaml
# .qode/scoring.yaml
rubrics:
  review:
    dimensions:
      - name: Correctness
        weight: 0  # zero weight
```

`Rubric.Total()` returns 0; score comparisons in `guard.go` and template rendering in `engine.go` silently produce wrong results.

**Unknown rubric key** — user writes `refin` instead of `refine`:

```yaml
rubrics:
  refin:   # typo
    dimensions: [...]
```

`BuildRubric(RubricRefine, cfg)` falls back to the default rubric silently; the custom rubric is ignored.

**Impossible gate** — `min_code_score: 15` when the review rubric totals 12 points:

```yaml
review:
  min_code_score: 15.0
```

Reviews can never pass; error is only discovered after running `qode review code`.

## Proposed Fix

Add a `Validate() error` method on `*Config` and call it at the end of `Load()`.

```go
// config.go
func Load(root string) (*Config, error) {
    cfg := defaultConfig()
    // ... existing merge logic ...
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid qode config: %w", err)
    }
    return cfg, nil
}
```

```go
// validate.go (new file in internal/config/)
func (c *Config) Validate() error {
    var errs []string

    // Score thresholds are non-negative
    if c.Review.MinCodeScore < 0 {
        errs = append(errs, "review.min_code_score must be >= 0")
    }
    if c.Review.MinSecurityScore < 0 {
        errs = append(errs, "review.min_security_score must be >= 0")
    }
    if c.Scoring.TargetScore < 0 {
        errs = append(errs, "scoring.target_score must be >= 0")
    }

    // Rubric keys must be known
    validKeys := map[string]bool{"refine": true, "review": true, "security": true}
    for key := range c.Scoring.Rubrics {
        if !validKeys[key] {
            errs = append(errs, fmt.Sprintf("scoring.rubrics: unknown key %q (valid: refine, review, security)", key))
        }
    }

    // Rubric dimensions: non-empty, positive weights
    for key, r := range c.Scoring.Rubrics {
        if len(r.Dimensions) == 0 {
            errs = append(errs, fmt.Sprintf("scoring.rubrics[%s]: must have at least one dimension", key))
            continue
        }
        for i, d := range r.Dimensions {
            if d.Name == "" {
                errs = append(errs, fmt.Sprintf("scoring.rubrics[%s].dimensions[%d]: name must not be empty", key, i))
            }
            if d.Weight <= 0 {
                errs = append(errs, fmt.Sprintf("scoring.rubrics[%s].dimensions[%d] (%s): weight must be > 0, got %d", key, i, d.Name, d.Weight))
            }
        }
    }

    if len(errs) > 0 {
        return errors.New(strings.Join(errs, "; "))
    }
    return nil
}
```

A follow-up validation (cross-field: threshold vs rubric total) can be added after rubric construction, since `BuildRubric` is called separately. Alternatively, validate thresholds against defaults for the common case.

## Impact

- **Early errors**: config problems surface when loading, not mid-command
- **Clear messages**: errors name the specific field and the expected constraint
- **Prevents cascading failures**: zero-weight totals, impossible gates, and silently ignored rubric keys are all caught before any command logic runs
- **No behavior change** for valid configs — `Validate()` returns `nil` and `Load()` proceeds unchanged
