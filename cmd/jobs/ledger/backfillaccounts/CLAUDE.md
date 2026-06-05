# backfillaccounts

<!-- archie:ai-start -->

> Cobra subcommand (`backfill-accounts`) of the `jobs` binary that provisions customer and business ledger accounts for the default namespace. It is the wiring/CLI shell that constructs a concrete ledger account-resolver stack and delegates all logic to `cmd/jobs/ledger/service.Service.Run`.

## Patterns

**Cobra command + RunE delegation** — Expose a package-level `var Cmd = &cobra.Command{...}` with flags bound in `init()` and a `RunE` that builds the service, reads `internal.App.NamespaceManager.GetDefaultNamespace()`, calls `service.Run`, prints a summary, and returns a non-nil error when `FailureCount > 0`. (`Cmd.RunE calls newService(), then service.Run(cmd.Context(), ledgerbackfillservice.RunInput{...})`)
**Concrete resolver stack construction (bypass wired DI)** — Build the ledger account stack manually from adapters/services because the wired public account-resolver surface is narrowed and does not expose CreateCustomerAccounts: accountadapter.NewRepo -> accountservice.New(repo, locker) -> resolversadapter.NewRepo -> resolvers.NewAccountResolver(...). (`accountResolver := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{AccountService: accountSvc, Repo: resolverRepo, Locker: locker})`)
**Shared job runtime via internal.App** — Pull EntClient, Logger, and NamespaceManager from the shared `cmd/jobs/internal.App` rather than wiring a new application; construct a fresh `lockr.NewLocker` with `internal.App.Logger`. (`accountRepo := accountadapter.NewRepo(internal.App.EntClient)`)
**RFC3339 flag parsing to UTC pointer** — Parse the `--created-before` string flag with `time.Parse(time.RFC3339, ...)`, normalize to UTC, and pass a `*time.Time` (nil when empty) into RunInput.CreatedBefore. (`parsed = parsed.UTC(); createdBeforeTime = &parsed`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `backfillaccounts.go` | Defines `Cmd`, its flags (created-before, customer-page-size, dry-run, continue-on-error, include-deleted), `newService()` constructor, and `printSummary()`. | newService deliberately constructs the concrete resolver stack rather than using wired DI (the comment explains CreateCustomerAccounts is not on the public surface); do not replace it with the default wired resolver. printSummary is always called even on the error path before returning. |

## Anti-Patterns

- Calling lower-level account adapters/services directly from the command instead of going through ledgerbackfillservice.Service.Run.
- Using the wired/narrowed account resolver from DI (it lacks CreateCustomerAccounts) instead of the concrete resolvers.NewAccountResolver stack.
- Defaulting customer-page-size to a literal here instead of ledgerbackfillservice.DefaultCustomerPageSize.
- Swallowing FailureCount: RunE must return an error when output.Result.FailureCount > 0.

## Decisions

- **Build the concrete account+resolver stack in newService() instead of relying on Wire DI.** — The public wired account-resolver surface is intentionally narrowed and does not expose CreateCustomerAccounts, which the backfill requires.

<!-- archie:ai-end -->
