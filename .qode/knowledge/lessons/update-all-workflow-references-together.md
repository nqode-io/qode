### Update all workflow references together when adding steps
When adding a new step to the qode workflow, there are multiple places that list the workflow steps: `root.go` (CLI long description), `help.go` (workflow diagram), `README.md`, and `workflowRule()`. Missing any location creates inconsistencies — for example, dropping `qode check` from `root.go` while keeping it everywhere else. Always grep for existing step numbering across all files before modifying.

**Example 1:** Find all workflow step references
```bash
grep -rn "qode check\|quality gates\|Ship\|Cleanup\|branch remove" --include="*.go" --include="*.md"
```