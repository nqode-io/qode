# Notes

## Design Decision: Remove ContextMode Entirely

The `--to-file` flag invocations must behave identically to stdout output.
Both always use reference mode: templates emit file-read instructions instead
of inlining content. There is no inline mode.

**Consequences:**
- `ContextModeInline` and `ContextModeReference` constants are removed from `engine.go`
- `ContextMode string` field is removed from `TemplateData`
- All builder functions (`BuildRefinePromptWithOutput`, `BuildSpecPromptWithOutput`,
  `BuildStartPrompt`, `BuildCodePrompt`, `BuildSecurityPrompt`) no longer accept
  a `contextMode` parameter
- All templates use reference-mode content only (no `{{if eq .ContextMode "inline"}}` blocks)
- Large content fields (Ticket, Analysis, Spec, Diff) are only populated for the
  `knowledge/add-branch` command and scoring judge, where inline content is needed
  for retrospective lesson extraction across branches; all other templates reference
  files via `{{.BranchDir}}`
- The `--to-file` flag only controls whether the rendered prompt is saved to disk
  or printed to stdout — it does not change prompt content
- `BranchDir` field stays in `TemplateData` (templates need it for file-path references)

## Design Decision: Exclude .qode/ from Git Diff

`DiffFromBase` in `internal/git/git.go` passes `-- :(exclude).qode/` to all three
`git diff` invocations. `.qode/branches/` contains workflow artifacts (spec.md,
code-review.md, diff.md, etc.) that are committed as part of the workflow. Including
them in the review diff wastes tokens and adds noise.

This is portable: `exec.Command` bypasses shell interpretation so `:`, `(`, `)` are
passed literally to git. `:(exclude)` is a git core pathspec feature (since git 1.9),
supported identically on macOS, Linux, and Windows.

Any new `git diff` calls should carry `"--", ":(exclude).qode/"` as the final args.

## Design Decision: Split `qode plan refine` into `qode plan refine` + `qode plan judge`

### Problem
The slash command `/qode-plan-refine` currently calls `qode plan refine` **twice** — once
to get the worker prompt (stdout), and again to generate the judge prompt file
(`.refine-judge-prompt.md`). The judge is buried inside `BuildRefinePromptWithOutput` and
the only way to access it is via the `--to-file` flag, which also saves the worker prompt.
This conflation is confusing and couples two distinct steps.

### Decision
Introduce `qode plan judge` as a first-class subcommand of `qode plan`. The judge command
is primarily consumed by IDE slash commands but is also usable from the terminal (with
`--to-file` for debugging the judge template).

**`qode plan refine`** (simplified) — worker prompt only:
- Generates and outputs the refine/worker prompt (stdout or `--to-file`)
- No longer generates or saves the judge prompt
- `BuildRefinePromptWithOutput` drops the `cfg.Scoring.TwoPass` judge generation block
- `RefineOutput.JudgePrompt` field removed; `SaveIterationFiles` saves worker prompt only

**`qode plan judge`** (new) — judge prompt only:
- Reads `refined-analysis.md` from `ctx.ContextDir` (hard error if absent)
- Calls `scoring.NewEngine(engine, cfg).BuildJudgePrompt(analysisContent, scoring.RefineRubric)`
- `--to-file` saves the rendered prompt to `.refine-judge-prompt.md` for debugging
- stdout outputs the judge prompt (primary use case for IDE slash commands)
- New builder: `plan.BuildJudgePrompt(engine, cfg, ctx) (string, error)` in `internal/plan/refine.go`

### Slash command flow (after split)
**`/qode-plan-refine`** — worker pass only:
  1. Run `qode plan refine`, use stdout as prompt
  2. Save worker output to `refined-analysis.md`
  3. Then run `/qode-plan-judge`

**`/qode-plan-judge`** (new) — judge pass:
  1. Run `qode plan judge`, use stdout as prompt
  2. Parse `**Total Score:** N/25` from judge output
  3. Detect iteration number from `<!-- qode:iteration=N -->` header in `refined-analysis.md`
  4. Rewrite header: `<!-- qode:iteration=N score=S/25 -->`
  5. Write copy to `refined-analysis-N-score-S.md`
  6. Report score; if S >= 25 suggest `qode plan spec`, else suggest re-running `/qode-plan-refine`

### Files to change
- `internal/plan/refine.go` — add `BuildJudgePrompt`, remove judge from `BuildRefinePromptWithOutput`, simplify `SaveIterationFiles`
- `internal/cli/plan.go` — add `newPlanJudgeCmd` + `runPlanJudge`, add to `newPlanCmd`, simplify `runPlanRefine`
- `internal/ide/claudecode.go` — add `qode-plan-judge` slash command, simplify `qode-plan-refine`
- `internal/ide/cursor.go` — same
- `internal/plan/refine_test.go` — add tests for `BuildJudgePrompt`, update existing tests
- `docs/how-to-customise-prompts.md` — document judge command and its `--to-file` use
- `README.md` — update workflow section

### What does NOT change
- `scoring.Engine.BuildJudgePrompt` — unchanged; still takes `workerOutput string`
- The judge template (`scoring/judge_refine.md.tmpl`) — unchanged
- `ParseIterationFromOutput`, `SaveIterationResult` — unchanged; called by the IDE slash command handler, not the CLI command

## Design Decision: Remove `two_pass` Config Flag — Two-Pass Is Always On

The `scoring.two_pass` field has been removed from `ScoringConfig`, `DefaultConfig`, `qode.yaml`,
and `docs/qode-yaml-reference.md`. Two-pass scoring (worker → judge) is now unconditional.

**Why:** Two-pass scoring is qode's core quality mechanism — the worker/judge separation
eliminates self-scoring bias and is what makes the scoring system trustworthy. Making it
optional creates a footgun: a project with `two_pass: false` gets a degraded quality signal
with no visible indication in the workflow. The original flag was a premature affordance for
a trade-off that should never be made.

**Consequences:**
- `ScoringConfig.TwoPass bool` field removed from `internal/config/schema.go`
- `TwoPass: true` removed from `DefaultConfig()` in `internal/config/defaults.go`
- `two_pass: true` removed from `qode.yaml` and `docs/qode-yaml-reference.md`
- Stale comment on `BuildRefinePrompt` ("If cfg.Scoring.TwoPass is true...") removed
- `qode plan judge` (and the judge pass in `/qode-plan-refine`) always works —
  there is no config flag that can disable it
- Existing `qode.yaml` files with `two_pass: true` continue to work (YAML unmarshalling
  ignores unknown fields by default in Go's `yaml.v3`)

## Design Decision: Remove `max_refine_iterations` Config Field

`scoring.max_refine_iterations` has been removed from `ScoringConfig`, `DefaultConfig`,
`qode.yaml`, and `docs/qode-yaml-reference.md`.

**Why:** The field was never read anywhere in the codebase — it had no runtime effect.
More importantly, imposing a hard cap on refinement iterations is the wrong model.
A developer should refine until the analysis is genuinely good (score 25/25), not until
they hit an arbitrary limit. A limit creates pressure to accept a low-quality analysis
rather than iterate further, which defeats the purpose of the scoring system.

**Consequences:**
- `ScoringConfig.MaxRefineIterations int` field removed from `internal/config/schema.go`
- `MaxRefineIterations: 5` removed from `DefaultConfig()` in `internal/config/defaults.go`
- `max_refine_iterations: 5` removed from `qode.yaml` and `docs/qode-yaml-reference.md`
- Existing `qode.yaml` files with `max_refine_iterations` continue to work (YAML
  unmarshalling ignores unknown fields)

## Design Decision: Remove Notes from TemplateData and Context

`Notes string` was a field that held the contents of `context/notes.md` for inline
embedding. Since all templates now use reference mode, there is no consumer of this
field. It has been removed from both `prompt.TemplateData` and `context.Context`.

The `refine/base.md.tmpl` template still instructs the AI to read
`{{.BranchDir}}/context/notes.md` directly, so the notes file is still effective.

## Design Decision: Remove "Prompt written to stdout" stderr Banner

The `# Prompt written to stdout — use --to-file to save.` message was printed to stderr
by every command that outputs a prompt to stdout (plan refine, plan judge, plan spec,
review code, review security, start, knowledge add-branch).

It has been removed from all six call sites in `internal/cli/`.

**Why:** The message is noise. Developers know they can redirect stdout; they do not need
a reminder on every invocation. The `--to-file` flag is documented in `--help` output.
The banner also pollutes piped output in scripts and IDE slash commands that consume stdout.
