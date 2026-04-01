# .qode/scoring.yaml Reference

Scoring rubrics live in `.qode/scoring.yaml`, separate from `qode.yaml`. This means re-running `qode init` (which always regenerates `qode.yaml` with defaults) never overwrites rubric customisations you have iterated on over time.

## How it is created

`qode init` writes `.qode/scoring.yaml` with the built-in default rubrics the first time it runs in a directory. On subsequent runs the file is left untouched.

To reset your rubrics to the built-in defaults, delete the file and re-run `qode init`.

## How it is loaded

`config.Load` merges configuration in this order:

1. Built-in defaults (in-binary)
2. `qode.yaml` — scoring thresholds (`strict`, `target_score`) and all other settings
3. `.qode/scoring.yaml` — rubric definitions override the defaults
4. `~/.qode/config.yaml` — user-level overrides (optional)

`.qode/scoring.yaml` wins over the built-in rubrics. `qode.yaml` does not contain rubrics and is not consulted for them.

## Structure

```yaml
rubrics:
  refine:
    dimensions:
      - name: Problem Understanding
        weight: 5
        description: Short description used in judge prompt headers
        levels:
          - "5: Perfect restatement in engineering terms"
          - "4: Mostly correct with minor gaps"
          - "3: Adequate but surface-level"
          - "2: Partial; significant gaps"
          - "1: Mostly incorrect or too vague"
          - "0: Missing or completely wrong"
  review:
    dimensions:
      - name: Correctness
        weight: 3
        description: Implements spec correctly, no logic bugs
  security:
    dimensions:
      - name: Injection Prevention
        weight: 3
        description: No SQL/command/template injection vectors
```

Omitting a rubric key (`refine`, `review`, or `security`) falls back to the built-in default for that rubric. Providing an empty `dimensions` list also falls back to the default.

## Fields

### `rubrics`

A map with up to three keys: `refine`, `review`, `security`. Each key maps to a rubric object.

### `rubrics.<name>.dimensions`

A list of scoring dimensions. Each dimension has the following fields:

| Field | Required | Description |
|---|---|---|
| `name` | yes | Display name shown in judge prompts and score tables |
| `weight` | yes | Maximum points for this dimension; determines the rubric total |
| `description` | no | One-line description shown in judge prompt headers |
| `levels` | no | Ordered score descriptions, highest first — only used by the `refine` rubric |

### `levels` format

Each entry is a string in the form `"N: description"` where `N` is the score value. Entries are shown to the AI judge in order, so list the highest score first:

```yaml
levels:
  - "5: Complete analysis covering all edge cases"
  - "4: Good coverage with minor omissions"
  - "3: Core cases addressed; gaps exist"
  - "2: Superficial; important cases missed"
  - "1: Generic or incorrect"
  - "0: Missing"
```

The `review` and `security` rubrics do not use `levels`. The AI judge for those rubrics receives only `name`, `weight`, and `description`.

## Relationship with qode.yaml

Scoring-related settings are split across two files:

| Setting | File |
|---|---|
| `scoring.strict` | `qode.yaml` |
| `scoring.target_score` | `qode.yaml` |
| `review.min_code_score` | `qode.yaml` |
| `review.min_security_score` | `qode.yaml` |
| Rubric dimensions (`refine`, `review`, `security`) | `.qode/scoring.yaml` |

When you change rubric weights, update the corresponding score thresholds in `qode.yaml` to reflect the new totals. For example, if your `review` rubric totals 14 points, a reasonable `min_code_score` might be `10` or `11`.

## Committing

Commit `.qode/scoring.yaml` to your repository so all team members use the same rubrics. The file is created inside `.qode/` rather than the project root to keep it co-located with other qode state.
