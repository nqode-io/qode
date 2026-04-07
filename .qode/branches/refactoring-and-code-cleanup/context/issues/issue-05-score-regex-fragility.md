# Issue #5: Score Extraction via Loose Regex in `scoring.ParseScore`

## Summary

`ParseScore()` in `internal/scoring/scoring.go` uses loose regex patterns to extract scores from unstructured AI-generated text. The patterns match the **first** occurrence of strings like `score: N/M` or `total score: N/M` anywhere in the output. A judge that includes explanatory examples or reference material containing similar strings will cause incorrect score extraction — silently gating or unblocking the workflow based on a wrong number.

Score extraction is a gate decision: `workflow.CheckStep("spec", ...)` in `internal/workflow/guard.go` uses the parsed score to block spec generation. A misextracted score can either block valid work or permit low-quality analysis to advance.

## Affected Files

- `internal/scoring/scoring.go` lines 55–62 — regex patterns for total and dimension scores
- `internal/plan/refine.go:140` — calls `scoring.ParseScore()`
- `internal/workflow/guard.go:41–58` — uses parsed score as a gate condition
- `internal/context/context.go:163–175` — reads score from saved markdown header
- `.qode/prompts/scoring/judge_refine.md.tmpl` — judge prompt that produces the scored output

## Current State

```go
// scoring.go lines 55–62
totalRe := regexp.MustCompile(`(?i)total\s*score[:\s*]*(\d+)\s*/\s*(\d+)`)
dimRe   := regexp.MustCompile(`(?i)(?:score|points)[:\s*]*(\d+)\s*/\s*(\d+)`)

totalMatch := totalRe.FindStringSubmatch(judgeOutput)   // first match only
dimMatches := dimRe.FindAllStringSubmatch(judgeOutput, -1) // positional, in text order
```

**False match scenario:** A judge that writes "Industry standard for this dimension is score: 5/5 for perfect coverage" before its actual judgment causes the dimension regex to pick up `5/5` from the example rather than the real score. The total score regex similarly fires on any "total score: N/M" occurrence in preamble text.

The single test in `internal/scoring/scoring_test.go` covers only a clean happy-path input with no extraneous score-like strings — no adversarial cases exist.

## Proposed Fix

Replace loose text matching with a **structured output contract**: require the judge to append a mandatory YAML block at the end of its response. The parser looks only for this block, ignoring all preceding text.

**Judge prompt addition** (in `judge_refine.md.tmpl`):
```
At the very end of your response, append this block exactly — do not skip it:

```yaml
judgment:
  total_score: <N>
  max_score: <M>
  dimensions:
    - name: "Problem Understanding"
      score: <N>
      max: 5
    # ... one entry per dimension
  improvements:
    - "<first improvement>"
    - "<second improvement>"
    - "<third improvement>"
```
```

**Parser update** (`scoring.go`):
```go
func ParseScore(judgeOutput string, rubric Rubric) Result {
    yamlRe := regexp.MustCompile("(?s)```yaml\\s*\n(.*?)```")
    if matches := yamlRe.FindStringSubmatch(judgeOutput); len(matches) > 1 {
        var j judgmentYAML
        if err := yaml.Unmarshal([]byte(matches[1]), &j); err == nil {
            return j.toResult(rubric)
        }
    }
    // Legacy regex fallback — deprecated, remove after prompt rollout
    return parseLegacyRegex(judgeOutput, rubric)
}
```

## Impact

**Silent failure modes eliminated:**
1. Judge includes example scores in preamble → YAML block parser ignores all text before the block
2. Judge produces dimension scores in a different order than rubric → YAML keys are explicit, not positional
3. Future prompt changes accidentally introduce score-like strings → parser is immune

**Gate correctness:** `workflow.CheckStep` in `guard.go` receives the right score; workflow gating becomes reliable in CI pipelines.

**Rollout path:**
1. Add failing test cases to `scoring_test.go` (judge output with examples containing score patterns)
2. Implement YAML block parsing with legacy regex fallback
3. Update judge prompt templates to require YAML block
4. Remove legacy regex fallback once prompt rollout is complete
