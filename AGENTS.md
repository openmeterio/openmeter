# OpenMeter

## Repository tooling

- Use the `Makefile` for common development tasks. The `justfile` is secondary.
- When tools are missing from the ambient shell, use:

  ```bash
  nix develop --impure .#ci -c <command>
  ```

- Always invoke `nix develop` from the repository root. Entering from a
  subdirectory writes CWD-relative devenv files and can break formatting or the
  pre-commit hook.
- Never run multiple `nix develop` commands concurrently in one worktree; they
  mutate shared `.devenv/` state.
- `.nvmrc` is the GitHub Actions source of truth for Node. Keep it aligned with
  `node -v` in the Nix `.#ci` shell.

## Repository invariants

- The repository has three Go modules: the production root, the publishable
  `api/v3/client` SDK, and the test-only `e2e` module.
- The root module must never require `api/v3/client`. The nested SDK is not
  independently tagged, and local `replace` directives are invisible to
  downstream consumers.
- Root `go build ./...`, `go test ./...`, and `go vet ./...` exclude `e2e`.
  Use `make etoe` or `go test -C e2e ./...`; `make lint-go` and `make mod`
  already cover all three modules.

## Go conventions

- Name string enum constants `<Type><Value>`, for example
  `InvoiceStatusDraft`.
- Prefer methods when behavior is intrinsic to an existing domain type. Use
  freestanding functions when an operation has
  no natural receiver.
- Do not extract trivial or single-use helpers unless the name captures
  non-obvious domain intent. Inline pass-through wrappers.
- Do not hide type switching, validation, persistence mapping, or meaningful
  domain translation inside local closures. Use a named helper; reserve inline
  callbacks for obvious, tiny logic.
- For `Validate() error`, collect field errors and return
  `models.NewNillableGenericValidationError(errors.Join(errs...))`. Preserve
  field context with wrapped errors.
- Do not introduce `context.Background()` or `context.TODO()` to bypass context
  propagation. Propagate the caller context or remove an unnecessary context
  parameter.
- Never use `panic` in non-test code. Return and propagate errors.
- Production constructors must receive a `*slog.Logger`; do not fall back to
  `slog.Default()`.
- Prefer standard `slices` and `maps`, then `github.com/samber/lo` where it is
  clearer. Reuse `pkg/slicesx` and `pkg/syncx.OnceValues`; do not add local
  `ptr`, `must`, or equivalent wrappers.
- Mapping helpers between API, domain, and DB representations follow
  [Type Translation Naming](docs/development/type-conversions.md). Use
  `map`/`mapped`, not `project`/`projected`, for representation translation.

## Testing conventions

- PostgreSQL-backed tests require PostgreSQL and
  `POSTGRES_HOST=127.0.0.1`; otherwise many suites silently skip.
- Root-package tests using `confluent-kafka-go` require `-tags=dynamic`. The
  standalone `e2e` module does not.
- Keep domain test helpers under `openmeter/.../testutils` independent from
  `app/common`; construct repositories, adapters, services, and locks from
  their package constructors.
- Prefer production-backed/service-backed fixtures when the real path can
  express the scenario. Hardcode suite-wide behavior into the shared harness
  instead of exposing per-test knobs.
- Use `t.Context()` when a `testing.T` or `testing.TB` is available.
- Add a test helper only when reused or when its name captures non-obvious
  domain semantics. Inline one-use setup/assertion wrappers and remove dead
  helpers.
- Begin non-trivial service/lifecycle subtests with concise `given`, `when`,
  and `then` intent comments.
- Pair `clock.FreezeTime(...)` immediately with `defer clock.UnFreeze()` in the
  same scope.
- When precision permits, compare `alpacadecimal.Decimal` through
  `InexactFloat64()` with `require.Equal`; inline one-off expected balances.

See [Testing](docs/development/testing.md) for harness patterns and examples.

## Documentation conventions

- Comments and docstrings explain intent, domain constraints, lifecycle state,
  and failure consequences that are not obvious from the code. Do not narrate
  conditions the reader can already see.
- Document domain helpers whose names compress important business semantics;
  describe observable behavior and why excluded cases are excluded.
- Preserve explanatory comments during refactors unless the change makes them
  false or misleading.
- Keep project documentation short enough that humans will read it.

## Further guidance

- TypeSpec and SDK generator work follows [api/spec/AGENTS.md](api/spec/AGENTS.md).
- Focused developer guides cover [service packages](docs/development/service-patterns.md),
  [testing](docs/development/testing.md),
  [refactoring](docs/development/refactoring.md),
  [type conversions](docs/development/type-conversions.md), and
  [collection helpers](docs/development/collection-helpers.md).

For changes or reviews that affect domain behavior, use the
[`domain-docs`](.agents/skills/domain-docs/SKILL.md) skill after establishing
the scope to locate the relevant package documentation. Treat domain docs as
intended semantics and code and tests as current behavior; investigate
discrepancies instead of automatically preferring either.

## CodeGraph and complex work

- Use CodeGraph when it materially shortens symbol, caller, flow, or impact
  exploration. If the worktree is not initialized, run
  `nix develop --impure .#ci -c codegraph init -i` from the repository root,
  then retry. Check `codegraph status`; if initialization left a truncated
  index, rebuild it with `nix develop --impure .#ci -c codegraph index` before
  falling back. Its server keeps a healthy index synchronized; do not manually
  sync it.
- Treat graph relationships as navigation hints. Current source and tests are
  authoritative; fall back to normal source navigation only when initialization
  fails, CodeGraph is unavailable, or its results are inconclusive.
- When planning or implementing ambiguous, design-heavy, or cross-cutting
  engineering changes, use the
  [`iterative-engineering-design`](.agents/skills/iterative-engineering-design/SKILL.md)
  skill.

## Maintaining agent guidance

- Keep automatically required repo guidance in this file. Use skills for
  task-shaped procedures and explicitly linked docs for scoped detail; do not
  assume root-started agents will discover descendant `AGENTS.md` files.
- Fold new guidance into the relevant section and merge or remove duplicates.
- `.agents/skills` is the source of truth for skills; `.claude/skills` contains
  compatibility symlinks. Keep skill text usable by both Codex and Claude.
