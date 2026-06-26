# collect

<!-- archie:ai-start -->

> Batch/cron driver (package billingworkercollect) that finds gathering invoices due for collection and converts each customer's pending lines into standard invoices via billing.Service.InvoicePendingLines. Mirrors the advance package's structure for the collection phase.

## Patterns

**Dual-service dependency via Config + NewInvoiceCollector** — InvoiceCollector holds a billing.GatheringInvoiceService (for listing/recalculating gathering invoices) and a billing.Service (for InvoicePendingLines), plus lockedNamespaces and a logger. Build only via NewInvoiceCollector, which nil-checks both services and the logger. (`c, err := NewInvoiceCollector(Config{GatheringInvoiceService: g, BillingService: s, Logger: l, LockedNamespaces: ns})`)
**Input structs with Validate()** — ListCollectableInvoicesInput and CollectCustomerInvoiceInput each have Validate() that collects into `var errs []error` and returns errors.Join(errs...); callers check it before doing work. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid input: %w", err) }`)
**CollectionAt filter via pkg/filter Or(Lte || not-exists)** — ListCollectableInvoices builds billing.ListGatheringInvoicesInput.CollectionAt as a filter.FilterTime Or of {Lte: collectionAt} and {Exists: false}, so invoices with a nil NextCollectionAt are also picked up (and warn-logged). (`CollectionAt: filter.FilterTime{Or: &[]filter.FilterTime{{Lte: ...}, {Exists: lo.ToPtr(false)}}}`)
**Collection disables progressive billing** — CollectCustomerInvoice calls InvoicePendingLines with billing.WithPartialInvoiceLinesDisabled() so system-driven collection never produces partial/progressive invoices. (`a.billingService.InvoicePendingLines(ctx, in, billing.WithPartialInvoiceLinesDisabled())`)
**Sentinel-error recovery: locked namespace and no-lines** — ErrNamespaceLocked → warn and skip (return nil,nil); ErrInvoiceCreateNoLines → warn and call gatheringInvoices.RecalculateGatheringInvoices to repair inconsistent state, then return nil,nil. Other errors are wrapped and returned. (`if errors.Is(err, billing.ErrInvoiceCreateNoLines) { ...RecalculateGatheringInvoices(ctx, params.CustomerID)... }`)
**Per-customer concurrent fan-out skipping locked namespaces** — All() maps gathering invoices to customer.CustomerID, dedupes with lo.Uniq, filters out lockedNamespaces via slices.Contains, lo.Chunk(batchSize), then runs a goroutine-per-customer WaitGroup with a buffered errChan joined by errors.Join (same shape as the advance package). (`lo.Filter(lo.Uniq(customerIDs), func(id customer.CustomerID, _ int) bool { return !slices.Contains(a.lockedNamespaces, id.Namespace) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Entire package: InvoiceCollector, ListCollectableInvoices, CollectCustomerInvoice (per-customer line-to-invoice conversion with sentinel handling), All() batch runner, the two input structs + Validate, and Config + NewInvoiceCollector. | All() reuses the loop var `err` inside goroutines (`_, err = a.CollectCustomerInvoice(...)`) — same race shape as advance.go; scope err locally in new code. Locked namespaces are filtered in All() but CollectCustomerInvoice still re-checks ErrNamespaceLocked, so don't assume one guard suffices. WithPartialInvoiceLinesDisabled is mandatory for system collection. |

## Anti-Patterns

- Allowing progressive/partial invoice lines during system collection — omitting WithPartialInvoiceLinesDisabled changes billing semantics.
- Treating ErrInvoiceCreateNoLines as fatal instead of triggering RecalculateGatheringInvoices to repair state.
- Treating ErrNamespaceLocked as an error rather than a skip — locked namespaces must be quietly bypassed.
- Constructing InvoiceCollector directly instead of via NewInvoiceCollector, skipping the required-dependency nil checks.

## Decisions

- **Disable partial invoice lines for collection runs.** — Background collection should produce complete invoices, not the progressive/partial lines used in interactive billing flows.
- **Recalculate gathering invoices on ErrInvoiceCreateNoLines instead of erroring.** — A gathering invoice flagged for collection but yielding no lines signals inconsistent state; recalculation self-heals rather than failing the batch.

<!-- archie:ai-end -->
