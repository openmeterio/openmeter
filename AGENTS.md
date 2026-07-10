# OpenMeter

OpenMeter is a usage metering and billing platform for AI and DevTool companies, built in Go.

## Quick Reference

Use the `Makefile` for all common tasks. A `justfile` also exists but is seldom used.
OpenMeter is a metering and billing platform with usage based pricing and access control.

## Tips for working with the codebase

If during your work anything confuses you or something isn't trivial for you, please augment AGENTS.md with your findings so next time it will be easier for you. AGENTS.md files are for you to edit and update as you go so you can interact with the codebase the most effectively.

Development commands are run via `Makefile`, it contains all commonly used commands during development. A `justfile` is also present but seldom used. Use the Makefile commands for common tasks like running tests, generating code, linting, etc.
The committed `.nvmrc` is the GitHub Actions source of truth for Node-based jobs on GitHub-hosted runners. Keep it aligned with the Nix `.#ci` shell's `node -v`; `flake.nix` refreshes it in `enterShell`, and CI validates the file against the Nix shell before running builds.

## AGENTS.md maintenance

- Treat this file as long-lived project guidance for all agents and contributors.
- Treat AGENTS.md as the repo-local source of truth. Do not bypass coding style, workflow, testing, or documentation guidance in this file. If a requested change appears to conflict with AGENTS.md, ask the human developer/reviewer to confirm the exception before proceeding, and make the exception explicit in the handoff.
- Prefer durable wording over time-based wording (avoid labels like "recent", "latest", "today").
- Keep entries actionable and specific (what to do, where, and why), not conversational history.
- Capture universal truths and cross-cutting coding conventions here when they become repeated practice or reviewer expectation. Do not leave them only in chat or pull request comments.
- Capture subsystem-specific guidance in the closest applicable nested `AGENTS.md` when the guidance should always apply to a subtree, such as `api/spec/AGENTS.md` for TypeSpec and SDK guidance. Use skills for reusable workflows or domain procedures that agents opt into for a task. Skills must stay usable by both Claude and Codex: write them as plain repo guidance, keep `.agents/skills` as the source of truth, and avoid assistant-specific assumptions unless a workflow truly requires them.
- When adding new guidance, fold it into the most relevant section and remove/merge stale or duplicate notes.

## Testing

| Task | Command |
|------|---------|
| Start dependencies | `make up` |
| Stop dependencies | `make down` |
| Run API server (hot reload) | `make server` |
| Run all tests | `make test` (root module only; excludes `e2e/`, its own module) |
| Run e2e tests | `make etoe` |
| Generate all code | `make generate-all` |
| Generate Go code only | `make generate` (runs `go generate ./...`) |
| Generate API + SDKs | `make gen-api` |
| Lint all | `make lint` |
| Lint Go only | `make lint-go` |
| Format code | `make fmt` |
| Tidy modules | `make mod` (root + `collector` + `e2e`) |
| Build all binaries | `make build` |

## Architecture

**Entry points:** `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`

Core business logic is in `openmeter/`, shared utilities in `pkg/`, API layer in `api/`.

**Stack:** Go + PostgreSQL (Ent ORM) + Kafka + ClickHouse. API defined in TypeSpec, generated to OpenAPI.

Domain packages under `openmeter/` follow a layered service/adapter pattern. See the `/service` skill for full details.

`cmd/server/main.go` now migrates the database before creating the default namespace. Register namespace handlers before `initNamespace(...)` if they must provision the default namespace during startup.

**Module layout:** the repo is three separate Go modules. The root module (`github.com/openmeterio/openmeter`) holds all production code (`cmd/`, `openmeter/`, `pkg/`, etc.). `api/v3/client` is the standalone, publishable v3 Go SDK module. `e2e/` is a third, never-published, test-only module that imports both — it pins itself to the working tree of each via `replace github.com/openmeterio/openmeter => ../` and `replace .../api/v3/client => ../api/v3/client`, so e2e always tests local code regardless of what's tagged. The root module must never `require` the SDK module: a `require` on an untagged nested module resolves to an unresolvable `v0.0.0` for anyone outside this repo (the `replace` directive that makes it resolve locally is invisible downstream), so any code that needs the SDK — today, only `e2e/` — has to live in its own module rather than the root one. Because of this, root `go build ./...` / `go test ./...` / `go vet ./...` no longer see `e2e/` at all; use `make etoe` (runs it against a live server) or `go test -C e2e ./...` / `go vet -C e2e ./...` (compiles it standalone, no server needed) instead. `make lint-go` and `make mod` already cover all three modules. For editor/gopls support across all three, run `go work init . ./api/v3/client ./e2e` locally — `go.work`/`go.work.sum` are gitignored and must never be committed.

### Project Layout

```
cmd/                    # Service entrypoints
openmeter/              # Core business logic (billing, customer, entitlement, meter, etc.)
openmeter/ent/schema/   # Ent entity definitions (source of truth for DB schema)
openmeter/ent/db/       # Generated ent code (DO NOT EDIT)
api/                    # API specs, generated code, SDKs
api/spec/               # TypeSpec API definitions (source of truth for API)
pkg/                    # Shared utility packages
tools/migrate/          # Migration tooling and SQL migration files
e2e/                    # End-to-end tests
deploy/                 # Helm charts
docs/                   # Documentation and ADRs
```

## Code Generation

All generated files have `// Code generated by X, DO NOT EDIT.` headers — never edit them manually:


| Generated artifact | Source | Regenerate with |
|---|---|---|
| `api/openapi.yaml`, `api/openapi.cloud.yaml` | TypeSpec in `api/spec/` | `make gen-api` |
| `api/client/javascript/`, `api/client/go/` | OpenAPI spec | `make gen-api` |
| `api/v3/client/` (v3 Go SDK, standalone module) | TypeSpec in `api/spec/` via `@openmeter/typespec-go` | `make gen-api` |
| `api/api.gen.go`, `api/v3/api.gen.go` | OpenAPI spec via oapi-codegen | `make gen-api` |
| `api/client/go/client.gen.go` | OpenAPI spec | `make gen-api` |
| `**/ent/db/` | Ent schema in `openmeter/ent/schema/` | `make generate` |
| `**/wire_gen.go` | Wire providers in `**/wire.go` | `make generate` |
| `**/convert.gen.go` | Goverter converter interfaces (`**/convert.go`) | `make generate` |
| `billing/derived.gen.go` | Goderive annotations | `make generate` |
| `tools/migrate/migrations/` | Ent schema diff | `atlas migrate --env local diff <name>` |

**Workflow for changing the API:**

1. Edit TypeSpec files in `api/spec/`
2. Run `make gen-api` to regenerate OpenAPI spec and SDKs
3. Run `make generate` to regenerate Go server/client code

The TypeSpec JS client emitted from `api/spec/packages/aip` now lands in `api/spec/packages/aip-client-javascript/`. The emitter regenerates `src/` and `README.md` only; `package.json` is **stable, hand-maintained, and committed** (the emitter's `writeOutput` only writes the paths it lists, so the manifest survives regeneration). Keep hand-written Playwright config, tests, and helpers outside `src/`, and put test-runner dependencies/scripts in the `api/spec/package.json` workspace root rather than the client package manifest. Static publish metadata (`name`, `license`, `homepage`, `repository`) lives directly in the client `package.json`; only the per-release `version` is injected at publish time.

The emitted `api/spec/packages/aip-client-javascript/src/openMeterClient.ts` exposes aggregated sub-client getters on `OpenMeterClient` (e.g. `meters`, `customers`, `features`, `productCatalog`, `subscriptions`, `billing`, `entitlements`). Access operations through those getters, e.g. `sdk.meters.list()`, `sdk.customers.create(...)`, `sdk.productCatalog.createPlan(...)`. Note that resources are grouped by tag, so some operations live under a grouped client rather than a same-named top-level getter (for example plan operations are `sdk.productCatalog.createPlan`, not `sdk.plans.create`).

**Workflow for changing Go types/DI:**

1. Edit the source files (ent schema, wire.go, converter interfaces)
2. Run `make generate` (or `go generate ./...`)

## Database Migrations

Uses [ent](https://entgo.io) for schema definition and [Atlas](https://atlasgo.io/) for migration generation. Migrations are in `tools/migrate/migrations/` using golang-migrate format.

**Schema files:** `openmeter/ent/schema/*.go`

**Workflow for schema changes:**

1. Edit the ent schema in `openmeter/ent/schema/`
2. Run `make generate` to regenerate ent code in `openmeter/ent/db/`
3. Generate migration: `atlas migrate --env local diff <migration-name>`
   - This creates timestamped `.up.sql` / `.down.sql` files in `tools/migrate/migrations/`
   - Also updates `tools/migrate/migrations/atlas.sum`
4. Migrations run automatically on startup when `postgres.autoMigrate` is set to `ent` (default for dev) or `migration`

**Ent view caveat:** in this repo's current Ent/Atlas setup, schemas declared with `ent.View` can generate query code under `openmeter/ent/db/`, but they do not appear in `openmeter/ent/db/migrate/schema.go` or the generated `migrate.Tables` list. If `atlas migrate --env local diff ...` reports no changes for a new view, verify whether the view exists in generated migration metadata before debugging Atlas; view DDL may need an explicit SQL migration until generator support is added.

**Atlas config:** `atlas.hcl` — schema source is `ent://openmeter/ent/schema`, migrations dir is `file://tools/migrate/migrations`.

**Local Postgres:** `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`

## Testing

Tests require PostgreSQL running locally. Start it with `docker compose up -d postgres`.

Keep domain test helpers under `openmeter/.../testutils` independent from `app/common`. Build test dependencies from the underlying package constructors (repos, adapters, services, `lockr`) instead of importing the application wiring layer, or unrelated wiring additions can create test-only import cycles.

For usage-based billing lifecycle tests, prefer driving behavior through `charges.Service.Create`, `AdvanceCharges`, and `ApplyPatches` rather than calling lower-level charge adapters directly. To model late-arriving or newly visible usage, use `MockStreamingConnector` events with explicit `StoredAt` values (or `SetSimpleEvents`) so the test exercises the real stored-at cutoff logic in finalization.

For OpenMeter Go tests that touch the database, explicitly set `POSTGRES_HOST=127.0.0.1`. Without it, many suites will skip during setup even if PostgreSQL is running and the repo environment is otherwise loaded correctly.

Use the repo's Nix CI dev shell when `go`, `gofmt`, or other toolchain binaries are missing from the ambient shell. The CI and local-compatible invocation pattern is:

```bash
nix develop --impure .#ci -c <command>
```

Always invoke `nix develop` with the repo root as the working directory (use absolute paths in `<command>` instead of `cd`-ing first). The devenv `enterShell` writes CWD-relative state: run from a subdirectory it drops `.devenv/`, `.nvmrc`, and `.pre-commit-config.yaml` there (which then fail `prettier --check` in lint) and reinstalls the git pre-commit hook with that subdirectory's config path baked in, breaking later commits until a root-CWD `nix develop` run repairs it.

Codex's default shell may not auto-load `.envrc`, so `direnv`-managed tools like `go` can be missing even when the repo is configured correctly. In that case, run commands through `nix develop --impure .#ci -c ...` explicitly instead of assuming the ambient shell reflects the flake environment. `direnv exec . <command>` is also a valid one-off fallback when `direnv` is installed and the repo has already been allowed.

When invoking commands through Codex tools, prefer direct command execution. Do not wrap commands in `sh -lc`, `bash -lc`, or other helper shells when the command can be run directly. For environment variables, prefer `env KEY=value <command>` or `KEY=value <command>` over shell-wrapped forms. This keeps failures attributable to the actual toolchain/runtime being tested.

In tests, prefer `t.Context()` when a `testing.T` or `testing.TB` is available instead of introducing `context.Background()`. This keeps cancellation and test-scoped lifecycle tied to the test harness.

Prefer one consistent test harness style over mixed ad hoc structures. Use production-backed paths, such as rating-backed or service-backed fixtures, when the real path can express the scenario; keep hand-assembled fixtures for cases that cannot be produced realistically. If a behavior is a suite-wide rule, hardcode it into the shared harness instead of exposing it as per-test knobs.

Avoid redundant test helpers and duplicate setup paths. Prefer parameterizing one helper over maintaining near-identical helpers, use literal helper names that state exactly what they do, and inline single-use helpers that only wrap setup, conversion, or assertions even when the test becomes longer. Add a test helper only when it is used by at least two tests in the same package or when the helper name captures non-obvious domain semantics that would otherwise be easy to miss. Clean up dead test helpers immediately after refactors.

For service and lifecycle subtests, start each subtest body with concise intent comments when the scenario is non-trivial:

```go
// given:
// - ...
// when:
// - ...
// then:
// - ...
```

When using `clock.FreezeTime(...)` in tests, immediately pair it with `defer clock.UnFreeze()` in the same scope so later assertions or subtests do not inherit frozen time accidentally.

When asserting `alpacadecimal.Decimal` equality in tests, prefer `require.Equal(t, expectedFloat64, actual.InexactFloat64())` over boolean assertions like `require.True(t, expected.Equal(actual))` when precision requirements allow it. Prefer simple `float64(5)`-style literals over verbose decimal construction for expected values. Inline one-off expected balance structs at the assertion site; name expected balances only when reused or when the name carries useful phase semantics across subtests.

After each meaningful test-related change, run focused `go vet` and focused `go test` for the touched package.

Examples:

```bash
nix develop --impure .#ci -c gofmt -w openmeter/ledger/historical/entry.go
nix develop --impure .#ci -c make lint-go
nix develop --impure .#ci -c env POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./openmeter/ledger/historical/...
```

| Command | Description |
|---------|-------------|
| `make test` | Run all tests (parallel: `-p 128 -parallel 16`) |
| `make test-nocache` | Run tests bypassing cache |
| `make test-all` | Run tests including Svix/Redis dependencies |
| `make test-go-sdk` | Build, vet, and test the v3 Go SDK module (`api/v3/client`) |
| `make etoe` | Run e2e tests (requires docker compose dependencies) |

**Running a single package directly:**

```bash
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/...
```

Key flags: `-tags=dynamic` (required for confluent-kafka-go), `-p 128 -parallel 16` (used by Make). Set `POSTGRES_HOST=127.0.0.1` or tests requiring Postgres will be skipped. `e2e/` is its own module and its import graph never reaches confluent-kafka-go, so `-tags=dynamic` is not needed there — `go test -C e2e ./...` (or `TZ=UTC OPENMETER_ADDRESS=... go test -C e2e ./...`) is enough.

See the `/test` skill for testing patterns, TestEnv setup, and examples.

## Building

```bash
make build              # All binaries → build/
make build-server       # Just the server
```

All builds use `GO_BUILD_FLAGS=-tags=dynamic`.

## Configuration

- Copy `config.example.yaml` to `config.yaml` (done automatically by Make targets)
- Load the repository environment with `direnv`, or run commands with `direnv exec . <command>`, so project-specific environment variables and tool configuration are applied consistently
- Key settings: `postgres.url`, `postgres.autoMigrate`, `billing`, `notification`, meter definitions
- `credits.enabled` needs explicit guarding at multiple layers: ledger-backed customer credit handlers in `api/v3/server`, customer ledger hooks, and namespace/default-account provisioning are wired separately and must each stay disabled when credits are off.
- When `credits.enabled` is `false`, `app/common` wires ledger account services/resolvers to noop implementations. Any ledger account backfill that must write real `ledger_accounts` / `ledger_customer_accounts` rows needs to construct concrete ledger account + resolver adapters directly instead of relying on the default DI outputs.
- Make targets for running services will warn if `config.yaml` is outdated vs `config.example.yaml`

## Coding Conventions

See the `/service` skill for service/adapter patterns, constructors, input types, errors, transactions, hooks, logging, multi-tenancy, and DI wiring. See the `/api` skill for HTTP handler patterns and ValidationIssue. See the `/ent` skill for Ent ORM patterns and Postgres type gotchas. See the `/ledger` skill for ledger package architecture, wiring, and testing. See the `/subscription` skill for subscription domain model, sync algorithm, patch system, workflow layer, and addon sub-system. See the `/notification` skill for notification event pipeline, Kafka consumers, Svix webhook delivery, reconciliation loop, and payload versioning.

For TypeSpec-specific coding constraints, update `api/spec/AGENTS.md` instead of adding them here.

### Documentation Constraints

- When adding comments or docstrings, document intent and domain constraints that are only available from human author context, not facts a reader can infer by reading the codebase. Avoid comments that merely translate obvious conditions, such as saying that a branch runs when `servicePeriod > 0`. A good comment should be understandable without the author's chat context and should explain why the code deliberately includes or excludes a case. For fallback or guard comments, name the concrete input shape or lifecycle state, the invariant being protected, the chosen behavior, and what would go wrong if the guard or fallback were removed. Avoid vague phrases like "can still arrive" or "as needed".
- Add a docstring to domain helpers when the name compresses important business semantics that are easy to misread at call sites. Explain the observable business contract and why excluded cases are excluded, not the implementation mechanics.
- When refactoring or reverting code, preserve existing explanatory comments by default. Remove or rewrite a comment only when the code change makes it false, stale, or misleading.

### Go Style Constraints

- For Go string enum constants, name values as `<Type><Value>` so the constant carries its enum type at the use site, for example `InvoiceStatusDraft` instead of `Draft`.
- Do not extract helper functions only to hide a couple of simple operations or short guard checks. If the helper would only wrap 2-4 lines and its name does not add meaningful domain or business intent, keep the code inline even when there is some duplication. Readers can inspect the function body to see what the code does; prefer function names that explain the domain reason for the call over names that merely restate the implementation steps. When you encounter a leftover pass-through wrapper that only calls another function without adding behavior, remove it and call the underlying function directly, even if it is outside the immediate change area.
- Do not hide non-trivial branching or domain translation inside local inline functions. If a closure performs type switching, validation, persistence mapping, or meaningful domain conversion, make it a named helper near the code that uses it so it is discoverable, testable, and grep-friendly. Reserve inline closures for tiny callbacks where the surrounding API requires a function literal and the logic is obvious at the call site.
- For `Validate() error` methods, prefer collecting all validation issues into `var errs []error` and returning `models.NewNillableGenericValidationError(errors.Join(errs...))` instead of returning on the first invalid field. Preserve field context with wrapped errors like `fmt.Errorf("field: %w", err)` and use plain `errors.New(...)` for simple local checks.
- Do not introduce `context.Background()` or `context.TODO()` to sidestep missing context propagation in application code. Either propagate the caller's context through the full call path, or remove the unused `context.Context` parameter from the API if the operation is purely local and does not need cancellation, deadlines, or request-scoped values.
- Never use `panic` in non-test code paths. If a new failure mode is possible, change the function signature to return an error and propagate it explicitly.
- In production constructors and initialization, do not use `slog.Default()` as a fallback dependency. Require a `*slog.Logger` in config/provider inputs and inject it explicitly.
- Prefer standard library `slices` and `maps` helpers for common collection operations, and use `github.com/samber/lo` when it makes pointer literals or collection transformations clearer than local wrappers or hand-written loops. See the `/samber-lo` skill for common OpenMeter use cases and caveats. Do not add local wrappers such as `ptr`, `loPtr`, `must`, or `loMust` when standard helpers or `lo` already cover the need.
- Use repo helper packages when they capture a common pattern better than ad hoc closures. For example, use `pkg/slicesx` for existing slice helpers (but prefer `samber/lo` and `slices` system packages if they fit), and use `pkg/syncx.OnceValues` for lazy context-aware database lookups that may be needed by multiple callbacks but should execute at most once.
- Keep helper functions honest and narrow. If a production helper is only called once and is just a short guard or a few straightforward lines, inline it unless the name carries meaningful domain semantics. Do not add helpers for trivial single-use struct literals, do not hide aggregate mutation inside construction helpers, and return the domain value a helper actually builds rather than a broader wrapper needed by one caller.
- For files and functions that convert between domain, API, and DB representations, use the `/go-types-conversion` skill. In prose, prefer `map` / `mapped` terminology for domain representation translation and avoid `project` / `projected` for that meaning; function names must still follow the skill's `FromAPI...`, `ToAPI...`, `FromDB...`, and `ToDB...` conventions.

### Generation And Dependency Constraints

- When `make generate` or `atlas migrate --env local diff ...` adds incidental `go.sum` entries, such as `tablewriter`, drop those `go.sum` changes unless the task explicitly requires a dependency change.

## Key Dependencies

| Category | Libraries |
|----------|-----------|
| DB | PostgreSQL (Ent ORM, Atlas migrations, pgx driver) |
| Analytics | ClickHouse |
| Events | Kafka (confluent-kafka-go) + Watermill |
| HTTP | Chi router + oapi-codegen |
| Invoicing | GOBL (invoice format) |
| Webhooks | Svix |
| Observability | OpenTelemetry |
| Config | Viper + Cobra |
| Utilities | samber/lo |

## CodeGraph

CodeGraph builds a semantic knowledge graph of the codebase (~1,800 Go files, ~36k symbols) for faster, smarter code exploration. The index lives in `.codegraph/codegraph.db` (gitignored). Generated files (`ent/db/`, `*_gen.go`, `wire_gen.go`, `*.gen.go`) are excluded.

### If `.codegraph/` exists

**Default to CodeGraph, not Grep/Glob/find.** CodeGraph understands symbols, call relationships, and file structure — those tools return string matches. On a ~1,800-file Go codebase the symbol-aware answer is almost always what you wanted. Fall back to Grep/Glob **only** when CodeGraph returns no results or the query is inherently textual (string literals, comments, log messages, SQL, YAML keys).

**Never call `codegraph_explore` or `codegraph_context` in the main session.** These tools return large source code blocks that fill up main-session context fast. Instead, spawn an Explore agent for any exploration question (e.g., "how does billing sync work?", "where is entitlement reset implemented?").

When spawning Explore agents, include this instruction in the prompt:

> This project has CodeGraph initialized (.codegraph/ exists). Use `codegraph_explore` as your PRIMARY exploration tool — it returns full source code sections from all relevant files in one call.
>
> **Rules:**
> 1. Follow the explore call budget in the `codegraph_explore` tool description — it scales automatically based on project size.
> 2. Do NOT re-read files that `codegraph_explore` already returned source code for. The source sections are complete and authoritative.
> 3. Only fall back to Grep/Glob/Read for files listed under "Additional relevant files" if you need more detail, or if CodeGraph returned no results.

**The main session should use these lightweight tools directly** for targeted lookups before making edits:

| Tool | Use for | Example |
|------|---------|---------|
| `codegraph_search` | Find symbols by name | `query: "BillingService"` |
| `codegraph_callers` | Who calls this function? | Before renaming or changing a signature |
| `codegraph_callees` | What does this function call? | Understanding a function's dependencies |
| `codegraph_impact` | Blast radius of a change | Before refactoring a shared type |
| `codegraph_node` | Single symbol details | Quick check on a struct or interface |
| `codegraph_files` | Project file tree | Faster than Glob for directory overviews |
| `codegraph_status` | Index health check | Verify the index is up to date |

### Choosing CodeGraph vs Grep/Glob

| Task | Prefer | Reason |
|------|--------|--------|
| "Where is `BillingService` defined?" | `codegraph_search` | Symbol lookup with location + signature |
| "Who calls `ListCustomers`?" | `codegraph_callers` | Call-graph edges, not text matches |
| "What does `Reconcile` call?" | `codegraph_callees` | Dependencies of a function |
| "Blast radius of changing this type?" | `codegraph_impact` | Transitive reverse-deps |
| "Show the `Filter` interface fields" | `codegraph_node` | Single-symbol detail without reading whole file |
| "List files under `api/v3/filters/`" | `codegraph_files` | Indexed tree; no disk walk |
| Find a string literal / log message / SQL fragment | `Grep` | Not a symbol |
| Find files by glob pattern (`**/*.tsp`) | `Glob` | CodeGraph indexes Go; non-Go globs go through Glob |
| Navigate a specific known path | `Read` | Direct reads are always fine |
| Running `find` on the shell | Don't | Use `codegraph_files` or `Glob` |
| Running `grep`/`rg` on the shell | Don't | Use `Grep` (or `codegraph_search` for symbols) |

Rule of thumb: **if the target is a Go identifier, start with CodeGraph. If it's a string, start with Grep.** Never shell out to `grep`, `rg`, or `find` — the dedicated tools (`Grep`, `Glob`, `codegraph_*`) give better output and permission handling.

### Keeping the index fresh

At the start of work, refresh CodeGraph before exploring Go code. If `.codegraph/` exists, run `codegraph sync`; if it does not exist, run `codegraph init -i` without asking first. Run `codegraph index` for a full rebuild if the index seems stale or after branch switches.

### If `.codegraph/` does NOT exist

Initialize it with `codegraph init -i` before doing code exploration. It indexes the Go codebase quickly and keeps symbol-aware lookup available.

## Skills

Skills are created inside [.agents/skills](.agents/skills/) by default and then symlinked to [.claude/skills](.claude/skills). Make sure you always treat `.agents/skills` as the source of truth. Keep skill guidance compatible with both Claude and Codex; avoid instructions that assume only one agent runtime unless the skill is explicitly about that runtime.
