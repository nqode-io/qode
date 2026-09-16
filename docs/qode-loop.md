# qode Ticket Loop (repo-internal)

A driver protocol for implementing this repository's GitHub issues one at a time with a
coding agent, using qode's own workflow (`/qode-*` commands) end to end. It is **contributor
tooling for the qode repo, not a qode feature**: nothing here ships in the binary or in
scaffolded assets. It is written for any agent that can invoke the project's qode commands,
spawn subagents on selectable models, ask its operator questions, and schedule its own
wake-ups; Claude Code specifics are in the last section.

One iteration = read state, advance the current ticket by one or more stages, reschedule.
**Serial: exactly one ticket in flight, ever.**

Start it with `/qode-loop` (Claude Code). `/qode-loop 80 81` overrides the ticket sequence
below with the given issue numbers, in order.

## Ticket sequence

Default sequence, in dependency order — do not reorder:

`72, 73, 74, 75`

- [#72](https://github.com/nqode-io/qode/issues/72) plan refine resolves open questions interactively before the judge pass
- [#73](https://github.com/nqode-io/qode/issues/73) OpenCode agent support (`question` tool handling)
- [#74](https://github.com/nqode-io/qode/issues/74) `qode init --config`; init reads an existing config and scaffolds only enabled entries
- [#75](https://github.com/nqode-io/qode/issues/75) rename IDEs to Agents, release `v0.4.0-beta`

**Progress as of 2026-09-16:** none merged; the next ticket is **72**. Update this line when a
ticket merges; the loop itself infers state from `.qode/contexts/`, not from here.

## Model assignments (owner decision, 2026-09-16)

| Role | Model tier | Claude mapping |
| --- | --- | --- |
| Refine worker, spec, implementation, ALL fix application (check fixes, post-review fixes, pr-resolve fixes) | strong builder | Opus |
| Refine judges, code review and security review subagents | strongest reviewer | Fable |
| PR bodies, status summaries — mechanical drafting only | economical drafter | Sonnet |

The orchestrating agent itself does the zero-delegation work: git/qode commands, bookkeeping,
score parsing, PR polling. Every worker/judge/reviewer runs as a **fresh subagent** — the
author of an artifact never scores it.

## Dogfooding: which `qode` binary drives the loop

The loop drives each ticket with the qode that the previous ticket shipped.

- **Before the first ticket:** whatever `qode` is on `PATH` (a released build).
- **After every merge:** `git checkout main && git pull && go install ./cmd/qode/` and, from
  then on, run every `qode` command with the Go bin dir first:
  `PATH="$(go env GOPATH)/bin:$PATH" qode …`. Homebrew's bin dir precedes it on most
  machines, so the prefix is what makes the freshly built binary win. Verify with
  `PATH="$(go env GOPATH)/bin:$PATH" qode --version` printing `dev`.
- **Never `go run ./cmd/qode`.** Local source may be mid-edit; the installed binary is the contract.

**Regenerating tracked assets.** This repo commits `qode.yaml`, `.qode/prompts/`,
`.qode/scoring.yaml`, `.cursor/`, `.claude/commands/`, and `.agents/skills/`. The prompt
engine reads `.qode/prompts/` **before** the embedded templates, so a ticket that changes an
embedded template or a scaffold template is not exercised by this repo until the tracked
copies are regenerated. When a ticket touches `internal/prompt/templates/` or
`internal/scaffold/`, the implement stage ends with:

```bash
go install ./cmd/qode/
PATH="$(go env GOPATH)/bin:$PATH" qode init
git checkout -- qode.yaml          # until #74 lands, init rewrites qode.yaml with qode_version: dev
git add .qode/prompts .cursor .claude/commands .agents/skills && git commit -m "chore: regenerate scaffolded assets"
```

Regenerated `.claude/commands/qode-loop.md` does not exist — `qode init` only writes the
`qode-*` workflow commands it knows about, so the loop command survives regeneration.

## State

`.qode/contexts/current/loop-state.json` (contexts are gitignored):

```json
{ "ticket": 72, "stage": "refine", "refineIteration": 2,
  "codeReviewRound": 0, "securityReviewRound": 0, "prNumber": null }
```

Stage ∈ `setup | refine | spec | implement | manual-tests | check | code-review |
security-review | pr-create | pr-gate | release`.

Orchestrator bookkeeping runs through the stable helper in [`tools/qode-loop/`](../tools/qode-loop)
— `state init|get|set` and `refine-score <iteration> <score> <max>` — never through ad-hoc
inline python/sed one-liners, so the commands are whitelistable (`go run ./tools/qode-loop …`).
The helper imports nothing from `internal/`, so it builds even while the tree is mid-implementation.

If the state file is missing but a context exists, infer the stage from
`PATH=… qode workflow status` and the artifacts (refined-analysis header score / spec.md /
branch commits / code-review*.md / PR existence). If no context exists, the previous ticket is
done — start the next ticket in the sequence.

## Gates and caps

- **Refine:** judge score must equal the rubric maximum — currently **25/25**, read `M` from
  the judge's `**Total Score:** S/M` line rather than assuming it. Never bypass with `--force`.
  Minimum 2 iterations even if an early pass scores full marks. Cap: **8 iterations**, then
  pause and ask the operator.
- **Code review ≥ 10/12. Security review ≥ 10/12** (from [qode.yaml](../qode.yaml)).
  Each review loop: findings fixed by the builder tier → **fresh** reviewer re-reviews (round
  N reviewers receive prior rounds and must give per-finding fixed/not-fixed verdicts verified
  by their own probes, never the fixer's word). Cap: **4 rounds each**, then pause and ask.
- After a review passes, do not touch code except through the next gated stage. Residual
  Low/Nit findings ride to the PR and are listed in its body.
- Fixers may dispute a finding they believe is wrong; a dispute is recorded in `notes.md` and
  put to the next reviewer round rather than silently ignored.

## Loop policy answers for in-command prompts

The `/qode-*` commands ask the user questions. Running unattended, the orchestrator answers
them per this policy instead of blocking:

| Prompt | Answer |
| --- | --- |
| "Post … as a new ticket comment?" (refine / spec / reviews) | **No.** The PR body carries the results; the loop does not comment on issues. |
| `/qode-check` "Accept / Stop / Comment" on failures | **Accept**, fixes applied by a builder-tier subagent. After 3 consecutive failing rounds, pause and ask the operator. |
| `/qode-plan-refine` clarification pass | Pick the candidate the ticket, the codebase or this document determines; answer "use your judgement" when none of them does, which the command records as an `(assumed)` entry. **Owner-level decisions** (naming, public CLI surface, breaking changes not settled by the ticket) go to the operator via a blocking question **before the spec**, and are recorded in the analysis as DECIDED items. |
| `/qode-pr-resolve` "wait for the user to confirm" | Confirm — the operator's review comments are the confirmation. |
| Anything asking for `--force` | **Never.** |

## Stages

Each qode stage goes through its `/qode-*` command, never the raw CLI — the commands carry the
bookkeeping protocol. Remember the `PATH` prefix from the dogfooding section once a ticket has merged.

1. **setup** (orchestrator): `git checkout main && git pull`; delete the previous merged local
   branch; create branch `<issue>-<shortslug>` (e.g. `72-interactive-refine`);
   `qode context init <issue>-<shortslug> --auto-switch`; fetch the issue into
   `.qode/contexts/current/ticket.md` — via `/qode-ticket-fetch <url>` when a GitHub MCP
   server is available, otherwise `env -u GITHUB_TOKEN gh issue view <n> --json title,body,url`
   written as `# <title>`, a `**URL:** <url>` line, and the body (the `**URL:**` line is what
   `/qode-pr-create` uses for `Closes #N`); `go run ./tools/qode-loop state init ticket=<n>`.
2. **refine**: invoke `/qode-plan-refine` and follow it. The worker prompt is executed by a
   builder-tier subagent writing `refined-analysis.md` (`<!-- qode:iteration=N -->` header).
   The judge pass runs as an independent reviewer-tier subagent — fresh context, never the
   author — reading `CLAUDE.md`, the ticket, and the analysis, and re-running any empirical
   claim it can probe. Orchestrator: parse `S/M`, run
   `go run ./tools/qode-loop refine-score N S M`, `state set refineIteration=N`. Below the
   maximum (or below 2 iterations): the next worker pass incorporates the judge's top-3
   improvements. Owner-level open questions go to the operator **before** the spec.
3. **spec**: invoke `/qode-plan-spec`. If its command output starts with `STOP.`, return to
   refine. Otherwise a builder-tier subagent writes `spec.md`, committing every decision the
   ticket delegated to the spec, including the documentation set the ticket lists
   (README, `docs/`, `site/index.html`, `CHANGELOG.md`).
4. **implement**: invoke `/qode-start`. A builder-tier subagent executes it: reads
   `CLAUDE.md` + spec, one commit per spec task, colocated table-driven tests, golden files
   regenerated with `-update` when templates change, `go test -race ./...` and
   `golangci-lint run` green, then the asset regeneration block from the dogfooding section
   when templates or scaffold changed, `CHANGELOG.md` entry under `[Unreleased]`, and the
   branch pushed. No scope creep, no unrelated refactors; deviations from spec are reported,
   not silently absorbed.
5. **manual-tests**: exercise the shipped behaviour headlessly — run
   `PATH=… qode init` in a scratch copy, render the affected prompt with the relevant
   `qode …` command, diff generated assets — and record results in the context's `notes.md`.
   Docs-only or test-only tickets: skip with a one-line note.
6. **check**: invoke `/qode-check` and follow its two phases. Failures → builder-tier fixes →
   re-run until green.
7. **code-review** loop: commit everything; invoke `/qode-review-code` (regenerates
   `diff.md`); an independent reviewer-tier subagent performs the adversarial review against
   `CLAUDE.md`, `spec.md`, `diff.md`, and the files on disk — empirical probes allowed with
   mandatory cleanup — writing `code-review.md` (`code-review-N.md` for round N > 1) with the
   full rating table. Below 10/12 → builder-tier subagent applies the findings (check stays
   green, commit+push) → next round with a fresh reviewer; `state set codeReviewRound=N`.
8. **security-review** loop: same shape via `/qode-review-security`, gate 10/12,
   `securityReviewRound`. Update `spec.md` when fixes change contract shapes.
9. **pr-create**: invoke `/qode-pr-create`. A drafter-tier subagent writes the PR body from
   the spec and final reviews using the command's template (what / changes / notes including
   residual findings / quality-gates table / test plan). Orchestrator creates the PR with
   `env -u GITHUB_TOKEN gh pr create` targeting `main`, records `prNumber`, and reports the
   URL to the operator.
10. **pr-gate** (parked — **the operator merges; the loop detects**). `main` requires one
    approving review, which the loop cannot give itself. Poll
    `env -u GITHUB_TOKEN gh pr view <n> --json state,mergedAt,reviews,comments,reviewDecision`:
    - **Merged** → `qode context remove <name>`; delete the local branch; back to fresh main;
      **install the merged qode** (dogfooding section); update the progress line above;
      advance to the next ticket. This is the only transition to a new ticket.
    - **New human review comments** → invoke `/qode-pr-resolve`; fixes by the builder tier; if
      code changed, one fresh reviewer-tier code-review round before pushing the resolution.
    - **CI failed** → treat the failure log as review comments: builder-tier fix, push, re-poll.
    - **Closed without merge** → pause and ask the operator.
    - Otherwise idle until the next poll.
11. **release** (only for a ticket whose issue says it cuts a release, currently #75): after
    its PR merges, ask the operator to confirm the tag, then run `tools/release.sh v0.4.0-beta`
    from a clean `main`, watch `env -u GITHUB_TOKEN gh run list --workflow release.yml`, and
    verify the GitHub Release carries the binaries and `checksums.txt.bundle`. Report the release URL.

## Scheduling

- Active work: continue within the iteration; a long fallback wake-up (20 min+) only guards
  detached work.
- Parked at pr-gate: poll every ~30 minutes, marking no-change ticks as such.
- Operator questions block by design; on answer, continue immediately.
- Stop condition: the last ticket in the sequence merged (and released, if it cuts a release)
  → report the batch complete and end the loop.

Each iteration ends with a status message: ticket, stage reached, gate scores, what's next,
and anything needing the operator.

## Process discipline (binding)

Carried over from the loop this one was modelled on; it holds only if every iteration applies it.

1. **Mutation-test every gate-relevant control, and watch the named test die.** A fix is done
   when deleting the guarded line kills a *named* test re-run against the final tree — never
   when the suite is merely green. Revert mutations by reverse edit, never `git checkout --`.
2. **Judges re-measure; they never re-read.** Every empirical claim in an analysis names the
   command that produced it; a judge spot-checks by executing, and a confident claim that fails
   reproduction is scored down hard.
3. **Assert the behaviour, never the configuration.** The rendered prompt, not the flag; the
   file that was (not) written, not the log line that said so.
4. **Reviews are adversarial and independent.** Fresh reviewer every round; per-finding verdicts
   verified by the reviewer's own probes; the author never scores their own artifact.
5. **Docs move with code.** A ticket is not done while `README.md`, `docs/`, `site/index.html`,
   or `CHANGELOG.md` say something the merged code no longer does (see `CLAUDE.md` gotchas).

## Claude Code specifics

- Start the loop with the `/qode-loop` command (wraps `/loop` self-paced mode), or directly:
  `/loop Execute one iteration of the qode ticket loop per docs/qode-loop.md — read it fully, then act.`
- Stage commands are invoked with the Skill tool (`qode-plan-refine`, `qode-start`, …);
  subagents with the Agent tool using `model: "opus" | "fable" | "sonnet"` per the table
  above; pacing via ScheduleWakeup; operator questions via AskUserQuestion.
- If `gh` returns 401 with a stale `GITHUB_TOKEN` in the environment, strip it per command:
  `env -u GITHUB_TOKEN gh …`. The GitHub MCP server fails the same way; fall back to `gh`.
- `.claude/settings.local.example.json` lists the permissions the loop needs to run unattended;
  copy it to `.claude/settings.local.json` (gitignored) and adjust.
