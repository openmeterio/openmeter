## Infrastructure Rules

### Ci Cd

- Always run toolchain commands through the Nix CI shell `nix develop --impure .#ci -c <command>` when go/gofmt/golangci-lint/atlas are missing from the ambient shell; CI itself runs build/lint/test/migrate-check/generators this way. *(source: `AGENTS.md Testing; .github/workflows/ci.yaml`)*
- Always set POSTGRES_HOST=127.0.0.1 for DB-touching Go tests (and ensure Postgres is up via `docker compose up -d postgres`), or suites silently skip; run tests with -tags=dynamic (confluent-kafka-go) and the Make parallelism flags -p 128 -parallel 16. *(source: `AGENTS.md Testing; Makefile test`)*
- Never let generated artifacts drift: CI fails if `make update-openapi`, `make generate-javascript-sdk`, or `go generate ./...` produce any git diff or untracked files — run make generate-all and commit before pushing. *(source: `.github/workflows/ci.yaml generators-* jobs`)*
- Always keep .nvmrc byte-identical to `nix develop --impure .#ci -c node -v`; CI fails the build job on mismatch and flake.nix enterShell rewrites .nvmrc from `node -v`. *(source: `.github/workflows/ci.yaml Validate Node version file; flake.nix enterShell; AGENTS.md`)*
- The Go build/test in Docker and Depot CI uses CGO + musl static linking against librdkafka pinned to v2.14.1; redirect GOTMPDIR/TMPDIR to the workspace on Depot runners to avoid ENOSPC in /run. *(source: `Dockerfile (-tags musl, -linkmode external -extldflags static); flake.nix rdkafka rev v2.14.1; .github/workflows/ci.yaml build/test`)*
- The chi-middleware oapi-codegen template is patched: run `make patch-oapi-templates` (copies the vendored template and applies api/v3/templates/chi-middleware.tmpl.patch) before code generation, as `make generate` does automatically. *(source: `Makefile patch-oapi-templates; .github/workflows/ci.yaml generators-go`)*

### Dependency Registry

- Keep both Go modules tidy together: `go mod tidy` for the root and `go mod tidy -C collector` for the collector module (which replaces the parent via local replace ../); the collector pulls a large Benthos/Redpanda Connect dependency tree. *(source: `Makefile mod; collector/go.mod replace directive`)*
- Drop incidental go.sum additions (e.g. tablewriter) introduced by `make generate` or `atlas migrate diff` unless the task explicitly requires a dependency change. *(source: `AGENTS.md Coding Conventions`)*

### Distribution

- Container images and Helm charts are tag-only releases pushed to ghcr.io (images ghcr.io/openmeterio/openmeter + benthos-collector multi-arch linux/amd64+arm64; charts to oci://ghcr.io/openmeterio/helm-charts) — release.yaml jobs gate on github.ref_type == 'tag'. *(source: `.github/workflows/release.yaml; .github/workflows/artifacts.yaml; deploy/charts/openmeter/Chart.yaml`)*
- Release tags must match v[0-9]+.[0-9]+.[0-9]+ (optionally -dev.N / -beta.N); main pushes publish a per-commit npm beta (1.0.0-beta-<sha>, dist-tag beta) while tags publish latest. *(source: `.github/workflows/release.yaml on.push.tags + sdk-javascript-meta`)*
- The npm @openmeter/sdk is published via OIDC Trusted Publishing (id-token: write, no token); the trusted publisher entry is keyed on the caller workflow file + the `prod` environment, so npm-release.yaml must be invoked from release.yaml and run on a GitHub-hosted runner with NPM_CONFIG_PROVENANCE=true. *(source: `.github/workflows/npm-release.yaml`)*

### Env Setup

- Copy config.example.yaml to config.yaml (Make targets do this automatically) and `touch` it whenever config.example.yaml changes, or `make server`/worker targets abort with a diff warning. *(source: `Makefile server target config.yaml freshness check; AGENTS.md Configuration`)*
- Local Postgres DSN is postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable; start dependencies with `make up` (docker compose; profiles dev/redis/webhook control optional services) and stop with `make down`. *(source: `atlas.hcl; docker-compose.yaml; Makefile up/down`)*
- Load the repo environment with direnv (or `direnv exec . <command>`); when direnv/ambient tools are missing, fall back to `nix develop --impure .#ci -c ...`. The Nix shell provides go/node/python/atlasx/golangci-lint/air/helm/benthos/sqlc/spectral/codegraph. *(source: `AGENTS.md Configuration/Testing; flake.nix packages`)*

### Git

- tools/migrate/migrations/atlas.sum is append-only; pr-checks.yaml runs check_atlas_sum_append_only.py against the PR base SHA and migrate-check enforces non-linear=error, data_depend=error, incompatible=error (destructive allowed). *(source: `.github/workflows/pr-checks.yaml; atlas.hcl lint; Makefile migrate-check`)*
- Every PR must carry a release-note label (one of release-note/ignore, kind/feature, release-note/feature, kind/bug, release-note/bug-fix, release-note/breaking-change); commit messages must follow Conventional Commits enforced by commitizen/prek (`prek run -a` and `prek run --stage manual` in CI). *(source: `.github/workflows/pr-checks.yaml release-label; flake.nix git-hooks; .github/workflows/ci.yaml Validate commit messages`)*
- End commit messages with the `Co-Authored-By: Claude ...` trailer and PR bodies with the Claude Code generated line; branch off main before committing/pushing and only push when asked. *(source: `system git instructions; AGENTS.md`)*

### Secrets

- Provide release secrets via GitHub Actions: GITHUB_TOKEN (GHCR/helm push), POETRY_PYPI_TOKEN_PYPI from secrets.PYPI_TOKEN (PyPI), vars.DEPOT_PROJECT (Depot builds), vars.TEST_CLICKHOUSE_DSN (tests); SVIX_JWT_SECRET=DUMMY_JWT_SECRET is a non-sensitive dev value only. *(source: `.github/workflows/release.yaml; .github/workflows/ci.yaml; Makefile SVIX_JWT_SECRET`)*
- Trufflehog secret scanning runs on PRs/pushes to main and fails on findings; never commit real secrets. *(source: `.github/workflows/security.yaml secret-scanning`)*