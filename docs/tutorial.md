# qode Tutorial — End-to-End Walkthrough

This tutorial takes you through one full feature, ticket to merge, the way qode is intended to be used. It demonstrates every workflow step, how to split one ticket across multiple subtasks, and the small habits that get the most out of your AI IDE.

If you only have ten minutes, read **Step 0** and the **Pro tip** boxes — those are the points that change output quality the most.

## The scenario

Ticket `ACME-456 — Add user profile editing`. The work is big enough to split into two subtasks under a single branch:

1. **Backend** — `PATCH /users/:id` endpoint, validation, tests.
2. **Frontend** — profile-edit form wired to the new endpoint.

One branch (`feat-user-profile-editing`), two qode contexts (`backend-api`, `frontend-form`), one tutorial.

## Step 0 — set up once

### Install

See the [README](../README.md#installation) for platform-specific commands. After installing, run:

```bash
qode --version
```

### Pick an IDE

qode ships the same 10 workflow names for **Claude Code** (CLI / desktop / IDE plugin), **Cursor**, and **Codex**. Cursor and Claude Code receive slash commands; Codex receives skills generated under `.agents/skills/`.

### Configure MCP servers

Ticket fetching, PR creation, and PR-comment resolution all flow through MCP servers configured in your IDE — qode itself holds no API keys. Set up the servers for your ticketing system (Jira, Linear, GitHub, Azure DevOps, Notion) and any linked-resource services (Figma, Google Drive, Confluence) before you start. The full setup guide is [how-to-use-ticket-fetch.md](how-to-use-ticket-fetch.md).

### Initialise the repo

Run once per project:

```bash
cd your-project
qode init
```

You get:

- `qode.yaml` — review thresholds, scoring config, IDE toggles, diff command.
- `.qode/scoring.yaml` — three rubrics: `refine` (for `/qode-plan-refine`), `review` (for `/qode-review-code`), `security` (for `/qode-review-security`). Each rubric has a `min_*_score` gate in `qode.yaml`. Strict mode (`scoring.strict`, default `false`) makes those gates blocking when flipped to `true`.
- `.qode/prompts/` — local copies of every prompt template. Edit them to match your project's conventions; the embedded defaults stay as fallback.
- `.cursor/commands/*.mdc`, `.claude/commands/*.md`, and `.agents/skills/*/SKILL.md` — generated IDE workflows wired into your IDEs.

Commit `qode.yaml`, `.qode/scoring.yaml`, `.qode/prompts/`, `.cursor/`, `.claude/`, and `.agents/skills/` so the whole team works against the same rubrics, prompts, and IDE workflows.

> **Invocation syntax.** The examples below use slash-command syntax for brevity. In Codex, invoke the same workflow names as skills instead: `/qode-plan-refine` → `$qode-plan-refine`, `/qode-ticket-fetch` → `$qode-ticket-fetch`, and so on.

> **Pro tip — keep your AI's context fresh.** AI assistants degrade as their conversation history grows: more drift, more hallucination, more "I forgot what we were doing." **Start a new chat (`/clear` in Claude Code, `New chat` in Cursor, new chat in Codex) between every workflow step.** Each qode slash command writes its output to disk under `.qode/contexts/current/`, so the next step always picks up exactly where the previous one left off — chat history is not load-bearing. This single habit moves output quality more than any other tweak.

## Step 1 — create the branch and the first context

Create the branch for the whole ticket:

```bash
git checkout -b feat-user-profile-editing
```

Create the first qode context (one per subtask) and switch to it in one shot:

```bash
qode context init backend-api --auto-switch
```

This creates `.qode/contexts/backend-api/` and points the `current` symlink at it. Every workflow step from here on reads and writes inside the active context.

> **Worktrees.** qode plays well with `git worktree`. If you want to work on two subtasks in parallel without IDE thrashing, run `git worktree add ../qode-feat-backend feat-user-profile-editing` (or one worktree per context). Each worktree has its own `.qode/contexts/current/` symlink, so contexts in different worktrees do not interfere.

## Step 2 — fetch the ticket

In your IDE, run:

```
/qode-ticket-fetch https://your-org.atlassian.net/browse/ACME-456
```

The AI calls the right MCP server and writes to the active context:

- `ticket.md` — title, description, metadata
- `ticket-comments.md` — every comment with author and timestamp
- `ticket-links.md` — linked resources (Figma frames, Google Docs, etc.) with summaries when their MCP servers are configured

You will re-run this command for the second subtask later — same ticket URL, fresh fetch — and that re-fetch will pick up any comments posted while the first subtask was in flight.

Start a new chat before the next step.

## Step 3 — add a focus note

When one ticket spans multiple subtasks, the ticket text itself does not say "in this context, work only on the backend." That's what `notes.md` is for:

```
/qode-note-add Focus: backend only. Implement PATCH /users/:id, validation, and tests. The frontend form is handled in a separate context (frontend-form).
```

Everything after `/qode-note-add` is treated as note content, so both single-line and multi-paragraph notes work. If you want an explicit stop marker, `end note` is allowed but optional.

Every downstream qode command reads `notes.md` automatically, which is how you keep refinements and specs scoped to the current subtask.

> **Combining workflows in one prompt.** In Cursor and Claude Code, qode workflows are slash commands, so you can chain them with prose freely. In Codex, use the matching `$qode-*` skill names instead.
>
> - `add a note to notes.md saying we must keep the existing GET /users/:id response shape unchanged, then run /qode-plan-refine`
> - `update notes.md to drop the rate-limiting requirement (out of scope for this iteration), then re-run /qode-plan-refine`
> - `record in notes.md that the auth-session table is on a separate shard, then re-run /qode-plan-spec`
>
> This is especially handy on the second or third pass — you nudge the analysis without leaving the IDE.

## Step 4 — refine the requirements

```
/qode-plan-refine
```

This runs in two passes:

1. **Worker** — produces an analysis with no self-score → `refined-analysis.md`.
2. **Judge** — a fresh AI instance scores the analysis against the `refine` rubric, independently → score appended to `refined-analysis.md`.

Iterate until the judge score clears `scoring.target_score` (default 25/25). Each pass is preserved as `iteration-N.md`, so you can diff iterations.

> **Course-correct mid-run.** If you notice while the prompt is executing that it's heading the wrong way — wrong scope, missing constraint, ignoring a comment — you can send a follow-up message **without stopping the run**. Claude Code handles this cleanly: the message gets queued and applied during or after the current turn. Codex behaves the same way in most setups. Cursor's behaviour depends on which model you've selected — verify before relying on it. When in doubt: stop, course-correct in `notes.md`, re-run.

When the score clears the target, **start a new chat** before moving on.

## Step 5 — generate the spec

```
/qode-plan-spec
```

Reads `refined-analysis.md` and emits `spec.md`. The command is gated: it refuses if the latest refine score is below `scoring.target_score`. Use `qode plan spec --force` only when you have an explicit reason to proceed with a sub-target score.

Read the spec. If something is off, **don't edit the spec by hand** — update `notes.md` and re-generate:

```
/qode-note-add The spec assumed `users` and `auth_sessions` share a DB cluster. They don't — `users` lives on the `profiles` shard. Re-run /qode-plan-spec.
```

Start a new chat once the spec is right.

## Step 6 — implement

```
/qode-start
```

The AI reads `spec.md`, the lessons in `.qode/knowledge/`, and the ticket, then writes the code. This is the longest step. The same mid-run rule applies: if you spot something wrong while it's writing, send a corrective message rather than letting it finish on the wrong path.

When the implementation lands, **start a new chat**.

## Step 7 — test locally

Manual. The AI can write tests, but the engineer is responsible for actually verifying correctness, tesing, hitting the new endpoint, and exerciseing edge cases. Capture anything broken in `notes.md` and loop back to `/qode-start` if needed or more probably just instruct the agent on what needs fixing.

## Step 8 — quality gates

```
/qode-check
```

Auto-detects your test runner and linter from project files (`package.json`, `go.mod`, `Cargo.toml`, etc.) and runs them. Must pass before review.

## Step 9 — code review

```
/qode-review-code
```

Reviews the diff against the hardened code-review prompt and the `review` rubric. Output: `code-review.md` plus a score. If below `review.min_code_score`, address the findings and re-run.

> **Verify the diff scope.** The review prompt occasionally builds against a partial diff (only the last commit instead of the whole branch). Before trusting a review, run `git diff main --stat` in a terminal and confirm the changed-file list matches what the review covered. Read the full per-file diffs directly when in doubt.

## Step 10 — security review

```
/qode-review-security
```

Same shape as code review, scored against the `security` rubric. Output: `security-review.md`. Address findings and re-run if below `review.min_security_score`.

## Step 11 — create the pull request

```
/qode-pr-create
```

The AI assembles a PR title and description from the context (refined analysis, spec, diff, reviews) and opens the PR via your Git provider's MCP server. The PR URL is saved in the context for later steps.

## Step 12 — resolve PR comments

When reviewers leave comments:

```
/qode-pr-resolve
```

The AI fetches the comments via MCP, addresses each one (code change, reply, or both), and pushes. Re-run after every fresh round of comments.

## Step 13 — capture lessons learned (optional, recommended)

```
/qode-knowledge-add-context
```

Extracts durable lessons from the finished context (gotchas, patterns, surprises) into `.qode/knowledge/lessons/`. Future `/qode-start` runs include these lessons automatically — this is how the project gets smarter over time.

This step may be run at any time before. It should be run whenever you loop in circles with the LLM struggling to understand something or repeating errors.

## Step 14 — clean up the context

```bash
qode context remove backend-api
```

The branch stays. Only the qode context directory is removed.

## Now do the second subtask

Same branch. Different context. Same ticket URL.

```bash
qode context init frontend-form --auto-switch
```

In the IDE:

```
/qode-ticket-fetch https://your-org.atlassian.net/browse/ACME-456
```

This pulls the same ticket again — including any comments posted while you were finishing the backend, plus reviewer feedback on the backend PR. That's exactly why you re-fetch instead of copying the previous `ticket.md`.

```
/qode-note-add Focus: frontend only. Build the profile-edit form against the new PATCH /users/:id endpoint shipped from the `backend-api` context. Re-use components from `src/components/forms/`. The form lives at `src/pages/profile/edit.tsx`. Backend response shape is documented in the merged backend PR.
```

Then walk Steps 4 → 14 again for the frontend subtask. When both subtasks are done, the ticket is done.

## Per-IDE best practices

| IDE | Start a new chat between steps | Combine prose + workflow invocation | Course-correct mid-run |
| --- | --- | --- | --- |
| Claude Code | `/clear` between steps; `/compact` if you want to keep context but trim tokens | Yes — chain prose and commands freely | Yes — follow-up messages queue and apply |
| Cursor | New chat (`Cmd-N` / `Ctrl-N`) between steps | Yes — same as Claude | Model-dependent; verify before relying on it |
| Codex | New chat between steps | Yes — same as Claude | Yes |

For all three:

- **Commit between steps**, even if it feels excessive. Small commits (one per workflow step) keep `git diff main` clean for the review prompt and make rollback trivial when an AI step goes sideways.
- **Use the installed `qode` binary**, never `go run` from a checkout. The local source might be mid-edit and broken; the installed binary is the contract.
- **Keep `notes.md` as your scratch pad.** Anything you'd otherwise type repeatedly into the chat — constraints, scope, "remember that…" — belongs there.

## Quick reference

| Command | When |
| --- | --- |
| `qode init` | Once per project |
| `qode context init <name> --auto-switch` | Per subtask |
| `/qode-ticket-fetch <url>` | After context init |
| `/qode-note-add` | Any time you want to record scope, constraints, corrections with free-form note text |
| `/qode-plan-refine` | Until score ≥ `scoring.target_score` |
| `/qode-plan-spec` | After refine clears the gate |
| `/qode-start` | After spec is correct |
| `/qode-check` | Before review |
| `/qode-review-code` / `/qode-review-security` | After quality gates |
| `/qode-pr-create` | After reviews clear the gates |
| `/qode-pr-resolve` | After reviewers leave comments |
| `/qode-knowledge-add-context` | After PR merges (optional) |
| `qode context remove <name>` | Cleanup per subtask |
| `qode workflow` | Show the canonical 12-step list |
| `qode workflow status` | Show live completion for the active context |

## Further reading

- [qode-yaml-reference.md](qode-yaml-reference.md) — every `qode.yaml` key
- [scoring-yaml-reference.md](scoring-yaml-reference.md) — rubric customisation
- [how-to-customise-prompts.md](how-to-customise-prompts.md) — editing prompt templates
- [how-to-use-ticket-fetch.md](how-to-use-ticket-fetch.md) — MCP setup per service
- [versioning.md](versioning.md) — release process and semver policy
