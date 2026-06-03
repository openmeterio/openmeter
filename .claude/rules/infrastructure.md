## Infrastructure Rules

### Ci Cd

- Always run CI build/test/lint/migrate/generator commands through 'nix develop --impure .#ci -c <command>' on Depot runners — the Nix .#ci shell pins Go/Node/Python/Atlas/golangci-lint/librdkafka versions *(source: `.github/workflows/ci.yaml, flake.nix`)*
- Always commit regenerated artifacts: CI fails (git diff/git status --porcelain non-empty) if make update-openapi, generate-javascript-sdk, or go generate ./... produce uncommitted changes; also runs make migrate-check (schema, diff, lint --latest 10, validate) *(source: `.github/workflows/ci.yaml (generators-openapi, generators-go, generators-javascript-sdk, migrations jobs)`)*
- Always keep .nvmrc in sync with the Nix .#ci shell node version — flake.nix enterShell writes 'node -v > .nvmrc' and CI's 'Validate Node version file' step fails if .nvmrc differs from 'nix develop --impure .#ci -c node -v' *(source: `.github/workflows/ci.yaml (Validate Node version file), flake.nix enterShell, .nvmrc`)*
- Always lint Go with the project golangci-lint (v2, enabled: errcheck/govet/staticcheck/sloglint/bodyclose/misspell/ineffassign/nolintlint/unconvert/whitespace; formatters gci/gofumpt/goimports with prefix github.com/openmeterio/openmeter) — collector/benthos/internal and examples are excluded *(source: `.golangci.yaml, .golangci-fast.yaml`)*

### Dependency Registry

- Always use pnpm (10.33.2 for api/spec, 10.33.0 for api/client/javascript, pinned in packageManager fields) for Node workspaces, and Poetry (poetry-core>=1.0.0) for the Python SDK — never npm/yarn; release.sh overwrites pyproject.toml version at publish *(source: `api/spec/package.json, api/client/javascript/package.json, api/client/python/pyproject.toml`)*
- Always apply vendored TypeSpec/openapi-typescript patches via pnpm patchedDependencies (patches target compiled dist/, not src); regenerate the patch and its base hash after any upstream version bump or pnpm install fails *(source: `api/spec/package.json (pnpm.patchedDependencies), api/client/javascript/package.json`)*
- Always run 'make patch-oapi-templates' before code generation — it copies oapi-codegen's chi-middleware.tmpl into api/v3/templates and applies chi-middleware.tmpl.patch for deepObject filter parsing; this template patch must stay aligned with api/v3/codegen.yaml on oapi-codegen upgrades *(source: `Makefile (patch-oapi-templates), api/CLAUDE.md`)*

### Distribution

- Always publish container images to ghcr.io/openmeterio/openmeter and ghcr.io/openmeterio/benthos-collector via Depot depot-build-push-action (linux/amd64+arm64); the multi-stage Dockerfile builds six static-musl binaries (CGO_ENABLED=1, -tags musl, linkmode external -static) *(source: `.github/workflows/artifacts.yaml, Dockerfile`)*
- Never push untrusted (PR-from-fork) Docker images to GHCR — untrusted-artifacts.yaml builds but does not publish; only trusted artifacts.yaml (push or same-repo PR) publishes *(source: `.github/workflows/untrusted-artifacts.yaml, .github/workflows/ci.yaml (trusted-artifacts gate)`)*
- Always publish @openmeter/sdk via npm OIDC trusted publishing (id-token: write, environment prod, GitHub-hosted runner) configured against the caller workflow filename, not a stored npm token *(source: `.github/workflows/npm-release.yaml`)*
- Always publish Helm charts to the GHCR OCI registry on release tags with Chart.yaml appVersion aligned to the container image tag (make package-helm-chart sets --app-version) *(source: `.github/workflows/release.yaml, Makefile (package-helm-chart), deploy/charts/openmeter/Chart.yaml`)*

### Env Setup

- Always copy config.example.yaml to config.yaml before running services (Make targets auto-copy and warn if config.yaml is older than config.example.yaml) and start Postgres ('docker compose up -d postgres') before tests — make test exits early if Postgres is unreachable *(source: `Makefile (config.yaml, server, test targets), config.example.yaml`)*
- Always load the repo environment with direnv (or 'direnv exec . <command>') so project tools are applied; when go/gofmt/atlas are missing from the ambient shell, run via 'nix develop --impure .#ci -c ...' *(source: `AGENTS.md Testing/Configuration, flake.nix`)*

### Git

- Always add exactly one release-note label to every PR before merge (release-note/ignore, kind/feature, release-note/bug-fix, release-note/breaking-change, ...) — pr-checks.yaml enforces it via required-labels; require-all-reviewers.yml gates PRs labeled 'require-all-reviewers' *(source: `.github/workflows/pr-checks.yaml, .github/workflows/require-all-reviewers.yml`)*
- Always write commitizen-conventional commit messages — flake.nix git-hooks run commitizen/commitizen-branch via prek, and CI runs 'prek run -a' and 'prek run --stage manual' to validate them *(source: `flake.nix (git-hooks.hooks), .github/workflows/ci.yaml (Validate commit messages)`)*

### Secrets

- Never commit secrets, config.yaml, .env*, *.key, or *.pem — config.yaml is gitignored and copied from config.example.yaml by Make targets; Trufflehog scans every PR/push with fail_on_findings=true *(source: `.github/workflows/security.yaml, .gitignore, Makefile (config.yaml target)`)*