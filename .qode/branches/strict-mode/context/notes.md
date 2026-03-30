# Notes

## Initial comments

- There is no need to add guards to `knowledge_cmd.go`. These are optional steps and capturing knowledge could be done at any time. Same goes for `/qode-check`. Only affected commands should be the ones that produce prompts from templates and are explicitly used in slash commands by either cursor or claude.
- The `qode plan status` subcommand has been removed. Ideally the `qode workflow` will not print the workflow with those borders, but step by step process with numbered/ordered bullet points. A new `qode workflow status` subcommand will show all the steps which were previously executed with info, and the first next step, for example:

```
1. Create branch - Completed.
2. Add context - Completed.
3. Refine requirements - 4 iterations, latest score: 25/25.
4. Generate spec - Completed.
5. Implement - Completed.
6. Test locally - Always done by the user.
7. Quality gates - Always done by the user.
8. Review - Code review passed with score: 11/12.

Up next: Complete review step by running `/qode-review-security`.
```

If a score is less than the minimum for a particular step, Up next should say something like `Consider fixing the issues and re-running the step.` or similar as applicable.

Test locally and Quality gates are always manual steps, so is the capturing of the lessons learned, ship and cleanup.

- Ensure all documentation in `.md` files and inline documentation in `.go` files along with comments, and propmt templates is up to date.
- If the guard fails, instead of emmiting the prompt we should emmit the instruction to AI to stop and inform the user that one of the previous steps has not been completed or the score does not match the minimal score. This should affect the print to stdout only, the `--to-file` should keep the existing behavior as this is used for debugging the prompt templates if the user overrides them.