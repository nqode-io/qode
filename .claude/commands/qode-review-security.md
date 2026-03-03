# Security Review — qode

First, run this command to generate the prompt:
  qode review security --prompt-only

Then read and execute the prompt in:
  .qode/branches/$BRANCH/.security-review-prompt.md

Where $BRANCH is the current git branch name.

After completing the review:
- Save to: .qode/branches/$BRANCH/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
