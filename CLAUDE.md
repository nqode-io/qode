# qode — Project Context

## Tech Stack

- **default** (go): `.`

## Project Structure

Topology: single

- `./` — default (go)

## Quality Standards

- Minimum code review score: 8.0/10
- Minimum security review score: 8.0/10
- Max function length: 50 lines

## Development Workflow

1. `qode branch create <name>` — Create feature branch
2. `qode ticket fetch <url>` — Fetch ticket context
3. `/qode-plan-refine` — Iterate requirements (target 25/25)
4. `/qode-plan-spec` — Generate tech spec
5. `/qode-start` — Generate and run implementation prompt
6. `/qode-review-code` + `/qode-review-security` — Reviews
7. `qode check` — All quality gates
8. `git commit && git push` — Ship

## Clean Code Rules

- Read existing code before writing new code
- Follow patterns in existing files — do not introduce new patterns
- Functions max 50 lines, single responsibility
- Handle all errors explicitly
- No magic numbers — use named constants
- No TODO comments in committed code

