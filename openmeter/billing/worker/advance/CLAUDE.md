# advance

<!-- archie:ai-start -->

> Batch auto-advance loop for StandardInvoices: polls three disjoint invoice states (DraftWaitingAutoApproval, DraftWaitingForCollection, stuck-advanceable), merges/deduplicates, then fans out parallel goroutines in lo.Chunk batches to call billing.Service.AdvanceInvoice. The scheduled polling counterpart to the event-driven asyncadvance handler.

## Patterns

**Config struct constructor with nil validation** — NewAdvancer takes a Config and errors if BillingService or Logger is nil. (`func NewAdvancer(config Config) (*AutoAdvancer, error) { if config.BillingService == nil { return nil, fmt.Errorf("billing service is required") } }`)
**Batched parallel fan-out with errChan + WaitGroup** — lo.Chunk splits work into batches; each batch spawns goroutines writing to a buffered errChan sized to total invoice count; sync.OnceFunc closes the channel; errors.Join collects failures. (`errChan := make(chan error, len(invoices)); closeErrChan := sync.OnceFunc(func() { close(errChan) }); defer closeErrChan()`)
**ErrInvoiceCannotAdvance is a non-fatal sentinel** — When AdvanceInvoice returns billing.ErrInvoiceCannotAdvance, fetch details, log a warning, and return nil — never propagate. (`if errors.Is(err, billing.ErrInvoiceCannotAdvance) { a.logger.WarnContext(ctx, "invoice cannot be advanced", logArgs...); return invoice, nil }`)
**List consolidation with lo.UniqBy before batching** — ListInvoicesToAdvance merges three list calls and deduplicates by invoice ID before batching to prevent double-advances. (`return lo.UniqBy(allInvoices, func(i billing.StandardInvoice) string { return i.ID }), nil`)
**Context with cancel wraps batch runs** — All() creates a cancellable child context so in-flight goroutines cancel when the parent cancels. (`ctx, cancel := context.WithCancel(ctx); defer cancel()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Entire package — AutoAdvancer with All(), ListInvoicesToAdvance(), ListInvoicesPendingAutoAdvance/PendingCollection/StuckInvoicesNeedingAdvance(), AdvanceInvoice(), NewAdvancer(). | batchSize=0 means single batch; errChan capacity must equal len(invoices) not len(batches) or sends block; ErrInvoiceCannotAdvance must stay a warning return (nil). |

## Anti-Patterns

- Calling billing.Adapter directly — all DB access goes through billing.Service.
- Propagating billing.ErrInvoiceCannotAdvance as an error from AdvanceInvoice.
- Forgetting lo.UniqBy deduplication on the merged list, causing double-advances.
- Using context.Background() inside goroutines instead of the caller-supplied ctx.
- Sizing errChan to len(batches) instead of len(invoices) — goroutine sends deadlock.

## Decisions

- **Three separate list queries merged with lo.UniqBy instead of a single compound query.** — The three states have distinct filter shapes in ListStandardInvoicesInput; merging at the query level would need OR semantics the adapter does not support cleanly.
- **Batched parallel goroutines rather than sequential advance.** — Advancement is per-invoice and independent; parallelism cuts wall-clock time for large namespaces while batchSize controls memory pressure.

## Example: Advance all eligible invoices in batches of 50

```
import billingworkeradvance "github.com/openmeterio/openmeter/openmeter/billing/worker/advance"

advancer, err := billingworkeradvance.NewAdvancer(billingworkeradvance.Config{BillingService: svc, Logger: logger})
if err != nil { return err }
return advancer.All(ctx, []string{"default"}, 50)
```

<!-- archie:ai-end -->
