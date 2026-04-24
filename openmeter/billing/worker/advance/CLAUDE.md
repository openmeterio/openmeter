# advance

<!-- archie:ai-start -->

> Batch auto-advance loop for StandardInvoices: lists invoices in DraftWaitingAutoApproval, DraftWaitingForCollection, and stuck-advanceable states, then fans out parallel goroutines (batched) to call billing.Service.AdvanceInvoice. Acts as the scheduled/polling counterpart to the event-driven asyncadvance handler.

## Patterns

**Config struct constructor with validation** — NewAdvancer accepts a Config struct; returns error if BillingService or Logger is nil. All new workers in this package must follow this pattern. (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.BillingService == nil { return nil, fmt.Errorf(...) } }`)
**Batched parallel fan-out with errChan + sync.WaitGroup** — lo.Chunk splits work into batches; each batch spawns goroutines writing to a buffered errChan. sync.OnceFunc closes the channel after all batches complete. errors.Join collects all failures. (`errChan := make(chan error, len(invoices)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)
**ErrInvoiceCannotAdvance is a non-fatal sentinel** — When AdvanceInvoice returns billing.ErrInvoiceCannotAdvance, log a warning with invoice details and return nil — do not propagate as an error. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { a.logger.WarnContext(ctx, ...) ; return invoice, nil }`)
**List consolidation with lo.UniqBy** — ListInvoicesToAdvance merges three separate list calls and deduplicates by invoice ID using lo.UniqBy before batching. (`return lo.UniqBy(allInvoices, func(i billing.StandardInvoice) string { return i.ID }), nil`)
**Context with cancel for batch runs** — All() immediately creates a cancellable child context so in-flight goroutines can be cancelled if the parent is cancelled. (`ctx, cancel := context.WithCancel(ctx); defer cancel()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Entire package — AutoAdvancer struct with All(), ListInvoicesToAdvance(), ListInvoicesPendingAutoAdvance(), ListInvoicesPendingCollection(), ListStuckInvoicesNeedingAdvance(), AdvanceInvoice(), and NewAdvancer(). | batchSize=0 means single batch (all invoices in one pass). errChan capacity must equal len(invoices), not len(batches), or sends will block. ErrInvoiceCannotAdvance must remain a warning, not an error. |

## Anti-Patterns

- Calling billing.Adapter directly — all DB access must go through billing.Service
- Propagating billing.ErrInvoiceCannotAdvance as an error return from AdvanceInvoice
- Forgetting to deduplicate the merged invoice list before batching, causing double-advances
- Using context.Background() instead of the caller-supplied ctx in goroutines

## Decisions

- **Three separate list queries merged with deduplication instead of a single compound query** — Waiting-auto-approval, waiting-for-collection, and stuck invoices have distinct filter shapes in billing.ListStandardInvoicesInput; merging at the query level would require OR semantics the adapter does not support cleanly.
- **Batched parallel goroutines rather than sequential advance** — Invoice advancement is per-invoice and independent; parallelism is safe and reduces wall-clock time for large namespaces.

## Example: Advance all eligible invoices in batches of 50

```
advancer, _ := billingworkeradvance.NewAdvancer(billingworkeradvance.Config{
    BillingService: svc,
    Logger:         logger,
})
if err := advancer.All(ctx, []string{"default"}, 50); err != nil {
    return err
}
```

<!-- archie:ai-end -->
