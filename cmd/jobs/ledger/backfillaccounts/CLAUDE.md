# backfillaccounts

<!-- archie:ai-start -->

> Cobra sub-command that backfills missing ledger accounts (business and customer) for a namespace. Intentionally bypasses the DI-wired account resolver to access CreateCustomerAccounts, which is not exposed via the public Wire surface.

## Patterns

**Bypass DI for concrete ledger stack** — Because app/common wires ledger services to noops when credits are disabled, and the public resolver surface omits CreateCustomerAccounts, this command constructs accountRepo, accountSvc, resolverRepo, and accountResolver directly from internal.App.EntClient. Do not use the wired resolver from app/common here. (`accountRepo := accountadapter.NewRepo(internal.App.EntClient); accountSvc := accountservice.New(accountRepo, ledgeraccount.AccountLiveServices{Locker: locker, Querier: ledgernoop.Ledger{}})`)
**Flags in init(), logic in RunE** — All flag vars are package-level, registered in init() with Cmd.Flags(). RunE constructs the service, resolves inputs from flags, calls service.Run, prints the summary, then checks FailureCount for a non-zero exit. (`func init() { Cmd.Flags().StringVar(&createdBefore, "created-before", "", "...") }`)
**RFC3339 time parsing for flag values** — Time-valued flags are passed as strings and parsed with time.Parse(time.RFC3339, ...) in RunE, not in flag.Value; always normalize to UTC. (`parsed, parseErr := time.Parse(time.RFC3339, createdBefore); parsed = parsed.UTC()`)
**Print summary before returning error** — printSummary is called before any error return so partial results are always visible even on failure. (`if err != nil { printSummary(output); return err }; printSummary(output)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `backfillaccounts.go` | Single-file Cobra command: flag definitions, newService constructor that bypasses DI, and RunE wiring. | internal.App is the global shared app struct from cmd/jobs/internal; do not import app/common wired services here — the wired accountResolver is a noop when credits are disabled. |

## Anti-Patterns

- Using the Wire-provided ledger account resolver from app/common — it is a noop when credits.enabled=false
- Omitting lockr.NewLocker — the accountservice requires a Locker for CreateCustomerAccounts
- Parsing time flags outside RunE or omitting UTC normalization
- Returning an error before printSummary — partial results become invisible

## Decisions

- **Build concrete ledger adapters directly instead of using DI outputs** — The AGENTS.md explicitly states: 'When writing a backfill that genuinely needs ledger writes, build the concrete adapters directly instead of relying on DI defaults (they are noops).'

<!-- archie:ai-end -->
