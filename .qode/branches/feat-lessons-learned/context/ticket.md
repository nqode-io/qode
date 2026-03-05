# FEAT: Add lessons learned to knowledge base

Currently the user can write a file and this can be added to the knowledge base via `qode knowledge add [path]` terminal command.

Add new functionality to add lessons learned from

- Current session context summary
- Summary of files in branch or branches

The commands are only applicable to slash commands, not to terminal commands. This includes equivalents in all IDE's

`/qode-knowledge-add-context` should be used to make a summary of the current session context, then extract lessons learned from it and add them to the knowledge base. This command is only applicable to slash commands, not to terminal commands. This includes equivalents in all IDE's.

`qode-knowledge-add-branch [one or more branch names separated by commas]` should read all the relevant files and based on them make a summary, then extract lessons learned from it and add them to the knowledge base. This should work as a slash command or equivalent in all IDEs, as well as from the terminal.

In both cases previously present lessons learned and the ones extracted from either the context or branch(es) should all be distinct.

Each lesson learned should be in its own file. The name of the file should be the kebab case of the title followed by the Markdown extension, for example `this-is-an-example-title.md`. The content should look like this:

```
### This is an example title
Short description goes here in a form of one paragraph with no more than 100 words. It should be specific especially if only applicable under certain condition.

**Example 1:** Title explaining that the following should or must not be followed depending on the case
Some example of code, pattern to be aware of and how to handle it, common mistakes that are reoccurring and and that may lead to longer claude code sessions with more tokens spent, etc.

More examples may follow for both positive and negative examples. Examples may be omitted if the lesson learned does not require it. On average one or two examples should be sufficient. If more examples are required make them as brief as possible.
```

Make sure to revise the documentation. Ideally, after the current step 7 - Review + Quality gates, we should add an optional or recommended step to create lessons learned from current session context summary meaning running `/qode-knowledge-add-context` or equivalent.

Template prompt(s) should be added to support this.
