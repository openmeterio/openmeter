# collect

<!-- archie:ai-start -->

> Cobra sub-command package for invoice collection operations (list collectable gathering invoices, collect for a specific customer, collect all). Uses structured input types from `billingworkercollect` rather than positional primitives.

## Patterns

**Structured input types from domain packages** — Pass `billingworkercollect.ListCollectableInvoicesInput` and `CollectCustomerInvoiceInput` structs rather than inline primitives, so the compiler enforces required fields. (`internal.App.BillingCollector.ListCollectableInvoices(cmd.Context(), billingworkercollect.ListCollectableInvoicesInput{Namespaces: namespaces, CollectionAt: time.Now()})`)
**time.Now() as default AsOf/CollectionAt** — Collection operations default to `time.Now()` for temporal boundary; expose a flag if callers need to override. (`AsOf: time.Now()`)
**StringSliceVar for multi-value filters** — Multi-value filters (namespaces, customerIDs, invoiceIDs) use `cmd.PersistentFlags().StringSliceVar` accepting comma-separated or repeated flags. (`cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")`)
**cmd.Context() for context propagation** — Always pass `cmd.Context()` to service calls. (`internal.App.BillingCollector.CollectCustomerInvoice(cmd.Context(), ...)`)
**Sub-command factory functions** — Each sub-command is a factory var registered in `init()` via `Cmd.AddCommand()`. (`var AllCmd = func() *cobra.Command { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Defines parent Cmd plus ListCmd, InvoiceCmd, AllCmd for gathering invoice collection. InvoiceCmd hardcodes namespace='default' — a known limitation. | InvoiceCmd hardcodes `Namespace: "default"` for CustomerID; if multi-namespace support is needed, add a --namespace flag and pass it through. |

## Anti-Patterns

- Hardcoding namespace strings (InvoiceCmd already has this debt — do not replicate it)
- Using context.Background() instead of cmd.Context()
- Inline construction of billing service instances instead of using internal.App
- Passing nil CollectionAt — always supply time.Now() or a flag-driven value

## Decisions

- **Use billingworkercollect input structs rather than individual parameters** — Input structs evolve independently of the CLI; adding new fields to the struct does not require changing every call site.

<!-- archie:ai-end -->
