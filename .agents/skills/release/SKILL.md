---
name: release
description: Create a release for gsc-cli. Use when preparing a new version, release PR, tag, and GoReleaser publish.
metadata:
  short-description: Prepare and publish a gsc-cli release
---

# Release Process (gsc-cli)

## Preconditions

- Repo is clean.
- `HOMEBREW_TAP_GITHUB_TOKEN` exists in repo secrets.
- Release tag does not already exist (`git tag --list vX.Y.Z`).

## Steps

1. **Create release branch**: `git checkout -b release/vX.Y.Z`
2. **Update changelog** in `CHANGELOG.md`:
   - Keep `## [Unreleased]` at the top.
   - Add `## [X.Y.Z]`.
   - Include user-facing changes only.
   - Ensure each entry has a link to the PR that introduced the change.
3. **Run preflight checks**:
   - `gofmt -w .`
   - `go vet ./...`
   - `go test ./...`
   - `go build ./...`
4. **Run release dry-run**:
   - Preferred: `goreleaser release --snapshot --clean`
   - Fallback if `goreleaser` is not installed: `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean`
5. **Commit and open PR**:
   - Commit changelog changes.
   - Push branch and open PR to `main`.
   - PR title: `chore(release): X.Y.Z`
6. **Merge PR** after CI is green.
7. **Sync and tag**:
   - `git checkout main && git pull --ff-only`
   - `git tag vX.Y.Z && git push origin vX.Y.Z`
8. **Verify publish**:
   - Watch release workflow: `gh run watch`
   - Confirm release/assets: `gh release view vX.Y.Z`
   - Confirm tap update landed on `jpreagan/homebrew-tap`.

## Conventions

- Follow semver:
  - patch = fixes
  - minor = features
  - major = breaking changes
- Branch: `release/vX.Y.Z`
- PR title: `chore(release): X.Y.Z`
- Tag: `vX.Y.Z`

## Notes

- Do not include secrets or token values in changelog, logs, or PR descriptions.
