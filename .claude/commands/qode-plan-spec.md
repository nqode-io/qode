# Generate Technical Specification — qode

Run this command and use its stdout output as your prompt:
  qode plan spec

If the output begins with `STOP.`, do not execute it as a prompt — report the prerequisite message to the user and wait for instructions. Use `qode plan spec --force` to bypass score gates when needed.

After generating the spec:
- Save it to: .qode/contexts/current/spec.md
- Suggest copying it to the ticket system for team review
