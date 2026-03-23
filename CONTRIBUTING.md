# Contributing to qode

Thanks for your interest in contributing to qode! This project is MIT-licensed and we welcome contributions of all kinds — bug fixes, new features, documentation improvements, and more.

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
qode branch create feat-my-feature        # Create a feature branch
# /qode-start (in IDE)                    # Run the implementation prompt
qode check                                # Run all quality gates

qode branch create <name>                 # Create a feature branch
qode ticket fetch <url>                   # Fetch ticket into context
#   /qode-ticket-fetch <url> (in IDE)    
#   /qode-plan-refine (in IDE)            # Refine requirements—iterate to 25/25
#   /qode-plan-spec (in IDE)              # Generate tech spec
#   /qode-start (in IDE)                  # Run implementation prompt
#   / qode-review-code (in IDE)           # Code review
#   /qode-review-security (in IDE)        # Security review
qode check                                # Run all quality gates
#   /qode-knowledge-add-context` (in IDE) # (Recommended) Extract lessons learned
10. qode branch remove <name>             # Cleanup
```

See the [README](README.md) for the full 9-step workflow.

## Quality Gates

All contributions must pass these checks:

| Gate | Command |
|------|---------|
| Unit tests | `go test ./...` |
| Lint | `golangci-lint run` |
| Build | `go build ./...` |
| Code review score | Minimum 8.0/10 |
| Security review score | Minimum 8.0/10 |

Run `qode check` to execute all gates at once.

## Code Style

- Functions must be 50 lines or fewer with a single responsibility
- Handle all errors explicitly
- Use named constants — no magic numbers
- No TODO comments in committed code
- Follow the patterns in existing files — do not introduce new patterns

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`qode branch create feat-description` or `git checkout -b feat-description`)
3. Make your changes
4. Ensure all quality gates pass (`qode check`)
5. Open a pull request against `main`

## Reporting Issues

Use our [issue templates](https://github.com/nqode-io/qode/issues/new/choose) to report bugs, request features, or ask questions.
