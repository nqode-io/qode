# Code Review — feat-lessons-learned

## Issues Found

### 1. Medium — `parseLessonHeader` uses `SplitN` with -1 (same as `Split`)

- **Severity:** Low
- **File:** `internal/knowledge/knowledge.go:198`
- **Issue:** `strings.SplitN(content, "\n", -1)` is functionally identical to `strings.Split(content, "\n")`. Using `-1` as the count parameter is misleading — it suggests an intentional limit but actually means "no limit."
- **Suggestion:**
```go
lines := strings.Split(content, "\n")
```

### 2. `isLessonFile` uses path separator matching which could false-positive

- **Severity:** Low
- **File:** `internal/knowledge/knowledge.go:224-226`
- **Issue:** `isLessonFile` checks for `/lessons/` anywhere in the path using `strings.Contains`. This could match paths like `/some-lessons/file.md` or a top-level directory named `lessons` outside the knowledge base. However, in practice this function is only called from `Load()` which scopes files to knowledge base paths, so the risk is minimal.
- **Suggestion:** No change needed given the current usage context. If the function becomes public or is used elsewhere in the future, consider tightening the match to check for the `.qode/knowledge/lessons/` suffix specifically.

### 3. `buildBranchLessonData` silently swallows `git.DiffFromBase` errors

- **Severity:** Low
- **File:** `internal/cli/knowledge_cmd.go:221-224`
- **Issue:** When `git.DiffFromBase()` fails, the error is silently ignored and `diff` stays empty. This matches the existing pattern in other commands (the diff is supplementary context), but a verbose-mode warning would be consistent with the `ListLessons` error handling two lines below.
- **Suggestion:**
```go
d, err := git.DiffFromBase(root, "")
if err != nil {
    if flagVerbose {
        fmt.Fprintf(os.Stderr, "Warning: getting diff: %v\n", err)
    }
} else {
    diff = truncateLines(d, maxDiffLines)
}
```

### 4. No branch name validation for directory traversal

- **Severity:** Medium
- **File:** `internal/cli/knowledge_cmd.go:185-243`
- **Issue:** The spec (Section 8: Security Considerations) explicitly calls out that branch names containing `..` should be rejected to prevent directory traversal. `buildBranchLessonData` passes user-provided branch names directly to `filepath.Join()` for constructing `branchDir` paths without validation. While `filepath.Join` normalizes paths, it doesn't prevent traversal — `filepath.Join(root, ".qode/branches", "../../etc")` resolves to `root/etc`.
- **Suggestion:** Add validation in `parseBranchArgs`:
```go
func parseBranchArgs(args []string) []string {
    var branches []string
    for _, arg := range args {
        for _, b := range strings.Split(arg, ",") {
            b = strings.TrimSpace(b)
            if b != "" && !strings.Contains(b, "..") {
                branches = append(branches, b)
            }
        }
    }
    return branches
}
```

### 5. `SaveLesson` recalculates `ToKebabCase(title)` in collision loop

- **Severity:** Nit
- **File:** `internal/knowledge/knowledge.go:168-179`
- **Issue:** `ToKebabCase(title)` is called once for the initial `base`, then again inside the loop for each collision. The result is deterministic, so it could be computed once.
- **Suggestion:**
```go
func SaveLesson(root, title, content string) error {
    dir := LessonsDir(root)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    kebab := ToKebabCase(title)
    dest := filepath.Join(dir, kebab+".md")
    for i := 2; fileExists(dest); i++ {
        dest = filepath.Join(dir, fmt.Sprintf("%s-%d.md", kebab, i))
    }
    return os.WriteFile(dest, []byte(content), 0644)
}
```

### 6. CLAUDE.md workflow numbering inconsistency

- **Severity:** Low
- **File:** `CLAUDE.md:37-47`
- **Issue:** Steps 1-6 are under headers but step 7 jumps to "Terminal commands:" with `qode check`, then step 8 is under "Either terminal or IDE:" with `/qode-knowledge-add-context`. The ordering is inconsistent — lessons (step 8) should come before quality check (step 7) in a natural workflow (capture lessons → check → ship). The current numbering has check=7, lessons=8, ship=9 in CLAUDE.md, but buildClaudeMD() has lessons=7, check=8, ship=9. The CLAUDE.md file diverges from the generated version.
- **Suggestion:** Align CLAUDE.md with the generated buildClaudeMD() ordering: lessons=7, check=8, ship=9, cleanup=10.

### 7. README.md typo: stray period after `refine`

- **Severity:** Nit
- **File:** `README.md:102`
- **Issue:** `qode plan refine.` has a trailing period that wasn't in the original.
- **Suggestion:** Change to `qode plan refine` (remove the period).

### 8. Missing `knowledge_test.go` from tracked files

- **Severity:** Low
- **File:** `internal/knowledge/knowledge_test.go`
- **Issue:** The new test file is untracked (`??` in git status). It won't be included in the commit unless explicitly added. This is not a code issue but a process observation.
- **Suggestion:** Stage the file before committing: `git add internal/knowledge/knowledge_test.go`

### 9. Missing template files from tracked files

- **Severity:** Low
- **File:** `internal/prompt/templates/knowledge/`
- **Issue:** The two new template files (`add-context.md.tmpl`, `add-branch.md.tmpl`) are untracked. Since templates are embedded via `//go:embed`, they compile into the binary and are needed. They are present (the build passes) but must be committed.
- **Suggestion:** Stage: `git add internal/prompt/templates/knowledge/`

### 10. `root.go` workflow removes `qode check` step

- **Severity:** Medium
- **File:** `internal/cli/root.go:36`
- **Issue:** The workflow in the root command's long description now goes straight from step 7 (security review) to step 8 (knowledge) to step 9 (branch remove), dropping `qode check` entirely. The quality gate step is an important part of the workflow.
- **Suggestion:** Add `qode check` back as step 9, making cleanup step 10:
```go
  8. /qode-knowledge-add-context (in IDE)       # Capture lessons learned
  9. qode check                                 # Run all quality gates
 10. qode branch remove <name>                  # Cleanup
```

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 2 |
| Low | 5 |
| Nit | 2 |

**Overall assessment:** The implementation is solid and follows the spec well. The code is clean, well-structured, and follows existing patterns in the codebase. Functions are appropriately sized, error handling is consistent, and test coverage is comprehensive (20 knowledge tests + 6 new IDE tests). The two medium issues — missing branch name validation (spec requirement) and the dropped `qode check` step in root.go — should be addressed before merging.

**Top 3 things to fix before merging:**
1. **Add branch name validation** — reject `..` in branch names as specified in the security section of the spec
2. **Restore `qode check` in root.go** — the quality gate step was dropped from the workflow description
3. **Align CLAUDE.md ordering** — make lessons=7, check=8, ship=9 consistent with buildClaudeMD() output

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Correctness | 1.5 | Missing branch name validation per spec; CLAUDE.md ordering diverges from generated version; root.go drops qode check |
| Code Quality | 2.0 | Clean, readable code; functions well-sized; follows existing patterns; minor nits only |
| Architecture | 2.0 | Proper layer separation; knowledge package handles storage, CLI handles orchestration, IDE handles generation |
| Error Handling | 1.5 | Consistent with existing patterns but silent DiffFromBase error could use verbose warning |
| Testing | 2.0 | Comprehensive test coverage with unit, integration, and edge case tests |

**Total Score: 9.0/10**
