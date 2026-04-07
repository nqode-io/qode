# Issue #1: Rubric Definition Duplication

## Summary

The project maintains two parallel, hand-synchronized representations of the same rubric definitions:

1. **Canonical** (`internal/scoring/rubric.go`): `DefaultRefineRubric`, `DefaultReviewRubric`, `DefaultSecurityRubric` as Go `Rubric` structs
2. **Mirror** (`internal/config/defaults.go`): `DefaultRubricConfigs()` returning identical data as `RubricConfig` structs

Any change to a rubric must be applied in both places. A test (`TestBuildRubric_DefaultReviewSecurityMirrorConfig` in `rubric_test.go`) exists solely to verify the two sources stay in sync — which is itself a symptom of the duplication.

## Affected Files

**Duplication sources:**
- `internal/scoring/rubric.go` lines 38–127 — `DefaultRefineRubric`, `DefaultReviewRubric`, `DefaultSecurityRubric`
- `internal/config/defaults.go` lines 25–118 — `DefaultRubricConfigs()` with identical dimension data

**Type definitions:**
- `internal/scoring/rubric.go` lines 15–35 — `Rubric` and `Dimension` structs
- `internal/config/schema.go` lines 19–30 — `RubricConfig` and `DimensionConfig` structs

**Consumers (read-only, no changes needed):**
- `internal/cli/init.go:72` — calls `DefaultRubricConfigs()` to seed `scoring.yaml`
- `internal/review/review.go:18,32` — calls `scoring.BuildRubric(..., cfg)`
- `internal/workflow/guard.go:49,77,82` — calls `scoring.BuildRubric(..., cfg)`
- `internal/cli/help.go:160–161` — calls `scoring.BuildRubric(..., cfg)`

**Test to delete after fix:**
- `internal/scoring/rubric_test.go` lines 131–151 — `TestBuildRubric_DefaultReviewSecurityMirrorConfig` verifies the two sources match; obsolete once there is only one source

## Current State

`DefaultRefineRubric` in `scoring/rubric.go`:
```go
var DefaultRefineRubric = Rubric{
    Kind: RubricRefine,
    Dimensions: []Dimension{
        {Name: "Problem Understanding", Weight: 5, Desc: "Correct restatement...",
         Levels: []string{"5: Perfect restatement...", "4: Good understanding...", ...}},
        // 4 more dimensions
    },
}
```

`DefaultRubricConfigs()` in `config/defaults.go` mirrors this verbatim:
```go
func DefaultRubricConfigs() map[string]RubricConfig {
    return map[string]RubricConfig{
        "refine": {
            Dimensions: []DimensionConfig{
                {Name: "Problem Understanding", Weight: 5, Description: "Correct restatement...",
                 Levels: []string{"5: Perfect restatement...", "4: Good understanding...", ...}},
                // 4 more dimensions — identical data, different types
            },
        },
        // "review" and "security" also duplicated
    }
}
```

Key structural difference: `Dimension.Desc` vs `DimensionConfig.Description` (field name differs); otherwise the data is identical.

## Proposed Fix

**Single source of truth in `scoring/rubric.go`; derive `DefaultRubricConfigs()` from it.**

### Step 1: Add conversion function in `internal/config/` (new file `conversion.go`)

```go
package config

import "github.com/nqode-io/qode/internal/scoring"

func RubricToConfig(r scoring.Rubric) RubricConfig {
    dims := make([]DimensionConfig, len(r.Dimensions))
    for i, d := range r.Dimensions {
        dims[i] = DimensionConfig{
            Name:        d.Name,
            Weight:      d.Weight,
            Description: d.Desc,
            Levels:      d.Levels,
        }
    }
    return RubricConfig{Dimensions: dims}
}
```

### Step 2: Rewrite `DefaultRubricConfigs()` in `config/defaults.go`

Replace lines 27–118 with:

```go
func DefaultRubricConfigs() map[string]RubricConfig {
    return map[string]RubricConfig{
        string(scoring.RubricRefine):   RubricToConfig(scoring.DefaultRefineRubric),
        string(scoring.RubricReview):   RubricToConfig(scoring.DefaultReviewRubric),
        string(scoring.RubricSecurity): RubricToConfig(scoring.DefaultSecurityRubric),
    }
}
```

Add `"github.com/nqode-io/qode/internal/scoring"` to imports. No circular dependency risk — `config` can safely import `scoring`.

### Step 3: Delete the sync-verification test

Remove `TestBuildRubric_DefaultReviewSecurityMirrorConfig` from `rubric_test.go` (lines 131–151). With a single source, the test is obsolete.

All other code — `BuildRubric()`, CLI init, review, guard, help — requires no changes.

## Impact

- **Maintenance**: rubric changes require one edit instead of two
- **Correctness**: divergence between representations is structurally impossible
- **Test cleanup**: one sync-verification test deleted
- **Backward compatibility**: full — `DefaultRubricConfigs()` API is unchanged; existing `scoring.yaml` files continue to work
