# Issue #18: `internal/iokit` Package for Shared File Utilities

## Summary

Three file I/O patterns recur across the codebase without a shared home:

1. **`readFileOr`** — defined in `internal/context/context.go:155–161`, not exported, cannot be reused
2. **`os.MkdirAll` + `os.WriteFile`** — duplicated sequence in at least 10 production files with inconsistent error messages
3. **`writeFile` helper** — exists in `internal/scaffold/cursor.go:149–154` but is package-private

A small `internal/iokit` package with `ReadFileOrString`, `WriteFile`, `AtomicWrite`, and `EnsureDir` would centralize these patterns, produce consistent error messages, and eliminate the duplication.

## Affected Files

**`readFileOr` definition (unexported, needs sharing):**

- `internal/context/context.go:155–161` — local helper used at lines 52, 71, 106, 109

**`writeFile` helper (local, not shared):**

- `internal/scaffold/cursor.go:149–154` — used only within that file; `claudecode.go` has its own inline version

**Files with inline `os.MkdirAll` + `os.WriteFile` pattern:**

- `internal/cli/init.go:53–120` (3 write calls, MkdirAll loop)
- `internal/cli/branch.go:67–79`
- `internal/cli/knowledge_cmd.go:69–79`
- `internal/cli/review.go:90`
- `internal/plan/refine.go:89–134` (5 WriteFile calls, 2 MkdirAll calls)
- `internal/knowledge/knowledge.go:156–167`
- `internal/scaffold/claudecode.go:12`
- `internal/config/config.go:75`

## Current State

**`readFileOr` in `context.go` (lines 155–161):**

```go
func readFileOr(path, fallback string) string {
    data, err := os.ReadFile(path)
    if err != nil {
        return fallback
    }
    return string(data)
}
```

Unexported — cannot be used in `plan`, `review`, or `workflow`.

**Local `writeFile` in `cursor.go` (lines 149–154):**

```go
func writeFile(path, content string) error {
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return err
    }
    return os.WriteFile(path, []byte(content), 0644)
}
```

Duplicated inline in `claudecode.go` and elsewhere.

**Inline pattern in `init.go` (lines 105–112):**

```go
for name, content := range templates {
    dst := filepath.Join(root, config.QodeDir, "prompts", name+".md.tmpl")
    if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
        return err
    }
    if err := os.WriteFile(dst, content, 0644); err != nil {
        return fmt.Errorf("writing template %s: %w", dst, err)
    }
}
```

**Errors are inconsistently wrapped** — some sites wrap with `fmt.Errorf("... %w", err)`, others return bare errors. Permission modes (`0644`, `0600`, `0755`) are chosen ad-hoc at each call site.

## Proposed Fix

Create `internal/iokit/iokit.go`:

```go
package iokit

import (
    "fmt"
    "os"
    "path/filepath"
)

// ReadFileOrString reads a file and returns defaultVal if it doesn't exist.
func ReadFileOrString(path, defaultVal string) (string, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return defaultVal, nil
        }
        return "", err
    }
    return string(data), nil
}

// WriteFile creates parent directories and writes data to path.
func WriteFile(path string, data []byte, perm os.FileMode) error {
    if err := EnsureDir(filepath.Dir(path)); err != nil {
        return err
    }
    if err := os.WriteFile(path, data, perm); err != nil {
        return fmt.Errorf("write %s: %w", path, err)
    }
    return nil
}

// AtomicWrite writes via a temp file + rename to avoid partial writes.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
    if err := EnsureDir(filepath.Dir(path)); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
    if err != nil {
        return fmt.Errorf("create temp: %w", err)
    }
    defer os.Remove(tmp.Name())
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return err
    }
    if err := tmp.Close(); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), path)
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
    if err := os.MkdirAll(path, 0755); err != nil {
        return fmt.Errorf("mkdir %s: %w", path, err)
    }
    return nil
}
```

**Migration steps:**

1. Create `internal/iokit/iokit.go` and `internal/iokit/iokit_test.go`
2. `internal/context/context.go` — replace local `readFileOr` with `iokit.ReadFileOrString`; export is no longer needed
3. `internal/scaffold/cursor.go` — delete local `writeFile`; replace with `iokit.WriteFile`
4. `internal/scaffold/claudecode.go` — same
5. `internal/cli/init.go`, `branch.go`, `knowledge_cmd.go`, `review.go` — replace inline patterns with `iokit.WriteFile`
6. `internal/plan/refine.go` — replace `os.MkdirAll` + `os.WriteFile` sequences; use `iokit.AtomicWrite` for `refined-analysis.md`
7. `internal/knowledge/knowledge.go` — replace `SaveLesson` body with `iokit.WriteFile`
8. `internal/config/config.go` — replace `os.WriteFile` with `iokit.AtomicWrite` (config is critical)
9. `internal/cli/root.go` — `writePromptToFile` (the existing atomic write helper) can delegate to `iokit.AtomicWrite`

## Impact

- **DRY**: ~20 instances of mkdir+write collapsed to `iokit.WriteFile`
- **Consistent errors**: all I/O errors include the file path in the message
- **Correct permissions**: permission choice is made once at the call site, not hidden inside helpers
- **`AtomicWrite`** available for critical files (`refined-analysis.md`, config) — currently only `writePromptToFile` does atomic writes
- **No external dependencies**: pure stdlib (`os`, `path/filepath`, `fmt`)
