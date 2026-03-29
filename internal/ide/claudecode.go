package ide

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nqode/qode/internal/config"
)

// SetupClaudeCode generates Claude Code configuration files.
func SetupClaudeCode(root string, cfg *config.Config) error {
	commandsDir := filepath.Join(root, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return err
	}

	cmds := claudeSlashCommands(cfg)
	for name, content := range cmds {
		if err := writeFile(filepath.Join(commandsDir, name+".md"), content); err != nil {
			return err
		}
	}

	fmt.Printf("  Claude Code: %d slash commands\n", len(cmds))
	return nil
}

func claudeSlashCommands(cfg *config.Config) map[string]string {
	name := cfg.Project.Name
	return map[string]string{
		"qode-plan-refine": fmt.Sprintf(`# Refine Requirements — %s

**Worker pass:** Run this command and use its stdout output as your worker prompt:
  qode plan refine

Save the worker output to:
  .qode/branches/$(git branch --show-current)/refined-analysis.md

**Judge pass (scoring):** Run this command and use its stdout output as your prompt:
  qode plan judge

Then:
1. Parse the "**Total Score:** N/25" line from the judge output
2. Detect iteration number N from the "<!-- qode:iteration=N -->" header in refined-analysis.md (default: 1)
3. Rewrite refined-analysis.md replacing the first line with: <!-- qode:iteration=N score=S/25 -->
4. Write a copy to: .qode/branches/$(git branch --show-current)/refined-analysis-N-score-S.md
5. Report the score to the user. If S >= 25, suggest running /qode-plan-spec. Otherwise suggest re-running /qode-plan-refine.
`, name),

		"qode-plan-spec": fmt.Sprintf(`# Generate Technical Specification — %s


Run this command and use its stdout output as your prompt:
  qode plan spec

After generating the spec:
- Save it to: .qode/branches/$(git branch --show-current)/spec.md
- Suggest copying it to the ticket system for team review
`, name),

		"qode-review-code": fmt.Sprintf(`# Code Review — %s


Run this command and use its stdout output as your prompt:
  qode review code

After completing the review:
- Save to: .qode/branches/$(git branch --show-current)/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
`, name),

		"qode-review-security": fmt.Sprintf(`# Security Review — %s


Run this command and use its stdout output as your prompt:
  qode review security

After completing the review:
- Save to: .qode/branches/$(git branch --show-current)/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
`, name),

		"qode-start": fmt.Sprintf(`# Start Implementation — %s


Run this command and use its stdout output as your prompt:
  qode start

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


Run this command and use its stdout output as your prompt:
  qode knowledge add-branch $ARGUMENTS
`, name),
	}
}
