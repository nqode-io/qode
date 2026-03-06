# Code Review — feat-support-apps-directory

## Issues

### 1.
- **Severity:** Medium
- **File:** `internal/detect/detect.go:47-51`
- **Issue:** Container dir detection in `Composite()` always runs `detectAt()` on the container itself (depth=1) even when children are found. This means `apps/` with a workspace-root `package.json` will produce a layer at `./apps` AND layers at `./apps/web`, `./apps/api`. The existing `dedup()` only removes root-level (path=`.`) duplicates, not container-level ones. If `apps/package.json` has React and `apps/web/package.json` also has React, both `./apps` (react) and `./apps/web` (react) survive dedup.
- **Suggestion:** Only call `detectAt()` on the container dir itself when no children were found:
```go
if knownContainerDirs[subPath] {
    childLayers := detectContainerChildren(subAbs, subPath)
    if len(childLayers) > 0 {
        layers = append(layers, childLayers...)
    } else {
        layers = append(layers, detectAt(subAbs, "./"+subPath, 1)...)
    }
    continue
}
```
This mirrors the `countContainerProjects()` logic in `workspace.go` which only falls back to treating the container as a project when no children are found.

### 2.
- **Severity:** Low
- **File:** `internal/workspace/workspace.go:53-56`
- **Issue:** When a directory matches `knownContainerDirs`, the `isGitRepo` check still runs (line 49-51) but the `continue` on line 55 skips the `looksLikeProjectDir` check. However, a container dir that is also a git subrepo would increment `subRepoCount` but its children would increment `techDirCount`. This is technically correct but could lead to an unexpected `multirepo` classification if two container dirs are also git repos. Unlikely in practice.
- **Suggestion:** No change needed — edge case is extremely unlikely and current behavior is defensible.

### 3.
- **Severity:** Nit
- **File:** `internal/detect/detect_test.go:255-287`
- **Issue:** `TestComposite_ContainerRootDedup` asserts that `./apps/web` is found but does not assert that `./apps` is NOT present. The test name implies dedup but doesn't verify the dedup actually happened.
- **Suggestion:** Add a check that no layer exists at path `./apps`:
```go
for _, l := range layers {
    if l.Path == "./apps" {
        t.Errorf("expected container-level ./apps to be deduped, but found: %+v", l)
    }
}
```

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 1 |
| Low | 1 |
| Nit | 1 |

**Overall assessment:** Clean, well-structured implementation that follows existing codebase patterns. Functions are appropriately sized, error handling is consistent with the existing code, and test coverage is thorough with good edge case coverage (10 new tests across both packages). The one medium issue (container-level layer not deduped when children exist) should be fixed before merging to avoid stale layers in generated `qode.yaml`.

**Top 3 things to fix before merging:**
1. Fix `Composite()` to skip `detectAt()` on container dirs when children are found (Medium)
2. Strengthen `TestComposite_ContainerRootDedup` to verify dedup actually removes `./apps` (Nit)
3. No other blocking issues

## Rating

| Dimension | Score (0-2) | Justification |
|-----------|-------------|---------------|
| Correctness | 1.5 | Spec implemented correctly; container-level layer not deduped when children exist is a minor correctness gap |
| Code Quality | 2.0 | Clean code, functions under 50 lines, clear naming, follows existing patterns |
| Architecture | 2.0 | Right layers modified, no unnecessary coupling, duplicated constant is justified |
| Error Handling | 2.0 | Consistent with existing code — `os.ReadDir` errors handled, graceful degradation |
| Testing | 1.5 | 10 new tests with good edge case coverage; dedup test could be more thorough |

**Total Score: 9.0/10**
