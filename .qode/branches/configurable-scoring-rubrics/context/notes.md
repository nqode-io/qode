# Notes

## Preliminary inputs

- `max_refine_iterations` and `two_pass` config fields were removed
- In the plan refine step, each dimension currently has an explanation for each score in the corresponding prompt template. Add these to the rubrics too.

## Design decisions

- Target score is configurable via `qode.yaml` (e.g. `scoring.target_score`) rather than always defaulting to rubric max.
- Performance is a dimension in the default review rubric (weight 2), resolving the mismatch between the current `review/code.md.tmpl` criteria and `ReviewRubric`.
- Default rubric configs in `internal/config/defaults.go` must reflect the current hardcoded state of `internal/scoring/rubric.go` exactly, including `Levels` for the refine rubric dimensions.
- Breaking schema change (`scoring.refine_target_score` → `scoring.target_score` / `scoring.rubrics`) is acceptable: project is in alpha phase where breaking changes are expected.
- `extractScore` guard updated to `len(m) < 3` and comment updated from `"X/10"` to `"X/M"` after regex gained a second capture group for the denominator.
- `pct` funcmap function is covered by a `BuildCodePrompt` rendering test asserting computed constraint thresholds.
