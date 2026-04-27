---
name: "qode-knowledge-add-context"
description: "Extract durable lessons learned from the current qode context."
---

# Extract Lessons Learned — qode

Reflect on the current session and extract actionable lessons learned.

1. First, check existing lessons by running: qode knowledge list
2. Review the full conversation history of this session
3. Identify 1-5 actionable, specific lessons (not generic advice)
4. Write each lesson to its own file at .qode/knowledge/lessons/<kebab-case-title>.md

Each lesson file MUST follow this format:

```
### Title in sentence case
Short description (one paragraph, max 100 words). Be specific — state when
this lesson applies and what to do or avoid.

**Example 1:** What to do or avoid
Code snippet, pattern, or concrete example illustrating the lesson.
```

Rules:
- Do NOT duplicate existing lessons (check the list from step 1)
- Do NOT include credentials, API keys, or secrets
- Focus on: recurring mistakes, non-obvious patterns, project-specific conventions
- If no actionable lessons can be identified, inform the user
