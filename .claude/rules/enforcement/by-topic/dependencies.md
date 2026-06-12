# Enforcement: dependencies (6 rules)

Topic file. Loaded on demand when an agent works on something in the `dependencies` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `infra-nix-001` — Run toolchain commands through the Nix CI shell when go/gofmt/golangci-lint/atlas are missing

*source: `deep_scan`*

**Why:** Always run toolchain commands through the Nix CI shell `nix develop --impure .#ci -c <command>` when go/gofmt/golangci-lint/atlas are missing from the ambient shell; CI itself runs build/lint/test/migrate-check/generators this way. Load the repo environment with direnv (or direnv exec . <command>) so project-specific tools are applied consistently.

**Example:**

```
nix develop --impure .#ci -c make lint-go
```

### `infra-config-001` — Touch config.yaml whenever config.example.yaml changes or make server/worker targets abort

*source: `deep_scan`*

**Why:** Copy config.example.yaml to config.yaml (Make targets do this automatically) and touch it whenever config.example.yaml changes, or make server / worker targets abort with a diff warning. Local Postgres DSN is postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable; start dependencies with make up and stop with make down.

### `infra-nvmrc-001` — Keep .nvmrc byte-identical to the Nix CI shell's node -v

*source: `deep_scan`*

**Why:** Always keep .nvmrc byte-identical to `nix develop --impure .#ci -c node -v`; CI fails the build job on mismatch and flake.nix enterShell rewrites .nvmrc from node -v. The committed .nvmrc is the GitHub Actions source of truth for Node-based jobs.

### `infra-modtidy-001` — Tidy both Go modules together and drop incidental go.sum additions from generate/diff

*source: `deep_scan`*

**Why:** Keep both Go modules tidy together: go mod tidy for the root and go mod tidy -C collector for the collector module (which replaces the parent via local replace ../). Drop incidental go.sum additions (e.g. tablewriter) introduced by make generate or atlas migrate diff unless the task explicitly requires a dependency change.

### `infra-release-001` — Releases are tag-only; PRs need a release-note label and Conventional Commit messages

*source: `deep_scan`*

**Why:** Container images and Helm charts are tag-only releases to ghcr.io, with release.yaml jobs gating on github.ref_type == 'tag' and tags matching v[0-9]+.[0-9]+.[0-9]+ (optionally -dev.N/-beta.N). Every PR must carry a release-note label (release-note/ignore, kind/feature, release-note/feature, kind/bug, release-note/bug-fix, release-note/breaking-change), and commit messages must follow Conventional Commits enforced by commitizen/prek. End commit messages with the Co-Authored-By: Claude trailer.

### `infra-cgo-001` — Builds use CGO + musl static linking against librdkafka v2.14.1; use -tags=dynamic locally

*source: `deep_scan`*

**Why:** The Go build/test in Docker and Depot CI uses CGO + musl static linking against librdkafka pinned to v2.14.1, with GO_BUILD_FLAGS=-tags=dynamic for local builds (confluent-kafka-go requires it). Redirect GOTMPDIR/TMPDIR to the workspace on Depot runners to avoid ENOSPC in /run.
