# backfillaccounts

<!-- archie:ai-start -->

> Single-file Cobra sub-command that backfills missing ledger accounts (business and customer) for a namespace. Intentionally bypasses the Wire DI surface to construct concrete ledger adapters directly, because the wired accountResolver is a noop when credits.enabled=false.

## Patterns

**Bypass DI for concrete ledger stack** — Construct accountRepo, accountSvc, resolverRepo, and accountResolver directly from internal.App.EntClient. Never use the wired resolver from app/common — it is a noop when credits are disabled. (`accountRepo := accountadapter.NewRepo(internal.App.EntClient); accountSvc := accountservice.New(accountRepo, locker); resolverRepo := resolversadapter.NewRepo(internal.App.EntClient); accountResolver := resolvers.NewAccountResolver(...)`)
**Flags in init(), logic in RunE** — All flag vars are package-level, registered in init() with Cmd.Flags(). RunE constructs the service, resolves inputs from flags, calls service.Run, prints the summary, then checks FailureCount for a non-zero exit. (`func init() { Cmd.Flags().StringVar(&createdBefore, "created-before", "", "...") }`)
**RFC3339 time parsing for flag values** — Time-valued flags are passed as strings and parsed with time.Parse(time.RFC3339, ...) in RunE, not in flag.Value; always normalize to UTC. (`parsed, parseErr := time.Parse(time.RFC3339, createdBefore); parsed = parsed.UTC(); createdBeforeTime = &parsed`)
**Print summary before returning error** — printSummary is always called before any error return so partial results are visible even on failure. (`if err != nil { printSummary(output); return err }; printSummary(output)`)
**Non-zero exit on FailureCount** — After printing the summary, check output.Result.FailureCount and return a descriptive error so the process exits non-zero when any customer provisioning failed. (`if output.Result.FailureCount > 0 { return fmt.Errorf("backfill completed with %d failures", output.Result.FailureCount) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `backfillaccounts.go` | Single-file Cobra command: package-level flag vars, init() registration, newService() that builds concrete adapters, RunE wiring, and printSummary. | internal.App is the global app struct from cmd/jobs/internal. Do not import app/common wired services — the wired accountResolver is a noop when credits.enabled=false. lockr.NewLocker must be created here and passed to accountservice.New. |

## Anti-Patterns

- Using the Wire-provided ledger account resolver from app/common (it is a noop when credits.enabled=false)
- Omitting lockr.NewLocker — accountservice.New requires a Locker for CreateCustomerAccounts
- Parsing time flags outside RunE or omitting UTC normalization
- Returning an error before printSummary (partial results become invisible)
- Adding business logic inside RunE beyond input construction, service.Run invocation, and summary printing

## Decisions

- **Build concrete ledger adapters directly instead of using DI outputs** — AGENTS.md explicitly states: 'When writing a backfill that genuinely needs ledger writes, build the concrete adapters directly instead of relying on DI defaults (they are noops when credits.enabled=false).'
- **Delegate all pagination and provisioning logic to cmd/jobs/ledger/service** — Keeps the Cobra command thin and testable: the service package contains pure-Go logic with no Cobra/Ent imports, enabling unit tests without a real database.

## Example: Construct the concrete ledger stack and run the backfill from RunE

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
// ...
```

<!-- archie:ai-end -->
