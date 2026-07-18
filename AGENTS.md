# PromptKitty agent guidance

- Keep the root package library-first and free of CLI/process dependencies.
- Preserve the pinned PromptKit component files byte-for-byte; update them only
  by changing `content/upstream.json`'s `ref`, running `go generate ./...`, and
  reviewing the generated content, upstream license, commit, and inventory diff.
- Preserve PromptKit SPDX headers and third-party attribution.
- Assembly must resolve every declared template parameter and remain
  deterministic and offline.
- Run `go test ./...`, `go test -race ./...`, the configured linter, and
  `govulncheck` before completion.

<!-- promptkitty:release-cycle:begin -->
## PromptKitty release cycle

When asked to design, change, or run a PromptKitty release, act as a senior DevOps release maintainer and use GitHub Actions conventions only.

### Grounding and safety

- Treat the repository, its workflows, the selected release tag, and live GitHub/npm state as the sources of truth. Distinguish verified facts from inferences and assumptions. Mark unknown required details as `[UNKNOWN: ...]` and platform behavior that still needs checking as `[VERIFY: ...]`; never invent action names, versions, secrets, endpoints, paths, or release results.
- Inspect before mutating. Preserve unrelated worktree changes and never include them in release commits. Do not edit pinned PromptKit component files outside the documented upstream-ref and generation flow. Preserve SPDX headers, attribution, and every bundled third-party license.
- Obtain explicit authorization for the intended SemVer version and for public publication. Do not create prereleases, alternate npm dist-tags, overwrite tags, delete releases, or unpublish packages unless the user explicitly requests that exact action.
- Keep `main` deployable. Use a descriptive branch from an up-to-date `main`, conventional commits, a pull request, and squash merge. Never tag a commit that has not passed both pull-request checks and post-merge `main` checks.

### Prepare the release

1. Confirm the change classification and choose the next SemVer version. Verify that the tag does not already exist locally, on GitHub, or in npm.
2. Update the centralized release version and synchronize every checked-in version-bearing surface: CLI version, plugin manifests, marketplace metadata, pinned skill fallbacks, README Go examples, and release assertions. Use repository tests to detect drift instead of relying on manual inspection alone.
3. Review release-facing documentation and live metadata end to end for each audience: agent installation, npx CLI, global npm CLI, Go CLI, Go library, npm package page, GitHub Release, and repository About fields.
4. Preserve the root package's library-first boundary and deterministic offline assembly behavior. Do not broaden the release change into an upstream PromptKit refresh unless separately authorized.

### Verify before the pull request

- Run `go mod tidy` and confirm the module files change only when justified; run `go mod verify`.
- Validate GitHub workflow syntax with the repository-approved actionlint version. Verify referenced actions and tool versions from authoritative sources when updating them.
- Run `go test ./...`, `go test -race ./...`, the configured golangci-lint command, and `go tool govulncheck ./...`.
- Reproduce the npm release locally with the repository's Omnidist configuration and release-compatible tool version. Build all configured targets, stage npm only, apply the workflow's metadata and license customizations, run Omnidist verification, and inspect `npm pack --dry-run --json`.
- Confirm the staged native CLI reports the proposed version, the package manifest has the expected description, keywords, MIT license, homepage, issue tracker, repository, author, README, and platform optional dependencies, and agent-host setup writes the documented files.
- Review `git diff --check`, the complete staged diff, and the exact file list. Keep temporary staging files outside tracked source and leave unrelated files untouched.

### Merge and release

1. Push the branch and open a focused pull request containing the change summary and every verification command. Wait for all required GitHub checks; inspect annotations as well as conclusions.
2. Merge only when the pull request is green. Wait for the resulting `main` test, lint, and security runs and confirm the merge commit is synchronized with `origin/main`.
3. Confirm the npm publishing secret exists by name without exposing its value. Verify that the centralized version matches the intended `vX.Y.Z` tag.
4. Create an annotated SemVer tag at the green `main` commit and push only that tag.
5. Monitor both tag-triggered workflows independently: the GitHub Release workflow and the npm-only Omnidist workflow. Follow job dependencies through checks, preparation, artifact transfer, and publication. Treat warnings and annotations as findings even when a run is green.

### Verify publication

- Confirm the GitHub Release is published, non-draft, non-prerelease, and points to the intended tag and commit.
- Confirm npm `latest` resolves to the new version and re-check the public description, keywords, license, homepage, issue tracker, repository, author, and rendered README.
- Resolve every published platform package for macOS and Linux on amd64/arm64 and Windows on amd64.
- Run a fresh `npx --yes @baldaworks/promptkitty@latest --version`, install the tagged Go CLI into a temporary `GOBIN`, and confirm both report the release version.
- In a temporary project, run at least one agent-host setup path relevant to the release and verify the exact installed filenames and pinned fallback version.
- Re-read the GitHub repository description, website, topics, and license. Ensure their wording and typography match the current README.
- Report the release tag, merge commit, pull request, workflow URLs, npm package, completed checks, consumer tests, examined surfaces, method, exclusions, and limitations.

### Failure handling

- Before tagging, fix failures on the branch and repeat the entire affected verification set.
- After tagging, first determine which immutable external states exist: tag, GitHub Release, platform packages, main npm package, and dist-tags. Do not retry publication blindly.
- If a workflow failed before external publication, retry only the failed idempotent job after correcting the cause. If publication was partial, preserve the tag, document the exact state, and prefer a forward patch release; obtain explicit approval before any deletion or registry mutation.
- Never reuse a published version or move a public release tag. Do not claim rollback when npm artifacts are immutable; distinguish remediation, dist-tag movement, deprecation, and a new patch release.
- Before the final report, sample and re-verify at least three concrete claims against repository, GitHub, or npm evidence. Ensure the report is internally consistent and states what was not examined.
<!-- promptkitty:release-cycle:end -->
