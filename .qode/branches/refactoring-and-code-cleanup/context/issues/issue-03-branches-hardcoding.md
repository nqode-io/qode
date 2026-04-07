# Issue #3: Hard-coding `branches[0]` for `branchDir` in Multi-Branch Lesson Extraction

## Summary

`runKnowledgeAddBranch` correctly aggregates ticket, spec, analysis, and review data from all branches passed as arguments, but then uses only `branches[0]` to compute the output directory for the saved prompt. For single-branch usage this is harmless, but for 2+ branches the prompt is silently written to the wrong location.

## Affected Files

**`internal/cli/knowledge_cmd.go`**

- Line 219 in `buildBranchLessonData()`: uses `branches[0]` for `branchDir`
- Line 223: also sets `Branch: branches[0]` in `TemplateData`
- The caller `runKnowledgeAddBranch()` already has the current branch on line 140 (`branch` var from `git.CurrentBranch()`), which is the correct value to use

## Current State

```go
// runKnowledgeAddBranch (lines 135–174)
func runKnowledgeAddBranch(args []string, toFile bool) error {
    root, err := resolveRoot()
    // ...
    branch, err := git.CurrentBranch(root)  // line 140: correct current branch
    // ...
    branches := parseBranchArgs(args)  // line 149: all source branches from args
    fmt.Fprintf(os.Stderr, "Extracting lessons from branches: %s\n", strings.Join(branches, ", "))

    data, err := buildBranchLessonData(root, engine, branches)  // passes all branches
    // ...
    if toFile {
        branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branch))
        // line 163 correctly uses current branch — but only because it has the variable
    }
}

// buildBranchLessonData (lines 176–232)
func buildBranchLessonData(root string, engine *prompt.Engine, branches []string) (prompt.TemplateData, error) {
    var allTicket, allAnalysis, allSpec, allExtra strings.Builder

    for _, b := range branches {  // line 180: correctly loops all branches
        ctx, err := gocontext.Load(root, b)
        // aggregates ticket, analysis, spec, reviews from ALL branches
    }

    // ...
    branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(branches[0]))  // BUG: line 219
    return prompt.TemplateData{
        Branch:    branches[0],   // BUG: line 223
        BranchDir: branchDir,     // wrong for multi-branch
        Ticket:    allTicket.String(),   // correctly aggregated from all
        Analysis:  allAnalysis.String(), // correctly aggregated from all
        // ...
    }, nil
}
```

The data aggregation (lines 180–209) is correct; only the output path computation is wrong.

## Proposed Fix

Pass the current branch into `buildBranchLessonData` so it can set `BranchDir` correctly:

```go
// Change signature
func buildBranchLessonData(root string, engine *prompt.Engine, branches []string, currentBranch string) (prompt.TemplateData, error) {
    // ... existing aggregation loop unchanged ...

    // Line 219 becomes:
    branchDir := filepath.Join(root, config.QodeDir, "branches", git.SanitizeBranchName(currentBranch))

    return prompt.TemplateData{
        Branch:    currentBranch, // use current, not branches[0]
        BranchDir: branchDir,
        // ... rest unchanged ...
    }, nil
}

// Update call site in runKnowledgeAddBranch (line 152):
data, err := buildBranchLessonData(root, engine, branches, branch)
```

Alternatively, remove `branchDir` from `buildBranchLessonData` entirely and let `runKnowledgeAddBranch` set `data.BranchDir` after the call, using the `branch` variable it already holds.

## Impact

**When it manifests:** `qode knowledge add-branch feat/api,feat/ui --to-file` while on any branch other than `feat/api`.

**Expected:** prompt saved to `.qode/branches/<current-branch>/.knowledge-add-branch-prompt.md`

**Actual:** prompt saved to `.qode/branches/feat--api/.knowledge-add-branch-prompt.md` (wrong branch's directory)

**Severity:** Medium — silently wrong for any multi-branch invocation; single-branch usage works by coincidence.
