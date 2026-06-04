## Infrastructure Rules

### Ci Cd

- Always run CI build/test/lint/migrate/generator commands through 'nix develop --impure .#ci -c <command>' on Depot runners — the Nix .#ci shell pins Go/Node/Python/Atlas/golangci-lint/librdkafka versions *(source: `.github/workflows/ci.yaml, flake.nix`)*
- Always build/test Go with -tags=dynamic (GO_BUILD_FLAGS) so confluent-kafka-go links librdkafka; CI sets GOTMPDIR/TMPDIR to the workspace disk to avoid /run space exhaustion under cgo external linking *(source: `Makefile (GO_BUILD_FLAGS), .github/workflows/ci.yaml (Build components)`)*
- Always commit regenerated artifacts — CI fails (git diff / git status --porcelain non-empty) if make update-openapi / generate-javascript-sdk / go generate produce uncommitted changes; also runs make migrate-check (schema, diff, lint --latest 10, validate) *(source: `.github/workflows/ci.yaml (generators-openapi, generators-javascript-sdk, migrations jobs)`)*
- Always keep .nvmrc in sync with the Nix .#ci node version — flake.nix enterShell writes 'node -v > .nvmrc' and CI's 'Validate Node version file' step fails if it differs from the Nix shell node -v *(source: `.github/workflows/ci.yaml (Validate Node version file), flake.nix enterShell, .nvmrc`)*
- Always lint Go with the project golangci-lint v2 (.golangci.yaml: errcheck/govet/staticcheck/sloglint/bodyclose/misspell/ineffassign/nolintlint/unconvert/whitespace; formatters gci/gofumpt/goimports prefix github.com/openmeterio/openmeter) — collector/benthos/internal and examples are excluded *(source: `.golangci.yaml, .golangci-fast.yaml`)*
- Always generate Atlas migrations with 'atlas migrate --env local diff'; PR Checks enforces atlas.sum is append-only via .github/tools/migrate/check_atlas_sum_append_only.py against the PR base SHA, and atlas.hcl lints with non_linear+data_depend+incompatible as errors *(source: `.github/workflows/pr-checks.yaml (migration-atlas-sum), atlas.hcl`)*

### Dependency Registry

- Always use pnpm (10.33.2 for api/spec, 10.33.0 for api/client/javascript, pinned in packageManager fields) for Node workspaces and Poetry (poetry-core>=1.0.0) for the Python SDK — never npm/yarn *(source: `api/spec/package.json, api/client/javascript/package.json, api/client/python/pyproject.toml`)*
- Always apply vendored TypeSpec/openapi-typescript patches via pnpm patchedDependencies (patches target compiled dist/, not src/); regenerate the patch and its base hash after any upstream version bump or pnpm install fails *(source: `api/spec/package.json (pnpm.patchedDependencies), api/client/javascript/package.json (patches/openapi-typescript.patch)`)*
- Always run 'make patch-oapi-templates' before code generation — it copies oapi-codegen's chi-middleware.tmpl into api/v3/templates and applies chi-middleware.tmpl.patch for deepObject filter parsing; the patch must stay aligned with api/v3/codegen.yaml on oapi-codegen upgrades *(source: `Makefile (patch-oapi-templates), api/v3/codegen.yaml`)*
- Always keep the collector as a separate Go module (collector/go.mod, replace openmeter => ../) so its heavy Redpanda Connect transitive dependency set is isolated from the root module; run 'go mod tidy -C collector' alongside root tidy *(source: `collector/go.mod, Makefile (mod target)`)*

### Distribution

- Always publish container images to ghcr.io/openmeterio/openmeter and ghcr.io/openmeterio/benthos-collector via Depot depot-build-push-action (linux/amd64+arm64); the multi-stage Dockerfile builds six static-musl binaries (CGO_ENABLED=1, -tags musl, -linkmode external -static) and benthos-collector.Dockerfile builds CGO_ENABLED=0 *(source: `.github/workflows/artifacts.yaml, Dockerfile, benthos-collector.Dockerfile`)*
- Never push untrusted (PR-from-fork) Docker images to GHCR — untrusted-artifacts.yaml builds with push:false; only artifacts.yaml (called with publish:true from release.yaml on tags) publishes *(source: `.github/workflows/untrusted-artifacts.yaml, .github/workflows/artifacts.yaml, .github/workflows/release.yaml`)*
- Always publish @openmeter/sdk via npm OIDC trusted publishing (id-token: write, environment prod, GitHub-hosted runner, NPM_CONFIG_PROVENANCE=true) keyed on the caller workflow filename — tag pushes go to dist-tag latest, main pushes publish a per-commit beta *(source: `.github/workflows/npm-release.yaml, .github/workflows/release.yaml (sdk-javascript-meta)`)*
- Always publish Python SDK via 'make -C api/client/python publish-python-sdk' (release.sh computes PEP-440 version from live PyPI, stamps _version.py/_commit.py, poetry publish --build); stable releases are tag-only, beta on main push (sdk-python-dev-release.yaml, environment dev) *(source: `.github/workflows/release.yaml (sdk-python-release), .github/workflows/sdk-python-dev-release.yaml, api/client/python/scripts/release.sh`)*
- Always publish Helm charts to oci://ghcr.io/openmeterio/helm-charts on release tags via 'make package-helm-chart CHART=<c> VERSION=<v>' which sets --app-version; keep Chart.yaml appVersion aligned to the released image tag (openmeter chart bundles altinity-clickhouse-operator, bitnami kafka/postgresql/redis deps gated by conditions) *(source: `.github/workflows/release.yaml (helm-release), Makefile (package-helm-chart), deploy/charts/openmeter/Chart.yaml`)*

### Env Setup

- Always copy config.example.yaml to config.yaml before running services (Make targets auto-copy and warn if config.yaml is older than config.example.yaml) and start Postgres ('docker compose up -d postgres') before tests — make test exits early via a psql healthcheck if Postgres is unreachable *(source: `Makefile (config.yaml, server, test targets), config.example.yaml`)*
- Always set POSTGRES_HOST=127.0.0.1 for DB tests (suites silently skip otherwise); use docker-compose profiles (dev/redis/postgres/webhook) for optional services; the local stack pins kafka cp-kafka 8.0.3, clickhouse 25.12.3, redis 7.4.7, postgres 14.20, svix v1.84.1 *(source: `Makefile (test), docker-compose.yaml, docker-compose.base.yaml`)*

### Git

- Always add exactly one release-note label to every PR before merge (release-note/ignore, kind/feature, release-note/feature, kind/bug, release-note/bug-fix, release-note/breaking-change, release-note/deprecation, area/dependencies, kind/refactor, release-note/misc, kind/documentation) — pr-checks.yaml enforces it via mheap/github-action-required-labels *(source: `.github/workflows/pr-checks.yaml (release-label)`)*
- Always write commitizen-conventional commit messages — flake.nix git-hooks run commitizen/commitizen-branch via prek, and CI runs 'prek run -a' and 'prek run --stage manual' to validate them *(source: `flake.nix (git-hooks.hooks), .github/workflows/ci.yaml (Validate commit messages)`)*

### Secrets

- Never commit secrets, config.yaml, .env.local, *.key, or *.pem — config.yaml and /.env.local are gitignored; Trufflehog (Kong secret-scan) scans every PR/push with fail_on_findings=true, plus Kong SCA scan and GitHub workflow scan *(source: `.github/workflows/security.yaml, .gitignore, .syft.yaml`)*