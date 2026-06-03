# backfillaccounts

<!-- archie:ai-start -->

> Single-file Cobra sub-command that backfills missing ledger accounts (business + customer) for the default namespace. Intentionally bypasses the Wire DI surface to construct concrete ledger adapters, because the wired accountResolver is a noop when credits.enabled=false.

## Patterns

**Bypass DI for the concrete ledger stack** — newService() builds accountRepo, accountSvc, resolverRepo, and accountResolver directly from internal.App.EntClient; never uses the wired resolver from app/common (a noop / narrowed surface when credits are disabled). (`accountRepo := accountadapter.NewRepo(internal.App.EntClient); accountSvc := accountservice.New(accountRepo, locker); accountResolver := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{AccountService: accountSvc, Repo: resolverRepo, Locker: locker})`)
**Flags in init(), logic in RunE** — Package-level flag vars (createdBefore, customerPageSize, dryRun, continueOnError, includeDeleted) are registered in init() via Cmd.Flags(); RunE builds the service, resolves inputs, calls service.Run, prints summary, then exits non-zero on failures. (`func init() { Cmd.Flags().StringVar(&createdBefore, "created-before", "", "...") }`)
**RFC3339 flag parsing normalized to UTC** — Time-valued flags are strings parsed with time.Parse(time.RFC3339, ...) inside RunE and normalized to UTC. (`parsed, err := time.Parse(time.RFC3339, createdBefore); parsed = parsed.UTC(); createdBeforeTime = &parsed`)
**Print summary before returning error** — printSummary is always called before any error return so partial results stay visible even on failure. (`if err != nil { printSummary(output); return err }; printSummary(output)`)
**Non-zero exit on FailureCount** — After printing the summary, return a descriptive error when output.Result.FailureCount > 0 so the process exits non-zero. (`if output.Result.FailureCount > 0 { return fmt.Errorf("backfill completed with %d failures", output.Result.FailureCount) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `backfillaccounts.go` | Single-file Cobra command: flag vars, init() registration, newService() building concrete ledger adapters, RunE orchestration over cmd/jobs/ledger/service, and printSummary. | Use internal.App globals; do not import wired app/common ledger services (noop when credits disabled). lockr.NewLocker must be created here and passed to accountservice.New (required for CreateCustomerAccounts). Namespace comes from internal.App.NamespaceManager.GetDefaultNamespace(). |

## Anti-Patterns

- Using the Wire-provided ledger account resolver from app/common (noop when credits.enabled=false)
- Omitting lockr.NewLocker — accountservice.New requires a Locker
- Parsing time flags outside RunE or skipping UTC normalization
- Returning an error before printSummary (partial results become invisible)
- Adding business logic in RunE beyond input construction, service.Run, and summary printing

## Decisions

- **Build concrete ledger adapters directly instead of using DI outputs** — AGENTS.md states backfills that genuinely need ledger writes must build concrete adapters directly because DI defaults are noops when credits.enabled=false.
- **Delegate pagination/provisioning logic to cmd/jobs/ledger/service** — Keeps the Cobra command thin and testable: the service package is pure Go with no Cobra/Ent imports, enabling unit tests without a real database.

## Example: Construct the concrete ledger stack for the backfill

```
import (
  accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
  accountservice "github.com/openmeterio/openmeter/openmeter/ledger/account/service"
  "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
  resolversadapter "github.com/openmeterio/openmeter/openmeter/ledger/resolvers/adapter"
  "github.com/openmeterio/openmeter/pkg/framework/lockr"
)
func newService() (*ledgerbackfillservice.Service, error) {
  locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: internal.App.Logger})
  if err != nil { return nil, fmt.Errorf("create locker: %w", err) }
  accountRepo := accountadapter.NewRepo(internal.App.EntClient)
  accountSvc := accountservice.New(accountRepo, locker)
  resolverRepo := resolversadapter.NewRepo(internal.App.EntClient)
  accountResolver := resolvers.NewAccountResolver(resolvers.AccountResolverConfig{
    AccountService: accountSvc, Repo: resolverRepo, Locker: locker,
// ...
```

<!-- archie:ai-end -->
