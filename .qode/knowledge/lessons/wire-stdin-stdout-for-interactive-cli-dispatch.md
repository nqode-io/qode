### Wire stdin/stdout/stderr for interactive CLI dispatch

When dispatching to a subprocess (e.g., `claude`) from a terminal command, wire `cmd.Stdin`, `cmd.Stdout`, and `cmd.Stderr` directly to `os.Stdin`/`os.Stdout`/`os.Stderr` instead of using `--print` flag with buffer capture. Buffered capture makes the process non-interactive: users see no output until completion and cannot abort. Check `isatty` first to fall back to batch mode in non-TTY environments (CI, pipes).

**Example 1:** Interactive wiring
```go
cmd := exec.CommandContext(ctx, binaryPath, promptPath)
cmd.Stdin  = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
return cmd.Run()
```

**Example 2:** TTY detection for CI fallback
```go
func isTTY() bool {
    fi, err := os.Stdin.Stat()
    if err != nil { return false }
    return (fi.Mode() & os.ModeCharDevice) != 0
}
```
