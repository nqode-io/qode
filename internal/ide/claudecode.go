package ide

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nqode/qode/internal/config"
)

// SetupClaudeCode generates Claude Code configuration files.
func SetupClaudeCode(root string, cfg *config.Config) error {
	commandsDir := filepath.Join(root, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return err
	}

	if cfg.IDE.ClaudeCode.ClaudeMD {
		claudeMDPath := filepath.Join(root, "CLAUDE.md")
		if _, err := os.Stat(claudeMDPath); os.IsNotExist(err) {
			if err := writeFile(claudeMDPath, buildClaudeMD(cfg)); err != nil {
				return err
			}
		}
	}

	if cfg.IDE.ClaudeCode.SlashCommands {
		cmds := claudeSlashCommands(cfg)
		for name, content := range cmds {
			if err := writeFile(filepath.Join(commandsDir, name+".md"), content); err != nil {
				return err
			}
		}
	}

	fmt.Printf("  Claude Code: CLAUDE.md + %d slash commands\n", len(claudeSlashCommands(cfg)))
	return nil
}

func buildClaudeMD(cfg *config.Config) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — Project Context\n\n", cfg.Project.Name)

	if cfg.Project.Description != "" {
		sb.WriteString(cfg.Project.Description + "\n\n")
	}

	sb.WriteString("## Tech Stack\n\n")
	for _, l := range cfg.Layers() {
		fmt.Fprintf(&sb, "- **%s** (%s): `%s`\n", l.Name, l.Stack, l.Path)
	}
	sb.WriteString("\n")

	sb.WriteString("## Project Structure\n\n")
	fmt.Fprintf(&sb, "Topology: %s\n\n", cfg.Project.Topology)
	for _, l := range cfg.Layers() {
		fmt.Fprintf(&sb, "- `%s/` — %s (%s)\n", l.Path, l.Name, l.Stack)
	}
	sb.WriteString("\n")

	sb.WriteString("## Quality Standards\n\n")
	fmt.Fprintf(&sb, "- Minimum code review score: %.1f/10\n", cfg.Review.MinCodeScore)
	fmt.Fprintf(&sb, "- Minimum security review score: %.1f/10\n", cfg.Review.MinSecurityScore)
	fmt.Fprintf(&sb, "- Max function length: %d lines\n", cfg.Architecture.CleanCode.MaxFunctionLines)
	sb.WriteString("\n")

	sb.WriteString("## Development Workflow\n\n")
	sb.WriteString("1. `qode branch create <name>` — Create feature branch\n")
	sb.WriteString("2. `qode ticket fetch <url>` — Fetch ticket context\n")
	sb.WriteString("3. `/qode-plan-refine` — Iterate requirements (target 25/25)\n")
	sb.WriteString("4. `/qode-plan-spec` — Generate tech spec\n")
	sb.WriteString("5. `/qode-start` — Generate and run implementation prompt\n")
	sb.WriteString("6. `/qode-review-code` + `/qode-review-security` — Reviews\n")
	sb.WriteString("7. `/qode-knowledge-add-context` — (Recommended) Extract lessons learned\n")
	sb.WriteString("8. `qode check` — All quality gates\n")
	sb.WriteString("9. `git commit && git push` — Ship\n\n")

	sb.WriteString("## Clean Code Rules\n\n")
	sb.WriteString("- Read existing code before writing new code\n")
	sb.WriteString("- Follow patterns in existing files — do not introduce new patterns\n")
	sb.WriteString("- Functions max 50 lines, single responsibility\n")
	sb.WriteString("- Handle all errors explicitly\n")
	sb.WriteString("- No magic numbers — use named constants\n")
	sb.WriteString("- No TODO comments in committed code\n\n")

	if cfg.TicketSystem.Type != "" && cfg.TicketSystem.Type != "manual" {
		fmt.Fprintf(&sb, "## Ticket System\n\nType: %s\n", cfg.TicketSystem.Type)
		if cfg.TicketSystem.URL != "" {
			fmt.Fprintf(&sb, "URL: %s\n", cfg.TicketSystem.URL)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func claudeSlashCommands(cfg *config.Config) map[string]string {
	name := cfg.Project.Name
	return map[string]string{
		"qode-plan-refine": fmt.Sprintf(`# Refine Requirements — %s

First, run this command to generate the prompts:
  qode plan refine --prompt-only

Where $BRANCH is the current git branch name.

**Worker pass:** Read and execute the worker prompt in:
  .qode/branches/$BRANCH/.refine-prompt.md

Save the worker output to:
  .qode/branches/$BRANCH/refined-analysis.md

**Judge pass (scoring):**
1. Read .qode/branches/$BRANCH/.refine-judge-prompt.md
2. Replace the placeholder line "[Worker output will be pasted here by the user after running the worker prompt above]" with the full content of refined-analysis.md
3. Execute the modified judge prompt
4. Parse the "**Total Score:** N/25" line from the judge output
5. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
6. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/25 -->
7. Write a copy to: .qode/branches/$BRANCH/refined-analysis-N-score-S.md
8. Report the score to the user. If S >= 25, suggest running "qode plan spec". Otherwise suggest re-running /qode-plan-refine.
`, name),

		"qode-plan-spec": fmt.Sprintf(`# Generate Technical Specification — %s

First, run this command to generate the prompt:
  qode plan spec --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.spec-prompt.md

Where $BRANCH is the current git branch name.

After generating the spec:
- Save it to: .qode/branches/$BRANCH/spec.md
- Suggest copying it to the ticket system for team review
`, name),

		"qode-review-code": fmt.Sprintf(`# Code Review — %s

First, run this command to generate the prompt:
  qode review code --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.code-review-prompt.md

Where $BRANCH is the current git branch name.

After completing the review:
- Save to: .qode/branches/$BRANCH/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
`, name),

		"qode-review-security": fmt.Sprintf(`# Security Review — %s

First, run this command to generate the prompt:
  qode review security --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.security-review-prompt.md

Where $BRANCH is the current git branch name.

After completing the review:
- Save to: .qode/branches/$BRANCH/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
`, name),

		"qode-start": fmt.Sprintf(`# Start Implementation — %s

First, run this command to generate the prompt:
  qode start --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.start-prompt.md

Where $BRANCH is the current git branch name.

Execute the prompt as your implementation session.
`, name),

		"qode-ticket-fetch": `!qode ticket fetch $ARGUMENTS`,

		"qode-knowledge-add-context": fmt.Sprintf(`# Extract Lessons Learned — %s

Reflect on the current session and extract actionable lessons learned.

1. First, check existing lessons by running: qode knowledge list
2. Review the full conversation history of this session
3. Identify 1-5 actionable, specific lessons (not generic advice)
4. Write each lesson to its own file at .qode/knowledge/lessons/<kebab-case-title>.md

Each lesson file MUST follow this format:

`+"```"+`
### Title in sentence case
Short description (one paragraph, max 100 words). Be specific — state when
this lesson applies and what to do or avoid.

**Example 1:** What to do or avoid
Code snippet, pattern, or concrete example illustrating the lesson.
`+"```"+`

Rules:
- Do NOT duplicate existing lessons (check the list from step 1)
- Do NOT include credentials, API keys, or secrets
- Focus on: recurring mistakes, non-obvious patterns, project-specific conventions
- If no actionable lessons can be identified, inform the user
`, name),

		"qode-knowledge-add-branch": fmt.Sprintf(`# Extract Lessons from Branch — %s

First, run this command to generate the prompt:
  qode knowledge add-branch --prompt-only $ARGUMENTS

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.knowledge-add-branch-prompt.md

Where $BRANCH is the current git branch name.
`, name),
	}
}

func init() {
	_ = os.Getenv // suppress unused import if needed
}
