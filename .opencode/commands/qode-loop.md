---
description: Run the serial ticket loop for qode (repo-internal)
---

Run the serial ticket loop: one GitHub issue at a time through the full qode workflow
(refine → spec → implement → check → reviews → PR → operator merge), per the driver protocol.
This command is contributor tooling for the qode repository itself; `qode init` does not
generate it and it is not part of the qode product.

1. Read [docs/qode-loop.md](../../docs/qode-loop.md) in full — it is the driver and the
   authority on stages, model assignments, gates, caps, dogfooding rules, and scheduling.
2. Execute one iteration with this prompt, verbatim (append ` Ticket sequence override: $ARGUMENTS`
   only when $ARGUMENTS is non-empty):

   `Execute one iteration of the qode ticket loop per docs/qode-loop.md — read it fully, then act.`

   OpenCode has no self-paced loop skill, so run one iteration per invocation: re-invoke
   `/qode-loop` for each further iteration.
3. Follow the driver from there. Do not start a second ticket while one is in flight; do not
   bypass a gate with --force; surface owner-level decisions and stall-cap pauses using the
   `question` tool; answer the in-command prompts per the driver's policy table.
