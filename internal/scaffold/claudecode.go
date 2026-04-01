package scaffold

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

	name := filepath.Base(root)
	cmds := claudeSlashCommands(name)
	for cmdName, content := range cmds {
		if err := writeFile(filepath.Join(commandsDir, cmdName+".md"), content); err != nil {
			return err
		}
	}

	fmt.Printf("  Claude Code: %d slash commands\n", len(cmds))
	return nil
}

// qodeCheckBody is the shared prompt body for both IDE generators.
// Only the header/frontmatter differs between Claude Code and Cursor.
const qodeCheckBody = `Run quality gates interactively in two sequential phases.

## Phase 1 — Unit tests

1. Inspect the project structure to determine the test runner:
   - go.mod present → run: go test ./...
   - package.json with a test script → run: npm test (or yarn test if yarn.lock is present)
   - pytest.ini / pyproject.toml with pytest config → run: pytest
   - For other stacks, apply equivalent discovery
   - If multiple layers are present, run tests for each layer
2. Run all unit tests. Do NOT read qode.yaml — determine the command from the project structure only.
3. **If any tests fail:**
   - Stop. Do not proceed to Phase 2.
   - Output a structured summary of all failures across all layers:
     - Which tests failed and why
     - Proposed fix for each failure
   - Ask the user how to proceed with exactly three options:
     - **Accept** — apply the proposed fixes and re-run the failing tests
     - **Stop** — exit without making any changes
     - **Comment** — let the user add notes or corrections before retrying
   - On Accept: apply fixes, re-run Phase 1 (do not advance to Phase 2 until all tests pass)
   - On Comment: incorporate the user's feedback before retrying; do not loop blindly
4. **If no test files are found:** skip Phase 1, note the skip, and proceed to Phase 2.
5. **If all tests pass:** proceed to Phase 2.

## Phase 2 — Linter

1. Inspect the project structure to determine the linter:
   - go.mod present and (.golangci.yml exists or golangci-lint is in PATH) → run: golangci-lint run
   - package.json with eslint in devDependencies or .eslintrc* file present → run: npx eslint .
   - ruff.toml or pyproject.toml with ruff config → run: ruff check .
   - For other stacks, apply equivalent discovery
   - If multiple layers are present, run the linter for each layer
2. Run the linter. Do NOT read qode.yaml — determine the command from the project structure only.
3. **If linting issues are found:**
   - Stop.
   - Output a structured summary of all violations across all layers:
     - Which rules were violated and where
     - Proposed fix for each violation
   - Ask the user how to proceed with exactly three options: **Accept / Stop / Comment**
   - On Accept: apply fixes, re-run Phase 2 (do not advance until lint is clean)
   - On Comment: incorporate the user's feedback before retrying; do not loop blindly
4. **If no linter is found:** skip Phase 2, report success.
5. **If lint is clean:** report that all quality gates passed and suggest running /qode-review-code.

## Unknown stack

If neither a test runner nor a linter can be determined for a given layer, report what was inspected and ask the user to specify the command. Do not guess.
`

func claudeSlashCommands(name string) map[string]string {
	return map[string]string{
		"qode-plan-refine": fmt.Sprintf(`# Refine Requirements — %s

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

		"qode-plan-spec": fmt.Sprintf(`# Generate Technical Specification — %s

Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode plan spec --force`"+` to bypass score gates when needed.

After generating the spec:
- Save it to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/spec.md
- Suggest copying it to the ticket system for team review
`, name),

		"qode-review-code": fmt.Sprintf(`# Code Review — %s

Run this command and use its stdout output as your prompt:
  qode review code

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review code --force`"+` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
`, name),

		"qode-review-security": fmt.Sprintf(`# Security Review — %s

Run this command and use its stdout output as your prompt:
  qode review security

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use `+"`qode review security --force`"+` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
`, name),

		"qode-check": fmt.Sprintf("# Quality Gates — %s\n\n", name) + qodeCheckBody,

		"qode-start": fmt.Sprintf(`# Start Implementation — %s

Run this command and use its stdout output as your prompt:
  qode start

If the output begins with `+"`STOP.`"+`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `+"`qode start --force`"+` to bypass the spec prerequisite when needed.

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
