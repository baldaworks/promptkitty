# Promptkitty agent guidance

- Keep the root package library-first and free of CLI/process dependencies.
- Preserve the pinned PromptKit component files byte-for-byte; update them only
  through the sync tool and review the generated inventory diff.
- Preserve PromptKit SPDX headers and third-party attribution.
- Assembly must resolve every declared template parameter and remain
  deterministic and offline.
- Run `go test ./...`, `go test -race ./...`, the configured linter, and
  `govulncheck` before completion.
