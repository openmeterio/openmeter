## Infrastructure Rules

### Ci Cd

- Always use Nix .#ci shell on Depot runners for CI — all jobs in ci.yaml use 'nix develop --impure .#ci -c <command>' to pin the Go/Node/Atlas toolchain *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/ci.yaml (Set up Nix + Build nix environment steps)`)*
- Always run the 'require-all-reviewers' gate when a PR is labeled 'require-all-reviewers' — all requested reviewers must approve before merge *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/require-all-reviewers.yml`)*
- Always run OpenSSF Scorecard analysis on every main push and weekly (Fridays) — analysis-scorecard.yaml publishes results to GitHub Security tab *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/analysis-scorecard.yaml`)*
- Never push untrusted (PR) Docker images to GHCR — the untrusted-artifacts.yaml reusable workflow builds but does not publish container images for PRs *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/untrusted-artifacts.yaml`)*

### Dependency Registry

- Always use pnpm (v10.33.0) as the package manager for all Node.js workspaces (TypeSpec, JavaScript SDK) — the packageManager field is pinned in api/spec/package.json *(source: `/Users/hamutarto/DEV/gbr/openmeter/api/spec/package.json (packageManager field)`)*

### Distribution

- Always publish Docker images to ghcr.io/openmeterio/openmeter and ghcr.io/openmeterio/benthos-collector using Depot depot-build-push-action with multi-platform support (linux/amd64, linux/arm64) *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/artifacts.yaml (platforms: linux/amd64,linux/arm64)`)*
- Always publish npm @openmeter/sdk via OIDC trusted publishing (not a stored npm token) — the npm-release.yaml reusable workflow is configured for OIDC *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/npm-release.yaml`)*
- Always publish Helm charts to GHCR OCI registry (not a traditional Helm index) on release tags, with Chart.yaml appVersion aligned to the container image tag *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/release.yaml (helm-release job) and deploy/charts/openmeter/Chart.yaml`)*

### Env Setup

- Always copy config.example.yaml to config.yaml before starting local services — Make targets warn if config.yaml is older than config.example.yaml *(source: `/Users/hamutarto/DEV/gbr/openmeter/Makefile (server/sink-worker/balance-worker targets: config.yaml staleness check)`)*
- Always keep .nvmrc in sync with the Nix .#ci shell's node version — CI validates the .nvmrc file against the Nix shell before running Node-based builds *(source: `/Users/hamutarto/DEV/gbr/openmeter/.nvmrc and AGENTS.md (Tips for working with the codebase)`)*

### Git

- Always add exactly one release-note label to every PR before merging (release-note/ignore, kind/feature, release-note/bug-fix, release-note/breaking-change, etc.) — the PR Checks workflow enforces this via mheap/github-action-required-labels *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/pr-checks.yaml`)*

### Secrets

- Never commit secrets or API keys — Trufflehog secret scanning runs on every PR and main push with fail_on_findings=true and blocks merge on findings *(source: `/Users/hamutarto/DEV/gbr/openmeter/.github/workflows/security.yaml (fail_on_findings: 'true')`)*