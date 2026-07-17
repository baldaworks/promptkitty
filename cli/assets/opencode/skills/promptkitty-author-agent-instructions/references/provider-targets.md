# Provider targets

Use lowercase kebab-case slugs. Keep generated instructions concise, imperative, self-contained, and ready to commit. Omit optional model, tool, and permission fields unless the user requests them.

## Managed project instructions

Wrap generated Markdown body content with stable markers:

```markdown
<!-- promptkitty:<slug>:begin -->
<generated instructions>
<!-- promptkitty:<slug>:end -->
```

On regeneration, replace only the matching block. Keep required YAML frontmatter outside the block. When an existing target has no matching block, show a diff and obtain confirmation before appending; never replace unrelated content.

| Provider | Target | Required shape |
|---|---|---|
| Codex | `AGENTS.md` | Plain Markdown managed block. |
| Claude Code | `.claude/rules/<slug>.md` | Plain Markdown managed block without `paths` frontmatter for project-wide scope. Add `paths` only when supplied by the user. |
| Grok Build | `.grok/rules/<slug>.md` | Plain Markdown managed block. |
| GitHub Copilot | `.github/instructions/<slug-or-concern>.instructions.md` | YAML `description` and `applyTo`; use `applyTo: '**'` unless the user supplies a narrower scope. The authoring specification may decompose independent concerns into multiple files. |
| OpenCode | `AGENTS.md` | Plain Markdown managed block shared with Codex. When both are selected, create one block and list both consumers in the manifest. |
| Cursor | `.cursor/rules/<slug>.mdc` | YAML `alwaysApply: true` for project-wide scope. Use `description` plus `alwaysApply: false` only when the user requests relevance-based activation; use `globs` only for an explicit path scope. |

## Subagent profiles

Create one standalone profile per selected provider. The instruction body comes from the condensed assembled source. Existing files require preview and explicit overwrite approval.

| Provider | Target | Minimum required shape |
|---|---|---|
| Codex | `.codex/agents/<slug>.toml` | TOML strings `name`, `description`, and `developer_instructions`. Encode multiline content as valid TOML and validate it before writing. |
| Claude Code | `.claude/agents/<slug>.md` | YAML `name` and `description`, followed by the instruction body. |
| Grok Build | `.grok/agents/<slug>.md` | YAML `name` and `description`, followed by the instruction body. Grok also understands Claude-compatible agent fields, but omit them by default. |
| GitHub Copilot | `.github/agents/<slug>.agent.md` | YAML `name`, `description`, and `user-invocable: false`, followed by the instruction body. |
| OpenCode | `.opencode/agents/<slug>.md` | YAML `description` and `mode: subagent`, followed by the instruction body; the filename supplies the agent name. |
| Cursor | `.cursor/agents/<slug>.md` | YAML `name` and `description`, followed by the instruction body. |

## Verification

- Resolve and normalize every target beneath the confirmed project root.
- Deduplicate `AGENTS.md` when Codex and OpenCode are both selected.
- Preserve YAML frontmatter while updating managed blocks.
- Reject unresolved Mustache parameters.
- Ensure the provider file contains no PromptKit assembly headers or instructions to read PromptKit source files.
- Mention how the selected host discovers or invokes each resulting file.
