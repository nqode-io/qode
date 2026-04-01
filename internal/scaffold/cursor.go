package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

const cursorCommandsDir = ".cursor/commands"

// SetupCursor generates Cursor IDE configuration files.
func SetupCursor(root string) error {
	if err := os.MkdirAll(filepath.Join(root, cursorCommandsDir), 0755); err != nil {
		return err
	}

	name := filepath.Base(root)
	cmds := slashCommands(name)
	for cmdName, content := range cmds {
		p := filepath.Join(root, cursorCommandsDir, cmdName+".mdc")
		if err := writeFile(p, content); err != nil {
			return err
		}
	}

	fmt.Printf("  Cursor: .cursor/commands/ (%d commands)\n", len(cmds))
	return nil
}

func slashCommands(name string) map[string]string {
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
`, name),

		"qode-plan-spec": fmt.Sprintf(`---
description: Generate technical specification for %s
---

Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode plan spec --force`"+` to bypass score gates when needed.

After generating the spec, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/spec.md
`, name),

		"qode-review-code": fmt.Sprintf(`---
description: Code review for %s
---

Run this command and use its stdout output as your prompt:
  qode review code

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review code --force`"+` to bypass the uncommitted-diff check.

After completing the review, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/code-review.md
`, name),

		"qode-review-security": fmt.Sprintf(`---
description: Security review for %s
---

Run this command and use its stdout output as your prompt:
  qode review security

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review security --force`"+` to bypass the uncommitted-diff check.

After completing the review, save it to:
  .qode/branches/$(git branch --show-current | sed 's|/|--|g')/security-review.md
`, name),

		"qode-check": fmt.Sprintf("---\ndescription: Run quality gates for %s\n---\n\n", name) + qodeCheckBody,

		"qode-start": fmt.Sprintf(`---
description: Start implementation session for %s
---

Run this command and use its stdout output as your prompt:
  qode start

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode start --force`"+` to bypass the spec prerequisite when needed.

Execute the prompt as your implementation session.
`, name),

		"qode-ticket-fetch": fmt.Sprintf(`---
description: Fetch a ticket into branch context for %s
---

Run the following command with the ticket URL provided after the slash command:
  qode ticket fetch $ARGUMENTS
`, name),

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
`, name),

		"qode-knowledge-add-branch": fmt.Sprintf(`---
description: Extract lessons learned from branch context for %s
---

Run this command and use its stdout output as your prompt:
  qode knowledge add-branch $ARGUMENTS
`, name),
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
