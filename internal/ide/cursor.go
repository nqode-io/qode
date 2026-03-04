package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
)

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(root string, cfg *config.Config) error {
	rulesDir := cfg.IDE.Cursor.RulesDir
	if rulesDir == "" {
		rulesDir = ".cursorrules"
	}
	commandsDir := cfg.IDE.Cursor.CommandsDir
	if commandsDir == "" {
		commandsDir = ".cursor/commands"
	}

	if err := os.MkdirAll(filepath.Join(root, rulesDir), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, commandsDir), 0755); err != nil {
		return err
	}

	// Workflow rule.
	if err := writeFile(filepath.Join(root, rulesDir, "qode-workflow.mdc"), workflowRule(cfg)); err != nil {
		return err
	}

	// Clean code rule.
	if err := writeFile(filepath.Join(root, rulesDir, "qode-clean-code.mdc"), cleanCodeRule(cfg)); err != nil {
		return err
	}

	// Slash commands.
	cmds := slashCommands(cfg)
	for name, content := range cmds {
		p := filepath.Join(root, commandsDir, name+".mdc")
		if err := writeFile(p, content); err != nil {
			return err
		}
	}

	fmt.Printf("  Cursor: %s/ (%d rules, %d commands)\n", rulesDir, 2, len(cmds))
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
3. Refine → /qode-plan-refine (iterate until 25/25)
4. Spec → /qode-plan-spec
5. Implement → /qode-start → code in Cursor
6. Review → /qode-review-code + /qode-review-security
7. Check → qode check
8. Ship → git commit && push

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
description: Refine requirements for %s (target 25/25)
---

First, run this command to generate the prompts:
  qode plan refine --prompt-only

**Worker pass:** Read and execute the worker prompt in:
  .qode/branches/$(git branch --show-current)/.refine-prompt.md

Save the worker output to:
  .qode/branches/$(git branch --show-current)/refined-analysis.md

**Judge pass (scoring):**
1. Read .qode/branches/$(git branch --show-current)/.refine-judge-prompt.md
2. Replace the placeholder line "[Worker output will be pasted here by the user after running the worker prompt above]" with the full content of refined-analysis.md
3. Execute the modified judge prompt
4. Parse the "**Total Score:** N/25" line from the judge output
5. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
6. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/25 -->
7. Write a copy to: .qode/branches/$(git branch --show-current)/refined-analysis-N-score-S.md
8. Report the score to the user. If S >= 25, suggest running "qode plan spec". Otherwise suggest re-running /qode-plan-refine.
`, cfg.Project.Name),

		"qode-plan-spec": fmt.Sprintf(`---
description: Generate technical specification for %s
---

First, run this command to generate the prompt:
  qode plan spec --prompt-only

Then read and execute the prompt in:
  .qode/branches/$(git branch --show-current)/.spec-prompt.md

After generating the spec, save it to:
  .qode/branches/$(git branch --show-current)/spec.md
`, cfg.Project.Name),

		"qode-review-code": fmt.Sprintf(`---
description: Code review for %s
---

First, run this command to generate the prompt:
  qode review code --prompt-only

Then read and execute the prompt in:
  .qode/branches/$(git branch --show-current)/.code-review-prompt.md

After completing the review, save it to:
  .qode/branches/$(git branch --show-current)/code-review.md
`, cfg.Project.Name),

		"qode-review-security": fmt.Sprintf(`---
description: Security review for %s
---

First, run this command to generate the prompt:
  qode review security --prompt-only

Then read and execute the prompt in:
  .qode/branches/$(git branch --show-current)/.security-review-prompt.md

After completing the review, save it to:
  .qode/branches/$(git branch --show-current)/security-review.md
`, cfg.Project.Name),

		"qode-start": fmt.Sprintf(`---
description: Start implementation session for %s
---

First, run this command to generate the prompt:
  qode start --prompt-only

Then read and execute the prompt in:
  .qode/branches/$(git branch --show-current)/.start-prompt.md

Execute the prompt as your implementation session.
`, cfg.Project.Name),

		"qode-ticket-fetch": fmt.Sprintf(`---
description: Fetch a ticket into branch context for %s
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
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

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
