# Security Review — qode

Where $BRANCH is the current git branch name.

Run this command and use its stdout output as your prompt:
  qode review security

After completing the review:
- Save to: .qode/branches/$BRANCH/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
