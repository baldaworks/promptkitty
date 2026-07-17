---
name: promptkitty-author-agent-instructions
description: Turn a fully assembled PromptKit prompt into ready-to-commit project instructions or a provider-native subagent profile. Use when the user asks to create persistent agent guidance, custom agent instructions, an agent profile, a subagent, or equivalent files for Codex, Claude Code, Grok Build, GitHub Copilot, OpenCode, or Cursor.
---

# PromptKitty Author Agent Instructions

Use `promptkitty` when installed. Otherwise use the pinned fallback `npx --yes @baldaworks/promptkitty@0.4.2` for every PromptKitty command. Use only the public CLI and remain offline after the package is available.

Read [references/provider-targets.md](references/provider-targets.md) completely before planning or writing provider files. Its current paths and schemas override stale platform paths in the pinned PromptKit template.

## Obtain source material

Accept a PromptKitty Assemble handoff containing final source Markdown and its resolved template metadata. If it is absent, load PromptKitty Assemble (`assemble` or `promptkitty-assemble`) in `source-only` mode and describe the reusable behavior the user wants. Do not accept unresolved template parameters.

The source Markdown is authoritative for persona identity, protocols, behaviors, and output expectations. Do not execute its task.

## Collect authoring choices

Resolve these values from the handoff or ask for them:

- output type: `instructions` for persistent project guidance or `agent` for a subagent profile
- providers: any subset of Codex, Claude Code, Grok Build, GitHub Copilot, OpenCode, and Cursor, including `All`
- lowercase kebab-case slug
- target project root, defaulting to the current repository only after confirmation
- optional project context, path scope, model, tools, or permissions

Do not invent optional model, tool, permission, or path-scope settings. Omit them unless the user supplies them.

## Assemble the authoring specification

Inspect `author-agent-instructions` and resolve every declared parameter:

```bash
promptkitty show "author-agent-instructions" --json
```

Set:

- `platform` to the selected provider names
- `output_type` to `instructions` or `agent`
- `base_persona` from the resolved source template persona
- `selected_protocols` from the resolved source components
- `behaviors` to the complete assembled source Markdown via `--param-file`
- `scope` to `project`
- `context` to the confirmed project context plus a statement that the assembled source is complete and external PromptKit files must not be read

Then assemble the specification:

```bash
promptkitty assemble "author-agent-instructions" \
  --param "platform=<providers>" \
  --param "output_type=<instructions-or-agent>" \
  --param "base_persona=<persona>" \
  --param "selected_protocols=<protocols>" \
  --param-file "behaviors=<source-prompt-path>" \
  --param "scope=project" \
  --param-file "context=<context-path>"
```

Use temporary multiline parameter files outside the target repository. Treat the assembled result as an authoring specification with two adaptations: use the supplied source instead of reading PromptKit component files, and use the provider reference for current paths and syntax.

## Preview and write

1. Produce a manifest of every proposed path, provider, purpose, and whether it is new, managed, or conflicting. Deduplicate shared targets.
2. Validate that every source behavior is represented, no `{{parameter}}` remains, and provider frontmatter or TOML is syntactically valid.
3. Show the manifest and concise diffs. Obtain explicit confirmation before any write.
4. For project instructions, add or replace only the matching PromptKitty managed block. Never replace unrelated content.
5. For an existing subagent file, require explicit overwrite confirmation after showing its diff. A same-path non-PromptKitty project file also requires confirmation before appending a managed block; otherwise choose another slug.
6. Write only beneath the confirmed target root, then report the paths and provider activation notes.

Do not implement application code, execute the source task, modify PromptKit components, commit, or publish as part of this skill.
