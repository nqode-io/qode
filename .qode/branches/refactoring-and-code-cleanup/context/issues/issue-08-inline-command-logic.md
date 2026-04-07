# Issue #8: Inline Command Logic in CLI Handlers

## Summary

The qode CLI has inconsistent patterns for command handlers. While some commands (like `newPlanRefineCmd`, `newPlanSpecCmd`) correctly delegate their `RunE` logic to named `run*` functions, others (like `newStartCmd`, `newBranchCreateCmd`, `newKnowledgeListCmd`) embed their entire business logic as inline closures. Inline closures cannot be unit tested independently, while delegated functions are fully testable. Standardizing on the delegation pattern will improve testability, maintainability, and consistency.

## Affected Files

### `internal/cli/start.go`
- `newStartCmd` (lines 19–104): inline closure contains ~70 lines of logic

### `internal/cli/branch.go`
- `newBranchCreateCmd` (lines 37–97): inline closure contains ~52 lines
- `newBranchRemoveCmd` (lines 99–139): inline closure contains ~30 lines

### `internal/cli/knowledge_cmd.go`
- `newKnowledgeListCmd` (lines 26–54): inline closure contains ~23 lines
- `newKnowledgeAddCmd` (lines 56–86): inline closure contains ~24 lines
- `newKnowledgeSearchCmd` (lines 88–116): inline closure contains ~22 lines

### `internal/cli/init.go`
- `newInitCmd` (lines 15–33): mostly correct but RunE is not a clean one-liner

### `internal/cli/help.go` (or root.go)
- `newWorkflowCmd` (lines 14–25): inline print in closure

## Current State

### Problematic example — `newStartCmd` (`start.go`)

```go
func newStartCmd() *cobra.Command {
    var toFile bool
    var force bool
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            root, err := resolveRoot()
            if err != nil {
                return err
            }
            cfg, err := config.Load(root)
            // ... ~60 more lines of business logic inline ...
        },
    }
    return cmd
}
```

The entire start implementation — loading config, checking workflow steps, loading context, reading the knowledge base, building the prompt — is embedded in the closure with no way to test it independently.

### Correct example — `newPlanRefineCmd` (`plan.go`)

```go
func newPlanRefineCmd() *cobra.Command {
    var toFile bool
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            ticketURL := ""
            if len(args) > 0 {
                ticketURL = args[0]
            }
            return runPlanRefine(ticketURL, toFile) // one-liner
        },
    }
    return cmd
}

func runPlanRefine(ticketURL string, toFile bool) error {
    // 43 lines of fully-testable business logic
}
```

## Proposed Fix

Extract the body of each inline `RunE` closure to a named `run*` function. The `RunE` closure becomes a one-liner:

| File | Function | Action |
|------|----------|--------|
| `start.go` | `newStartCmd` | Extract to `runStart(toFile, force bool) error` |
| `branch.go` | `newBranchCreateCmd` | Extract to `runBranchCreate(name, base string) error` |
| `branch.go` | `newBranchRemoveCmd` | Extract to `runBranchRemove(name string, keepCtx bool) error` |
| `knowledge_cmd.go` | `newKnowledgeListCmd` | Extract to `runKnowledgeList() error` |
| `knowledge_cmd.go` | `newKnowledgeAddCmd` | Extract to `runKnowledgeAdd(src string) error` |
| `knowledge_cmd.go` | `newKnowledgeSearchCmd` | Extract to `runKnowledgeSearch(query string) error` |
| `init.go` | `newInitCmd` | Clean up to `return runInitExisting(root)` one-liner |
| `help.go` | `newWorkflowCmd` | Extract to `runWorkflow() error` |

After refactoring, a test can call `runStart(false, true)` directly without invoking the cobra framework, enabling isolated unit tests for all command logic.

## Impact

- **Testability**: named `run*` functions can be called directly in tests; inline closures cannot
- **Consistency**: all commands follow the same structure, reducing cognitive overhead
- **Separation of concerns**: CLI wiring stays in the `new*Cmd` constructor; business logic lives in `run*`
