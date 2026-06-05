# advance

<!-- archie:ai-start -->

> Batch/cron driver (package billingworkeradvance) that finds standard invoices eligible for auto-advancement or collection and advances them concurrently. It is a thin orchestration layer over billing.Service.AdvanceInvoice — it owns no persistence and no state machine.

## Patterns

**Service-only dependency via Config + NewAdvancer** — AutoAdvancer wraps a single billing.Service plus a *slog.Logger. Construct via NewAdvancer(Config{...}) which nil-checks BillingService and Logger and returns an error; never build the struct literal directly. (`a, err := NewAdvancer(Config{BillingService: svc, Logger: log})`)
**Three-source candidate union** — ListInvoicesToAdvance composes ListInvoicesPendingAutoAdvance (DraftWaitingAutoApproval + DraftUntilLTE), ListInvoicesPendingCollection (DraftWaitingForCollection + CollectionAtLTE), and ListStuckInvoicesNeedingAdvance (HasAvailableAction=Advance, a fail-safe), then dedupes with lo.UniqBy on invoice.ID. (`lo.UniqBy(allInvoices, func(i billing.StandardInvoice) string { return i.ID })`)
**Concurrent per-batch fan-out with buffered errChan** — All() chunks invoices via lo.Chunk(batchSize), spawns a goroutine per invoice inside a sync.WaitGroup per batch, pushes each result to a buffered errChan, and joins via errors.Join after a sync.OnceFunc close. (`errChan := make(chan error, len(invoices)); closeErrChan := sync.OnceFunc(func(){ close(errChan) })`)
**ErrInvoiceCannotAdvance is swallowed, not failed** — AdvanceInvoice treats billing.ErrInvoiceCannotAdvance as benign: it re-fetches via GetStandardInvoiceById, logs a WarnContext with invoice id/namespace/status/draft_until, and returns nil error so the next run retries. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { ... return invoice, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Entire package: AutoAdvancer struct, the four List* methods, AdvanceInvoice wrapper, All() batch runner, Config + NewAdvancer constructor. | All() reuses the loop variable `err` across goroutines (`_, err = a.AdvanceInvoice(...)`) — a known data-race shape; do not copy this pattern into new concurrent code, scope err inside the goroutine. Also the field is named `invoice` but holds a billing.Service, not an invoice. |

## Anti-Patterns

- Adding persistence/state-machine logic here — advancement transitions belong in billing.Service; this package only lists and dispatches.
- Treating ErrInvoiceCannotAdvance as a hard failure — it must remain a logged warning that returns nil so the cron retries.
- Constructing AutoAdvancer without NewAdvancer, bypassing the nil checks on BillingService/Logger.

## Decisions

- **Union three candidate queries (auto-approve, collection, stuck) instead of one.** — The HasAvailableAction=Advance query is an explicit fail-safe to recover invoices that got stuck in an advanceable state outside the normal time-based triggers.

<!-- archie:ai-end -->
