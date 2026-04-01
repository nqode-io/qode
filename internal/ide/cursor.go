package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
)

const cursorRulesDir = ".cursorrules"
const cursorCommandsDir = ".cursor/commands"

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(root string, cfg *config.Config) error {
	if err := os.MkdirAll(filepath.Join(root, cursorRulesDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, cursorCommandsDir), 0755); err != nil {
		return err
	}

	// Workflow rule.
	if err := writeFile(filepath.Join(root, cursorRulesDir, "qode-workflow.mdc"), workflowRule(cfg)); err != nil {
		return err
	}

	// Clean code rule.
	if err := writeFile(filepath.Join(root, cursorRulesDir, "qode-clean-code.mdc"), cleanCodeRule(cfg)); err != nil {
		return err
	}

	// Slash commands.
	cmds := slashCommands(cfg)
	for name, content := range cmds {
		p := filepath.Join(root, cursorCommandsDir, name+".mdc")
		if err := writeFile(p, content); err != nil {
			return err
		}
	}

	fmt.Printf("  Cursor: %s/ (%d rules, %d commands)\n", cursorRulesDir, 2, len(cmds))
	return nil
}

func workflowRule(cfg *config.Config) string {
	return fmt.Sprintf(`---
description: qode workflow rules for %s
globs: ["**/*"]
alwaysApply: true
---

# qode Workflow

## Project
%s

## Tech Layers
%s

## Workflow
1. Branch → qode branch create
2. Context → qode ticket fetch / edit context/ticket.md
3. Refine → /qode-plan-refine (iterate until pass threshold)
4. Spec → /qode-plan-spec
5. Implement → /qode-start → code in Cursor
6. Check → /qode-check
7. Review → /qode-review-code + /qode-review-security
8. Lessons → /qode-knowledge-add-context (recommended)
9. Ship → git commit && push

## Rules
- Always read existing code before writing new code
- Follow patterns in existing files
- Keep functions under 50 lines
- Handle all errors explicitly
`, cfg.Project.Name, cfg.Project.Name, layerList(cfg))
}

func cleanCodeRule(cfg *config.Config) string {
	stacks := collectStacks(cfg)
	return fmt.Sprintf(`---
description: Clean code enforcement for %s
globs: %s
alwaysApply: true
---

# Clean Code Requirements

## Universal Rules
- Functions/methods: max 50 lines
- Single responsibility per function
- Named constants instead of magic numbers
- Explicit error handling — never swallow errors
- No TODO comments in committed code

## Stack-Specific

%s
`, cfg.Project.Name, globsForStacks(stacks), stackCleanCodeRules(stacks))
}

func slashCommands(cfg *config.Config) map[string]string {
	return map[string]string{
		"qode-plan-refine": fmt.Sprintf(`---
description: Refine requirements for %s
---

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine

Save the worker output to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/refined-analysis.md

**Judge pass (scoring):** Run this command and use its stdout output as your prompt:
  qode plan judge

Then:
1. Parse the "**Total Score:** S/M" line and the "**Pass threshold:** T/M" line from the judge output to get score S, max M, and pass threshold T
2. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
3. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/M -->
4. Write a copy to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/refined-analysis-N-score-S.md
5. Report the score to the user. If S >= T, suggest running /qode-plan-spec. Otherwise suggest re-running /qode-plan-refine.
`, cfg.Project.Name),

		"qode-plan-spec": fmt.Sprintf(`---
description: Generate technical specification for %s
---

Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode plan spec --force`"+` to bypass score gates when needed.

After generating the spec, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/spec.md
`, cfg.Project.Name),

		"qode-review-code": fmt.Sprintf(`---
description: Code review for %s
---

Run this command and use its stdout output as your prompt:
  qode review code

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review code --force`"+` to bypass the uncommitted-diff check.

After completing the review, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/code-review.md
`, cfg.Project.Name),

		"qode-review-security": fmt.Sprintf(`---
description: Security review for %s
---

Run this command and use its stdout output as your prompt:
  qode review security

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review security --force`"+` to bypass the uncommitted-diff check.

After completing the review, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/security-review.md
`, cfg.Project.Name),

		"qode-check": fmt.Sprintf("---\ndescription: Run quality gates for %s\n---\n\n", cfg.Project.Name) + qodeCheckBody,

		"qode-start": fmt.Sprintf(`---
description: Start implementation session for %s
---

Run this command and use its stdout output as your prompt:
  qode start

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode start --force`"+` to bypass the spec prerequisite when needed.

Execute the prompt as your implementation session.
`, cfg.Project.Name),

		"qode-ticket-fetch": ticketFetchCursorCommand(cfg),

		"qode-knowledge-add-context": fmt.Sprintf(`---
description: Extract lessons learned from current session for %s
---

Reflect on the current session and extract actionable lessons learned.

1. First, check existing lessons by running: qode knowledge list
2. Review the full conversation history of this session
3. Identify 1-5 actionable, specific lessons (not generic advice)
4. Write each lesson to its own file at .qode/knowledge/lessons/<kebab-case-title>.md

Each lesson file MUST follow this format:
`+"```"+`
### Title in sentence case
Short description (one paragraph, max 100 words). Be specific.

**Example 1:** What to do or avoid
Code snippet, pattern, or concrete example.
`+"```"+`

Rules:
- Do NOT duplicate existing lessons
- Do NOT include credentials, API keys, or secrets
- Focus on: recurring mistakes, non-obvious patterns, project-specific conventions
- If no actionable lessons can be identified, inform the user
`, cfg.Project.Name),

		"qode-knowledge-add-branch": fmt.Sprintf(`---
description: Extract lessons learned from branch context for %s
---

Run this command and use its stdout output as your prompt:
  qode knowledge add-branch $ARGUMENTS
`, cfg.Project.Name),
	}
}

// helpers

func layerList(cfg *config.Config) string {
	var sb strings.Builder
	for _, l := range cfg.Layers() {
		fmt.Fprintf(&sb, "- %s (%s) at %s\n", l.Name, l.Stack, l.Path)
	}
	return sb.String()
}

func collectStacks(cfg *config.Config) []string {
	seen := map[string]bool{}
	var stacks []string
	for _, l := range cfg.Layers() {
		if !seen[l.Stack] {
			seen[l.Stack] = true
			stacks = append(stacks, l.Stack)
		}
	}
	return stacks
}

func globsForStacks(stacks []string) string {
	globs := []string{}
	for _, s := range stacks {
		switch s {
		case "react", "nextjs":
			globs = append(globs, "**/*.tsx", "**/*.ts", "**/*.jsx", "**/*.js")
		case "angular":
			globs = append(globs, "**/*.ts", "**/*.html", "**/*.scss")
		case "dotnet":
			globs = append(globs, "**/*.cs", "**/*.csproj")
		case "java":
			globs = append(globs, "**/*.java", "**/*.kt")
		case "python":
			globs = append(globs, "**/*.py")
		case "go":
			globs = append(globs, "**/*.go")
		}
	}
	if len(globs) == 0 {
		return `["**/*"]`
	}
	return `["` + strings.Join(dedup(globs), `", "`) + `"]`
}

func dedup(items []string) []string {
	seen := map[string]bool{}
	result := items[:0]
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func stackCleanCodeRules(stacks []string) string {
	var sb strings.Builder
	for _, s := range stacks {
		switch s {
		case "react", "nextjs":
			sb.WriteString("### React / Next.js\n- Prefer functional components\n- Use hooks; no class components\n- Extract reusable logic into custom hooks\n- Use React Query / SWR for server state\n\n")
		case "angular":
			sb.WriteString("### Angular\n- Prefer standalone components\n- Use OnPush change detection\n- Inject services via constructor\n- Avoid any; use proper types\n\n")
		case "dotnet":
			sb.WriteString("### .NET\n- Use async/await throughout\n- Prefer records over classes for DTOs\n- Use Result<T> pattern for error propagation\n- Repository pattern for data access\n\n")
		case "java":
			sb.WriteString("### Java\n- Use Optional instead of null returns\n- Prefer records for DTOs (Java 14+)\n- Use streams; avoid imperative loops where readable\n- Constructor injection for dependencies\n\n")
		case "python":
			sb.WriteString("### Python\n- Type hints on all functions\n- Use dataclasses or Pydantic models for DTOs\n- Prefer f-strings\n- No bare except clauses\n\n")
		}
	}
	return sb.String()
}

func ticketFetchCursorCommand(cfg *config.Config) string {
	if cfg.TicketSystem.Mode == "mcp" {
		return fmt.Sprintf(`---
description: Fetch ticket via MCP for %s
---

Fetch the ticket at the URL provided in $ARGUMENTS using your available MCP tools.

Use whatever MCP tool is available (Jira, Linear, GitHub, Notion, or Azure DevOps server).
If no MCP tool is available, open the URL with a browser tool and extract the content.

Collect: title, description, all comments (author + timestamp), linked resources, attachment
summaries.

Write to the context directory for the current branch:
- context/ticket.md — title, description, metadata
- context/ticket-comments.md — all comments (omit if none)
- context/ticket-links.md — linked resource summaries (omit if none)

Report: "Fetched: <title> — <N> comments, <M> links."
`, cfg.Project.Name)
	}
	return fmt.Sprintf(`---
description: Fetch a ticket into branch context for %s
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
`, cfg.Project.Name)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

