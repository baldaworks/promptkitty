<div align="center">

# PromptKitty

**PromptKit, packaged for Go.**

Browse, compose, and assemble a pinned Microsoft PromptKit catalog from a
standalone command or a deterministic, offline Go library.

[![Go Reference](https://pkg.go.dev/badge/github.com/baldaworks/promptkitty.svg)](https://pkg.go.dev/github.com/baldaworks/promptkitty)
[![Test](https://github.com/baldaworks/promptkitty/actions/workflows/test.yml/badge.svg)](https://github.com/baldaworks/promptkitty/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[Command](#command) ·
[Library](#library) ·
[Original PromptKit](https://github.com/microsoft/PromptKit) ·
[License](#license)

</div>

PromptKitty embeds Microsoft PromptKit `v0.6.1`: personas, protocols, formats,
taxonomies, templates, and pipelines. Runtime use reads no external files,
makes no network calls, and resolves every declared template parameter before
rendering.

## Why PromptKitty?

- **Two integration surfaces** — use the standalone command, mount its reusable
  Cobra command tree, or call the root Go package directly.
- **Deterministic assembly** — the same catalog and inputs produce the same
  prompt.
- **Offline runtime** — every PromptKit component is compiled into the binary.
- **Strict parameters** — missing inputs are errors; empty strings remain
  explicit values.
- **Verifiable snapshot** — the upstream ref, resolved commit, license, and
  SHA-256 inventory are pinned in [`content/upstream.json`](content/upstream.json).

## Command

Install the standalone command:

```bash
go install github.com/baldaworks/promptkitty/cmd/promptkitty@v0.2.1
```

Browse the catalog and render prompts from a shell or automation:

```bash
promptkitty list --type template
promptkitty search security --type template
promptkitty show review-code --json
promptkitty assemble review-code \
  --param code='package main' \
  --param review_focus=correctness \
  --param language=Go \
  --param additional_protocols= \
  --param context='small example'
```

`assemble` writes Markdown to stdout by default. Use `--output` to write a
file, `--json` for the complete assembly result, and the repeatable `--param`,
`--param-file`, `--protocol`, and `--taxonomy` flags for composition inputs.

Host applications can mount the same command tree under their own Cobra root:

```go
cmd := cli.NewCommand(cli.Options{Use: "promptkit"})
host.AddCommand(cmd)
```

The reusable `cli` package keeps successful output on stdout and diagnostics on
stderr, matching the standalone command.

## Library

Install the Go module:

```bash
go get github.com/baldaworks/promptkitty@v0.2.1
```

Create the embedded library and assemble a fully parameterized prompt:

```go
func assemble() error {
    library, err := promptkitty.New()
    if err != nil {
        return fmt.Errorf("load PromptKit catalog: %w", err)
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
        return fmt.Errorf("assemble investigate-bug prompt: %w", err)
    }

    fmt.Println(result.Markdown)

    return nil
}
```

Parameter substitution is one-pass, so mustache syntax supplied as input stays
data instead of becoming another template expression.

### Browse the catalog

```go
templates := library.List(promptkitty.Filter{
    Type: promptkitty.ComponentTemplate,
})
security := library.Search("security", promptkitty.Filter{})
review, err := library.Show("review-code")
pipelines := library.Pipelines()
```

Catalog results are returned in stable type, category, and name order. Use
`NewFromFS` when tests or applications need a compatible private catalog
instead of the embedded snapshot.

### Compose semantic layers

PromptKitty follows PromptKit's composition order:

```text
persona → protocols → taxonomies → format → template
```

Use `Persona`, `AdditionalProtocols`, `AdditionalTaxonomies`, and `Format` on
`AssembleRequest` to replace configurable components or extend their defaults.

## Updating the PromptKit snapshot

Change only the `ref` field in [`content/upstream.json`](content/upstream.json)
to the desired immutable PromptKit tag, then run:

```bash
go generate ./...
```

The generator resolves the ref through GitHub, downloads its archive, copies
the supported components and upstream license, then rewrites the resolved
commit and SHA-256 inventory. Review the content, license, and lock diff
together. Runtime builds never contact GitHub.

## About PromptKit

[Microsoft PromptKit](https://github.com/microsoft/PromptKit) is a composable
prompt engineering library organized around reusable personas, protocols,
taxonomies, formats, templates, and pipelines. PromptKitty embeds the pinned
upstream documents for Go consumers without reinterpreting their content.

## License

PromptKitty's original Go code and documentation are available under the root
[MIT License](LICENSE), copyright Alexey Samoylov.

The embedded Microsoft PromptKit content remains under Microsoft's MIT license
and attribution. Its exact license copy is stored at
[`third_party/promptkit/LICENSE`](third_party/promptkit/LICENSE); see
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for third-party terms and
provenance.
