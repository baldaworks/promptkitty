# Promptkitty

Embedded PromptKit catalog and deterministic prompt assembler for Go.

Promptkitty packages the complete component library from Microsoft PromptKit
`v0.6.1` into a Go module. It loads no files and makes no network calls at
runtime. Applications can browse personas, protocols, formats, taxonomies,
templates, and pipelines, then assemble a fully parameterized prompt.

## Install

Install the library or standalone CLI:

```bash
go get github.com/baldaworks/promptkitty@v0.2.0
go install github.com/baldaworks/promptkitty/cmd/promptkitty@v0.2.0
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

The embedded snapshot is pinned by commit and SHA-256 inventory in
`content/upstream.json`. Maintainers update the ref and commit, then run:

```bash
go run ./internal/tools/syncpromptkit \
  -lock content/upstream.json \
  -dest content/promptkit \
  -refresh-lock
go generate ./...
```

Review component and inventory changes together. Runtime builds never contact
GitHub.

## License

Promptkitty is MIT licensed. Embedded PromptKit content remains under its
original MIT license and attribution; see `THIRD_PARTY_NOTICES.md`.
