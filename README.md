# PromptKitty

[![Test](https://github.com/baldaworks/promptkitty/actions/workflows/test.yml/badge.svg)](https://github.com/baldaworks/promptkitty/actions/workflows/test.yml)
[![Lint](https://github.com/baldaworks/promptkitty/actions/workflows/lint.yml/badge.svg)](https://github.com/baldaworks/promptkitty/actions/workflows/lint.yml)
[![Security](https://github.com/baldaworks/promptkitty/actions/workflows/security.yml/badge.svg)](https://github.com/baldaworks/promptkitty/actions/workflows/security.yml)
[![Latest release](https://img.shields.io/github/v/release/baldaworks/promptkitty)](https://github.com/baldaworks/promptkitty/releases/latest)
[![npm version](https://img.shields.io/npm/v/%40baldaworks%2Fpromptkitty)](https://www.npmjs.com/package/@baldaworks/promptkitty)
[![Go Reference](https://pkg.go.dev/badge/github.com/baldaworks/promptkitty.svg)](https://pkg.go.dev/github.com/baldaworks/promptkitty)
[![License: MIT](https://img.shields.io/github/license/baldaworks/promptkitty)](LICENSE)

**PromptKit workflows for coding agents, the command line, and Go.**

Install PromptKitty as a plugin or native Agent Skills, then describe the engineering work you want done. Its skills discover the most relevant PromptKit template, collect every required input, run interactive intake when needed, and assemble a complete prompt. They can also turn that behavior into provider-native project instructions or a subagent profile.

The same embedded catalog is available through a standalone CLI and a deterministic Go library. PromptKitty performs runtime discovery and assembly without reading external component files or contacting a service.

The embedded snapshot is PromptKit `v0.6.1`: 15 personas, 56 protocols, 24 formats, 5 taxonomies, 71 templates, and 4 pipelines. Every declared parameter must be resolved before assembly succeeds.

## Agent skills

PromptKitty installs two complementary skills:

| Skill | What it does | Result |
| --- | --- | --- |
| **PromptKitty Assemble** | Turns a natural-language engineering task into a catalog search, template selection, required-parameter intake, and final PromptKit assembly. | A raw prompt, or a handoff for project instructions or a subagent profile. |
| **PromptKitty Author Agent Instructions** | Accepts assembled source — or asks Assemble to prepare it — and adapts the behavior to the selected agent host. | Ready-to-commit project instructions or a provider-native subagent profile. |

Assemble handles both single-shot and interactive templates. Single-shot templates offer **Raw prompt**, **Project instructions**, or **Subagent profile** as the result. Interactive templates perform their first safe questioning and confirmation phase, fold the answers into the declared parameters, and then produce the final assembled source without executing the later task.

Author Agent Instructions resolves the provider, output type, slug, and project root; validates the provider-native file; and previews a manifest and concise diff. It requires explicit confirmation before writing anything.

## Install for your agent

Run setup directly from npm; no global PromptKitty installation is required:

```bash
npx --yes @baldaworks/promptkitty@latest --version
npx --yes @baldaworks/promptkitty@latest setup codex
```

PromptKitty supports six agent hosts. Setup installs both skills:

| Host | Setup | Assemble | Author Agent Instructions |
| --- | --- | --- | --- |
| Codex | `npx --yes @baldaworks/promptkitty@latest setup codex` | `$promptkitty:assemble` | `$promptkitty:author-agent-instructions` |
| Claude Code | `npx --yes @baldaworks/promptkitty@latest setup claude` | `/promptkitty:assemble` | `/promptkitty:author-agent-instructions` |
| Grok Build | `npx --yes @baldaworks/promptkitty@latest setup grok` | `/promptkitty-assemble` | `/promptkitty-author-agent-instructions` |
| Copilot CLI | `npx --yes @baldaworks/promptkitty@latest setup copilot` | `/promptkitty-assemble` | `/promptkitty-author-agent-instructions` |
| OpenCode | `npx --yes @baldaworks/promptkitty@latest setup opencode` | `/promptkitty-assemble` | `/promptkitty-author-agent-instructions` |
| Cursor | `npx --yes @baldaworks/promptkitty@latest setup cursor` | `promptkitty-assemble` skill | `promptkitty-author-agent-instructions` skill |

Codex, Claude Code, Grok Build, and Copilot CLI setup use the repository's plugin marketplace. OpenCode setup writes both skills and matching commands under `.opencode/`; Cursor setup writes both skills under `.cursor/skills/`. Existing local files are preserved; use `--force` only when known PromptKitty assets should be replaced. Host CLIs and credentials remain external.

## Use PromptKitty from Codex

Ask Assemble for an engineering artifact or workflow instead of selecting and parameterizing a template yourself:

```text
$promptkitty:assemble Write a requirements document for deterministic offline prompt discovery.
```

The skill searches the embedded catalog, explains any materially different template choices, inspects the selected template, and asks only for required values that are not already available from the request or repository context. It then returns the raw assembled prompt or hands the resolved source to Author Agent Instructions when you choose a reusable result.

You can also request reusable behavior directly:

```text
$promptkitty:author-agent-instructions Create a spec-writing subagent for Codex in this repository.
```

When no assembled source was supplied, the authoring skill asks Assemble to prepare it in source-only mode. It then collects the Codex target and slug, shows the proposed path and diff, validates the TOML, and waits for explicit approval before creating the subagent file.

## Project instructions and subagents

Author Agent Instructions uses the pinned PromptKit `author-agent-instructions` template, with current provider paths supplied by PromptKitty:

| Host | Project instructions | Subagent profile |
| --- | --- | --- |
| Codex | `AGENTS.md` | `.codex/agents/<name>.toml` |
| Claude Code | `.claude/rules/<name>.md` | `.claude/agents/<name>.md` |
| Grok Build | `.grok/rules/<name>.md` | `.grok/agents/<name>.md` |
| Copilot CLI | `.github/instructions/*.instructions.md` | `.github/agents/<name>.agent.md` |
| OpenCode | `AGENTS.md` | `.opencode/agents/<name>.md` |
| Cursor | `.cursor/rules/<name>.mdc` | `.cursor/agents/<name>.md` |

Project guidance is maintained inside stable PromptKitty markers, so unrelated instructions remain untouched. Existing subagent profiles require an explicit overwrite confirmation. Codex and OpenCode share one managed `AGENTS.md` block when both are selected.

## CLI quick start

The agent skills drive this same public CLI internally. Use it directly for shell workflows, automation, or catalog exploration.

Install the native Go command:

```bash
go install github.com/baldaworks/promptkitty/cmd/promptkitty@v0.4.0
```

Search for a template using a natural-language task:

```bash
promptkitty search "write a requirements document" --type template
promptkitty show author-requirements-doc --json
```

Assemble the selected template after supplying every declared parameter:

```bash
promptkitty assemble author-requirements-doc \
  --param project_name=PromptKitty \
  --param description='Add deterministic offline prompt discovery' \
  --param context='Go library and CLI with an embedded PromptKit catalog' \
  --param audience='Go maintainers and coding agents'
```

`assemble` writes Markdown to stdout. Use `--output` only when a file is wanted, `--json` for the complete assembly result, and repeatable `--param-file`, `--protocol`, and `--taxonomy` flags for multiline or additional composition inputs.

## BM25 relevance search

`search` uses the in-memory BM25 index from [vecgo](https://github.com/hupe1980/vecgo). PromptKitty indexes component names, descriptions, remaining metadata, and complete Markdown bodies with weights `4 / 2 / 1 / 1`.

```bash
promptkitty search "review Go code" --type template --json
promptkitty search "root cause of a memory leak bug" --type template
promptkitty search "thread safety" --type protocol
```

Tokenization is Unicode-aware, common query terms are suppressed, filters retain global corpus scoring, and ties use stable catalog order. Scores stay internal so the existing `Component` JSON contract remains unchanged.

The installed command performs search and assembly offline. `npx` may contact npm to acquire the package when it is not already cached; the native package then uses only the embedded catalog.

## Command surface

```text
promptkitty list [--type ...] [--category ...] [--language ...] [--json]
promptkitty search <query> [--type ...] [--json]
promptkitty show <name> [--json]
promptkitty assemble <template> [composition flags]
promptkitty setup <codex|claude|grok|copilot|opencode|cursor> [--force]
```

Applications that already use Cobra can mount the same command tree:

```go
cmd := cli.NewCommand(cli.Options{Use: "promptkit"})
host.AddCommand(cmd)
```

The reusable `cli` package keeps successful output on stdout and diagnostics on stderr. The root package remains library-first and has no CLI or process dependencies.

## Go library

Install the module:

```bash
go get github.com/baldaworks/promptkitty@v0.4.0
```

Load the embedded catalog and assemble a fully parameterized prompt:

```go
library, err := promptkitty.New()
if err != nil {
    return err
}

result, err := library.Assemble(promptkitty.AssembleRequest{
    Template: "investigate-bug",
    Params: map[string]string{
        "problem_description": "Parser crashes on empty input",
        "code_context":        "src/parser.c",
        "environment":         "Linux amd64",
    },
})
if err != nil {
    return err
}

fmt.Println(result.Markdown)
```

Browse the same immutable catalog through Go:

```go
templates := library.List(promptkitty.Filter{Type: promptkitty.ComponentTemplate})
matches := library.Search("security review", promptkitty.Filter{Type: promptkitty.ComponentTemplate})
detail, err := library.Show("review-code")
pipelines := library.Pipelines()
```

`NewFromFS` loads a compatible private catalog for tests or applications. Runtime assembly remains one-pass, deterministic, and strict about unresolved parameters.

## npm distribution

Releases publish `@baldaworks/promptkitty` with statically linked native executables for macOS and Linux on amd64/arm64 and Windows on amd64. Omnidist builds every target with `CGO_ENABLED=0`. The npm launcher selects the matching binary, so users need neither Go nor CGO tooling.

## Updating the PromptKit snapshot

Change only the `ref` field in [`content/upstream.json`](content/upstream.json), then run:

```bash
go generate ./...
```

The generator resolves that ref through GitHub, downloads the archive, copies supported components and the upstream license, and rewrites the resolved commit and SHA-256 inventory. Review content, license, commit, and inventory changes together. Do not hand-edit pinned component files.

## About PromptKit

[Microsoft PromptKit](https://github.com/microsoft/PromptKit) is a composable prompt engineering library organized around personas, protocols, taxonomies, formats, templates, and pipelines. Its [bootstrap workflow](https://github.com/microsoft/PromptKit/blob/main/bootstrap.md) inspired PromptKitty's output chooser and agent-instruction authoring. PromptKitty adapts discovery, interactive intake, and parameter gathering to the stable `search`, `show`, and `assemble` CLI instead of reading or rewriting upstream files.

## License

PromptKitty's original Go code and documentation are available under the root [MIT License](LICENSE), copyright Alexey Samoylov.

The embedded Microsoft PromptKit content remains under Microsoft's MIT license and attribution. Its exact license copy is stored at [`third_party/promptkit/LICENSE`](third_party/promptkit/LICENSE); see [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for provenance and third-party terms.

PromptKitty uses vecgo's Apache-2.0-licensed BM25 implementation. The exact license is stored at [`third_party/vecgo/LICENSE`](third_party/vecgo/LICENSE) and included with statically linked npm artifacts.
