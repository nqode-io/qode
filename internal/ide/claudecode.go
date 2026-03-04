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

	count := 0

	if cfg.IDE.ClaudeCode.ClaudeMD {
		if err := writeFile(filepath.Join(root, "CLAUDE.md"), buildClaudeMD(cfg)); err != nil {
			return err
		}
		count++
	}

	if cfg.IDE.ClaudeCode.SlashCommands {
		cmds := claudeSlashCommands(cfg)
		for name, content := range cmds {
			if err := writeFile(filepath.Join(commandsDir, name+".md"), content); err != nil {
				return err
			}
		}
		count += len(cmds)
	}

	fmt.Printf("  Claude Code: CLAUDE.md + %d slash commands\n", len(claudeSlashCommands(cfg)))
	return nil
}

func buildClaudeMD(cfg *config.Config) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s — Project Context\n\n", cfg.Project.Name))

	if cfg.Project.Description != "" {
		sb.WriteString(cfg.Project.Description + "\n\n")
	}

	sb.WriteString("## Tech Stack\n\n")
	for _, l := range cfg.Layers() {
		sb.WriteString(fmt.Sprintf("- **%s** (%s): `%s`\n", l.Name, l.Stack, l.Path))
	}
	sb.WriteString("\n")

	sb.WriteString("## Project Structure\n\n")
	sb.WriteString(fmt.Sprintf("Topology: %s\n\n", cfg.Project.Topology))
	for _, l := range cfg.Layers() {
		sb.WriteString(fmt.Sprintf("- `%s/` — %s (%s)\n", l.Path, l.Name, l.Stack))
	}
	sb.WriteString("\n")

	sb.WriteString("## Quality Standards\n\n")
	sb.WriteString(fmt.Sprintf("- Minimum code review score: %.1f/10\n", cfg.Review.MinCodeScore))
	sb.WriteString(fmt.Sprintf("- Minimum security review score: %.1f/10\n", cfg.Review.MinSecurityScore))
	sb.WriteString(fmt.Sprintf("- Max function length: %d lines\n", cfg.Architecture.CleanCode.MaxFunctionLines))
	sb.WriteString("\n")

	sb.WriteString("## Development Workflow\n\n")
	sb.WriteString("1. `qode branch create <name>` — Create feature branch\n")
	sb.WriteString("2. `qode ticket fetch <url>` — Fetch ticket context\n")
	sb.WriteString("3. `/qode-plan-refine` — Iterate requirements (target 25/25)\n")
	sb.WriteString("4. `/qode-plan-spec` — Generate tech spec\n")
	sb.WriteString("5. `qode start` — Generate implementation prompt\n")
	sb.WriteString("6. `/qode-review-code` + `/qode-review-security` — Reviews\n")
	sb.WriteString("7. `qode check` — All quality gates\n")
	sb.WriteString("8. `git commit && git push` — Ship\n\n")

	sb.WriteString("## Clean Code Rules\n\n")
	sb.WriteString("- Read existing code before writing new code\n")
	sb.WriteString("- Follow patterns in existing files — do not introduce new patterns\n")
	sb.WriteString("- Functions max 50 lines, single responsibility\n")
	sb.WriteString("- Handle all errors explicitly\n")
	sb.WriteString("- No magic numbers — use named constants\n")
	sb.WriteString("- No TODO comments in committed code\n\n")

	if cfg.TicketSystem.Type != "" && cfg.TicketSystem.Type != "manual" {
		sb.WriteString(fmt.Sprintf("## Ticket System\n\nType: %s\n", cfg.TicketSystem.Type))
		if cfg.TicketSystem.URL != "" {
			sb.WriteString(fmt.Sprintf("URL: %s\n", cfg.TicketSystem.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func claudeSlashCommands(cfg *config.Config) map[string]string {
	name := cfg.Project.Name
	return map[string]string{
		"qode-plan-refine": fmt.Sprintf(`# Refine Requirements — %s

First, run this command to generate the prompt:
  qode plan refine --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.refine-prompt.md

Where $BRANCH is the current git branch name.

After completing the analysis:
- Save the output to: .qode/branches/$BRANCH/refined-analysis.md
- Tell the user the analysis is complete and to run qode plan refine again for judge scoring
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

		"qode-ticket-fetch": `!qode ticket fetch $ARGUMENTS`,
	}
}

func init() {
	_ = os.Getenv // suppress unused import if needed
}
