# Security Review — qode

Read and execute the prompt in:
  .qode/branches/$BRANCH/.security-review-prompt.md

Where $BRANCH is the current git branch name.

If the file does not exist, tell the user to run: qode review security

After completing the review:
- Save to: .qode/branches/$BRANCH/security-review.md
- List all Critical and High vulnerabilities with OWASP categories
- Provide specific remediation for each issue
