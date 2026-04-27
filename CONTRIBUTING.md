# Contributing to qode

Thanks for your interest in contributing to qode! This project is MIT-licensed and we welcome contributions of all kinds — bug fixes, new features, documentation improvements, and more.

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to abide by its terms.

## Getting Started

### Prerequisites

- **Go 1.24+** — verify with `go version`
- **golangci-lint** — install via Homebrew or Go:
  ```bash
  # Homebrew (macOS / Linux)
  brew install golangci-lint

  # Or install via Go
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```

### Setup

```bash
git clone https://github.com/nqode-io/qode.git
cd qode
go test ./...
go install ./cmd/qode/
```

## Development Workflow

qode uses its own workflow for development. Once you have qode installed:

```bash
qode context init <name> --auto-switch  # Create a new work context and switch to it
/qode-ticket-fetch <url> (in IDE)       # Fetch ticket into context
/qode-plan-refine (in IDE)              # Refine requirements — worker + scoring pass
/qode-plan-spec (in IDE)                # Generate tech spec
/qode-start (in IDE)                    # Run implementation prompt
# Test locally (manual)
/qode-check (in IDE)                    # Run quality gates (tests + lint)
/qode-review-code (in IDE)              # Code review
/qode-review-security (in IDE)          # Security review
/qode-pr-create (in IDE)                # Create pull request via MCP
/qode-pr-resolve (in IDE)               # Resolve PR review comments via MCP
/qode-knowledge-add-context (in IDE)    # Capture lessons learned (optional)
qode context remove                     # Cleanup
```

See the [README](README.md) for the full workflow.

## Quality Gates

All contributions must pass these checks:

| Gate | Command |
|------|---------|
| Unit tests | `go test ./...` |
| Lint | `golangci-lint run` |

Run `/qode-check` (in IDE) to execute all gates interactively.

## Code Style

- Functions must be 50 lines or fewer with a single responsibility
- Handle all errors explicitly
- Use named constants — no magic numbers
- No TODO comments in committed code
- Follow the patterns in existing files — do not introduce new patterns

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feat-description`) and initialise a context (`qode context init feat-description --auto-switch`)
3. Make your changes
4. Ensure all quality gates pass (`/qode-check` in IDE)
5. Open a pull request against `main`

## Reporting Issues

Use our [issue templates](https://github.com/nqode-io/qode/issues/new/choose) to report bugs, request features, or ask questions.
