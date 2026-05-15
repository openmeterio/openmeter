# advance

<!-- archie:ai-start -->

> Batch auto-advance loop for StandardInvoices: polls three disjoint invoice states (DraftWaitingAutoApproval, DraftWaitingForCollection, stuck-advanceable), merges and deduplicates the lists, then fans out parallel goroutines in lo.Chunk batches to call billing.Service.AdvanceInvoice. Acts as the scheduled polling counterpart to the event-driven asyncadvance handler.

## Patterns

**Config struct constructor with nil validation** — NewAdvancer accepts a Config struct and returns error if BillingService or Logger is nil. All new workers in this package must follow this pattern. (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.BillingService == nil { return nil, fmt.Errorf("billing service is required") } }`)
**Batched parallel fan-out with errChan + sync.WaitGroup** — lo.Chunk splits work into batches; each batch spawns goroutines writing to a buffered errChan sized to total invoice count. sync.OnceFunc closes the channel after all batches complete. errors.Join collects all failures. (`errChan := make(chan error, len(invoices)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)
**ErrInvoiceCannotAdvance is a non-fatal sentinel** — When AdvanceInvoice returns billing.ErrInvoiceCannotAdvance, fetch invoice details, log a warning with status/draftUntil fields, and return nil — never propagate as an error. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { a.logger.WarnContext(ctx, "invoice cannot be advanced", logArgs...); return invoice, nil }`)
**List consolidation with lo.UniqBy before batching** — ListInvoicesToAdvance merges three separate list calls and deduplicates by invoice ID using lo.UniqBy before batching to prevent double-advances. (`return lo.UniqBy(allInvoices, func(i billing.StandardInvoice) string { return i.ID }), nil`)
**Context with cancel wraps batch runs** — All() immediately creates a cancellable child context so in-flight goroutines can be cancelled if the parent is cancelled. (`ctx, cancel := context.WithCancel(ctx); defer cancel()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Entire package — AutoAdvancer struct with All(), ListInvoicesToAdvance(), ListInvoicesPendingAutoAdvance(), ListInvoicesPendingCollection(), ListStuckInvoicesNeedingAdvance(), AdvanceInvoice(), and NewAdvancer(). | batchSize=0 means single batch (all invoices in one pass). errChan capacity must equal len(invoices) not len(batches) or sends will block. ErrInvoiceCannotAdvance must remain a warning return (nil), not an error propagation. |

## Anti-Patterns

- Calling billing.Adapter directly — all DB access must go through billing.Service
- Propagating billing.ErrInvoiceCannotAdvance as an error return from AdvanceInvoice
- Forgetting lo.UniqBy deduplication on the merged invoice list, causing double-advances
- Using context.Background() inside goroutines instead of the caller-supplied ctx
- Sizing errChan to len(batches) instead of len(invoices) — goroutine sends will deadlock

## Decisions

- **Three separate list queries merged with lo.UniqBy deduplication instead of a single compound query** — DraftWaitingAutoApproval, DraftWaitingForCollection, and stuck-advanceable invoices have distinct filter shapes in ListStandardInvoicesInput; merging at the query level would require OR semantics the adapter does not support cleanly.
- **Batched parallel goroutines rather than sequential advance** — Invoice advancement is per-invoice and independent; parallelism reduces wall-clock time for large namespaces while batchSize controls memory pressure.

## Example: Advance all eligible invoices in batches of 50

```
import billingworkeradvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"

advancer, err := billingworkeradvance.NewAdvancer(billingworkeradvance.Config{
    BillingService: svc,
    Logger:         logger,
})
if err != nil {
    return err
}
return advancer.All(ctx, []string{"default"}, 50)
```

<!-- archie:ai-end -->
