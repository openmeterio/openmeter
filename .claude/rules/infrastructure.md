## Infrastructure Rules

### Ci Cd

- Always use Nix .#ci shell on Depot runners for CI — all jobs in ci.yaml use 'nix develop --impure .#ci -c <command>' to pin the Go/Node/Atlas/golangci-lint toolchain. *(source: `.github/workflows/ci.yaml (Set up Nix + Build nix environment steps)`)*
- Always run 'make migrate-check' in CI (migrate-check-schema, migrate-check-diff, migrate-check-lint, migrate-check-validate) to verify Ent schema, migration diff, lint, and atlas.sum integrity. *(source: `.github/workflows/ci.yaml and Makefile (migrate-check targets)`)*
- Always run all generators and fail if the working tree is dirty in CI — catches TypeSpec, Ent, Wire, or Goverter regen not committed alongside source changes. *(source: `.github/workflows/ci.yaml (generate-all step followed by git diff check)`)*
- Always run the 'require-all-reviewers' gate when a PR is labeled 'require-all-reviewers' — all requested reviewers must approve before merge. *(source: `.github/workflows/require-all-reviewers.yml`)*
- Always run OpenSSF Scorecard analysis on every main push and weekly (Fridays) — analysis-scorecard.yaml publishes results to GitHub Security tab. *(source: `.github/workflows/analysis-scorecard.yaml`)*

### Dependency Registry

- Always use pnpm (v10.33.0+) as the package manager for all Node.js workspaces (TypeSpec, JavaScript SDK) — the packageManager field is pinned in api/spec/package.json and api/client/javascript/package.json. *(source: `api/spec/package.json (packageManager field) and api/client/javascript/package.json`)*

### Distribution

- Always publish Docker images to ghcr.io/openmeterio/openmeter and ghcr.io/openmeterio/benthos-collector using Depot depot-build-push-action with multi-platform support (linux/amd64, linux/arm64). *(source: `.github/workflows/artifacts.yaml (platforms: linux/amd64,linux/arm64)`)*
- Always publish npm @openmeter/sdk via OIDC trusted publishing (not a stored npm token) — npm-release.yaml is configured for OIDC; trusted publisher entry on npmjs.com is configured against the caller workflow + environment prod. *(source: `.github/workflows/npm-release.yaml`)*
- Always publish Helm charts to GHCR OCI registry (not a traditional Helm index) on release tags, with Chart.yaml appVersion aligned to the container image tag. *(source: `.github/workflows/release.yaml (helm-release job) and deploy/charts/openmeter/Chart.yaml`)*
- Never push untrusted (PR) Docker images to GHCR — use the untrusted-artifacts.yaml reusable workflow for PRs, which builds but does not publish container images. *(source: `.github/workflows/untrusted-artifacts.yaml`)*

### Env Setup

- Always copy config.example.yaml to config.yaml before starting local services — Make targets warn if config.yaml is older than config.example.yaml. *(source: `Makefile (server/sink-worker/balance-worker/billing-worker/notification-service targets: config.yaml staleness check)`)*
- Always start PostgreSQL before running tests with 'docker compose up -d postgres' — the make test target checks PostgreSQL availability and exits with an error if it is not running. *(source: `Makefile (test target PGPASSWORD check)`)*
- Always keep .nvmrc in sync with the Nix .#ci shell's node version — CI validates the .nvmrc file against the Nix shell before running Node-based builds. *(source: `.nvmrc and .github/workflows/ci.yaml (Validate Node version file step)`)*

### Git

- Always add exactly one release-note label to every PR before merging (release-note/ignore, kind/feature, release-note/bug-fix, release-note/breaking-change, etc.) — the PR Checks workflow enforces this via mheap/github-action-required-labels. *(source: `.github/workflows/pr-checks.yaml`)*

### Secrets

- Never commit secrets or API keys — Trufflehog secret scanning runs on every PR and main push with fail_on_findings=true and blocks merge on findings. *(source: `.github/workflows/security.yaml (fail_on_findings: 'true')`)*
- Never commit config.yaml, .env*, *.key, *.pem, or any file containing credentials — config.yaml is gitignored and generated from config.example.yaml. *(source: `.gitignore and Makefile (config.yaml target)`)*