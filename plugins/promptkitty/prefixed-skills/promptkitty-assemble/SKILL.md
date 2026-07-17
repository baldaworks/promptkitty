---
name: promptkitty-assemble
description: Discover and assemble a task-specific prompt from the embedded PromptKit catalog through the PromptKitty CLI. Use when the user wants to select a PromptKit template, compose a prompt for an engineering task, write a requirements or design artifact, investigate a bug, review code, or inspect available PromptKit templates.
---

# PromptKitty Assemble

Use `promptkitty` when it is installed. Otherwise use the pinned fallback `npx --yes @baldaworks/promptkitty@0.3.0` for every command in the task. Resolve this once and use the same command form throughout the workflow.

Use only the public PromptKitty CLI. Do not read component source files or fetch PromptKit from the network.

## Workflow

1. Reduce the user's task to a short intent query and search templates:

   ```bash
   promptkitty search "<intent>" --type template --json
   ```

   If no useful result appears, retry with fewer domain-specific words. Use `promptkitty list --type template --json` only when search cannot identify a candidate.

2. Select the best template from its name, description, and category. If two candidates imply materially different workflows, explain the difference briefly and ask the user to choose.

3. Inspect the selected template:

   ```bash
   promptkitty show "<template>" --json
   ```

   Treat every key in `metadata.params` as required. Reuse values already supplied by the user or available in the task context. Ask only for missing values; never invent them. An empty value is valid only when the user intentionally leaves that parameter empty.

4. Assemble with one `--param "<name>=<value>"` per short value. Use `--param-file "<name>=<path>"` for existing files or long multiline inputs. Quote assignments so spaces and shell metacharacters remain data.

   ```bash
   promptkitty assemble "<template>" \
     --param "<name>=<value>" \
     --param-file "<name>=<path>"
   ```

   Add `--persona`, `--protocol`, `--taxonomy`, `--format`, or `--no-format` only when the user requests or confirms that composition change.

5. Keep Markdown on stdout by default. Use `--output "<path>"` only when the user asks for a file. Never add `--force` without explicit permission to replace an existing file.

6. Return the assembled stdout unchanged. Do not wrap it in another document, add session headers, create role files, or execute the assembled prompt in the current session. Interactive PromptKit templates are also assembled and returned, not executed.

If the CLI reports unresolved parameters, inspect the template again, collect the missing values, and retry. Do not bypass PromptKitty's validation.
