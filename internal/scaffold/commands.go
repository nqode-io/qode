package scaffold

// CommandDef holds the IDE-agnostic definition of a slash command.
// Name is the command key (e.g. "qode-plan-refine").
// Title is used in the Claude Code heading: "# Title — <project>".
// Description is the Cursor frontmatter description; it contains %s for the project name.
// Body is the shared body text rendered after the IDE-specific header.
type CommandDef struct {
	Name        string
	Title       string
	Description string
	Body        string
}

const planRefineBody = `**Worker pass:** Run this command and use its stdout output as your worker prompt:
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
`

const planSpecBody = `Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with ` + "`STOP.`" + `, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use ` + "`qode plan spec --force`" + ` to bypass score gates when needed.

After generating the spec:
- Save it to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/spec.md
- Suggest copying it to the ticket system for team review
`

const reviewCodeBody = `Run this command and use its stdout output as your prompt:
  qode review code

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use ` + "`qode review code --force`" + ` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/code-review.md
- List all Critical and High issues clearly
- Provide specific, actionable fix suggestions
`

const reviewSecurityBody = `Run this command and use its stdout output as your prompt:
  qode review security

If the command produces no output (no uncommitted changes), inform the user to commit changes first. Use ` + "`qode review security --force`" + ` to bypass the uncommitted-diff check.

After completing the review:
- Save to: .qode/branches/$(git branch --show-current | sed 's|/|--|g')/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
`

const startBody = `Run this command and use its stdout output as your prompt:
  qode start

If the output begins with ` + "`STOP.`" + `, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use ` + "`qode start --force`" + ` to bypass the spec prerequisite when needed.

Execute the prompt as your implementation session.
`

const knowledgeAddContextBody = `Reflect on the current session and extract actionable lessons learned.

1. First, check existing lessons by running: qode knowledge list
2. Review the full conversation history of this session
3. Identify 1-5 actionable, specific lessons (not generic advice)
4. Write each lesson to its own file at .qode/knowledge/lessons/<kebab-case-title>.md

Each lesson file MUST follow this format:

` + "```" + `
### Title in sentence case
Short description (one paragraph, max 100 words). Be specific — state when
this lesson applies and what to do or avoid.

**Example 1:** What to do or avoid
Code snippet, pattern, or concrete example illustrating the lesson.
` + "```" + `

Rules:
- Do NOT duplicate existing lessons (check the list from step 1)
- Do NOT include credentials, API keys, or secrets
- Focus on: recurring mistakes, non-obvious patterns, project-specific conventions
- If no actionable lessons can be identified, inform the user
`

const knowledgeAddBranchBody = `Run this command and use its stdout output as your prompt:
  qode knowledge add-branch $ARGUMENTS
`

// allCommands is the single source of truth for all qode slash commands.
// Both claudeSlashCommands and cursorSlashCommands iterate over this slice
// and apply their respective IDE-specific header format.
var allCommands = []CommandDef{
	{
		Name:        "qode-plan-refine",
		Title:       "Refine Requirements",
		Description: "Refine requirements for %s",
		Body:        planRefineBody,
	},
	{
		Name:        "qode-plan-spec",
		Title:       "Generate Technical Specification",
		Description: "Generate technical specification for %s",
		Body:        planSpecBody,
	},
	{
		Name:        "qode-review-code",
		Title:       "Code Review",
		Description: "Code review for %s",
		Body:        reviewCodeBody,
	},
	{
		Name:        "qode-review-security",
		Title:       "Security Review",
		Description: "Security review for %s",
		Body:        reviewSecurityBody,
	},
	{
		Name:        "qode-check",
		Title:       "Quality Gates",
		Description: "Run quality gates for %s",
		Body:        qodeCheckBody,
	},
	{
		Name:        "qode-start",
		Title:       "Start Implementation",
		Description: "Start implementation session for %s",
		Body:        startBody,
	},
	{
		Name:        "qode-ticket-fetch",
		Title:       "Fetch Ticket via MCP",
		Description: "Fetch a ticket into branch context for %s",
		Body:        ticketFetchMCPBody,
	},
	{
		Name:        "qode-knowledge-add-context",
		Title:       "Extract Lessons Learned",
		Description: "Extract lessons learned from current session for %s",
		Body:        knowledgeAddContextBody,
	},
	{
		Name:        "qode-knowledge-add-branch",
		Title:       "Extract Lessons from Branch",
		Description: "Extract lessons learned from branch context for %s",
		Body:        knowledgeAddBranchBody,
	},
}
