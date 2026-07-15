# PromptKitty

Embedded PromptKit catalog and deterministic prompt assembler for Go.

PromptKitty packages a pinned component snapshot from Microsoft PromptKit into
a Go module. It loads no files and makes no network calls at runtime.
Applications can browse personas, protocols, formats, taxonomies, templates,
and pipelines, then assemble a fully parameterized prompt. The exact upstream
ref, resolved commit, and SHA-256 inventory live in `content/upstream.json`.

## Install

Install the library or standalone CLI:

```bash
go get github.com/baldaworks/promptkitty@v0.2.1
go install github.com/baldaworks/promptkitty/cmd/promptkitty@v0.2.1
```

## Assemble a prompt

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

Every parameter declared by the selected template must be present. An empty
string is an explicit value. Parameter substitution is one-pass and does not
reinterpret mustache syntax supplied as input data.

Composition follows PromptKit's semantic layers:

```text
persona → protocols → taxonomies → format → template
```

Use `Persona`, `AdditionalProtocols`, `AdditionalTaxonomies`, and `Format` on
`AssembleRequest` to resolve configurable templates or extend their defaults.

## Browse the catalog

```go
templates := library.List(promptkitty.Filter{Type: promptkitty.ComponentTemplate})
security := library.Search("security", promptkitty.Filter{})
review, err := library.Show("review-code")
pipelines := library.Pipelines()
```

The reusable `cli` package exposes the same command tree for host applications:

```go
cmd := cli.NewCommand(cli.Options{Use: "promptkit"})
host.AddCommand(cmd)
```

The standalone command supports `list`, `search`, `show`, and `assemble`:

```bash
promptkitty list
promptkitty search security --type template
promptkitty show review-code --json
promptkitty assemble review-code \
  --param code='package main' \
  --param review_focus=correctness \
  --param language=Go \
  --param additional_protocols= \
  --param context='small example'
```

`assemble` writes rendered Markdown to stdout by default. Use `--output` to
write a file or `--json` to receive the complete assembly result.

## Updating PromptKit content

The embedded snapshot is pinned by ref, resolved commit, and SHA-256 inventory
in `content/upstream.json`. To update it, change only the `ref` field to the
desired immutable PromptKit tag, then run:

```bash
go generate ./...
```

The generator resolves that ref through GitHub, downloads its archive, copies
the supported Markdown components, manifest, and upstream `LICENSE`, then
rewrites the resolved commit and all SHA-256 checksums. Review the content,
license, and lock diff together. Runtime builds never contact GitHub.

## License

PromptKitty's original Go code and documentation are distributed under the root
MIT `LICENSE`, copyright Alexey Samoylov. The embedded Microsoft PromptKit
content remains under Microsoft's MIT license and attribution; its exact
license copy is refreshed by `go generate` and stored at
`third_party/promptkit/LICENSE`. See `THIRD_PARTY_NOTICES.md`.
