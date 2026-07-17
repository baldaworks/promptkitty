---
name: promptkitty-assemble
description: Discover and assemble a task-specific prompt from the embedded PromptKit catalog through the PromptKitty CLI. Use when the user wants to select a PromptKit template, compose a prompt for an engineering task, run an interactive PromptKit intake, write a requirements or design artifact, investigate a bug, review code, inspect templates, or prepare source material for persistent agent instructions.
---

# PromptKitty Assemble

Use `promptkitty` when installed. Otherwise use the pinned fallback `npx --yes @baldaworks/promptkitty@0.4.0` for every command in the task. Resolve this once and keep the same command form throughout the workflow.

Use only the public PromptKitty CLI. Do not read PromptKit component source files or fetch PromptKit from the network.

## Discover and inspect

1. Reduce the task to a short intent query and search templates:

   ```bash
   promptkitty search "<intent>" --type template --json
   ```

   Retry with fewer domain-specific words when needed. Use `promptkitty list --type template --json` only when search cannot identify a candidate.

2. Select the best template from its name, description, and category. If two candidates imply materially different workflows, explain the difference briefly and ask the user to choose.

3. Inspect the selected template:

   ```bash
   promptkitty show "<template>" --json
   ```

   Treat every key in `metadata.params` as required. Reuse values supplied by the user or available in task context. Ask for missing values; never invent them. An empty value is valid only when the user intentionally leaves that parameter empty.

## Assemble the source

Use `--param "<name>=<value>"` for short values and `--param-file "<name>=<path>"` for files or long multiline input. Quote assignments so shell metacharacters remain data. Add composition flags only when the user requests or confirms them.

```bash
promptkitty assemble "<template>" \
  --param "<name>=<value>" \
  --param-file "<name>=<path>"
```

### Interactive templates

When `metadata.mode` is `interactive`:

1. Resolve every declared parameter and run a provisional assembly without writing it to the target project.
2. Read the provisional prompt only far enough to identify its first questioning, validation, and user-confirmation gate.
3. Perform that first phase with the user. Read-only inspection is allowed when already authorized and necessary to answer the phase. Do not edit files, generate deliverables, run side-effectful commands, commit, or continue past the first confirmation gate.
4. Fold confirmed answers into the matching declared parameters. Append a concise preflight decision summary to `context` when that parameter exists; otherwise refine only semantically matching declared parameters. Never add undeclared parameters.
5. Re-run `assemble` with the enriched values. This is the final source prompt. Do not execute its later phases.

## Choose the result

Treat a missing `metadata.mode` as single-shot. For a single-shot template, ask which result the user wants:

1. **Raw prompt** (default) — return the final assembled stdout unchanged.
2. **Project instructions** — hand off to PromptKitty Author Agent Instructions with output type `instructions`.
3. **Subagent profile** — hand off with output type `agent`.

Do not offer this chooser automatically for an interactive template. Honor an explicit request to turn an interactive result into project instructions or a subagent profile after its preflight and final assembly.

For raw output, use stdout unless the user asks for `--output`. Never add `--force` without explicit permission. Do not wrap the prompt, add session headers, create role files, or execute it.

## Authoring handoff

Load the installed PromptKitty Author Agent Instructions skill: `author-agent-instructions` in namespaced plugin hosts or `promptkitty-author-agent-instructions` in prefixed hosts. Pass this handoff in the current session:

- output type: `instructions` or `agent`
- selected template name and `metadata.mode`
- resolved template metadata, including persona and protocols
- resolved parameter values and confirmed composition overrides
- final assembled source Markdown
- provider, slug, and target root when already supplied

When Author Agent Instructions requests `source-only`, skip the result chooser and return this handoff to that skill. Do not print the source as the final user response.

If PromptKitty reports unresolved parameters, inspect the template again, collect the missing values, and retry. Do not bypass validation.
