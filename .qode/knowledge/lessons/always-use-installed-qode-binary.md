### Always use installed qode binary
Never run qode via `go run .` or `go run ./cmd/qode` during development sessions. The current source code might be broken, have compilation errors, or contain bugs that haven't been caught yet. Always assume qode is installed and available on the PATH, and invoke it directly as `qode <command>` through the terminal.

**Example 1:** Incorrect — running from source
```bash
go run ./cmd/qode review code
```

**Example 2:** Correct — using installed binary
```bash
qode review code
```