# collect

<!-- archie:ai-start -->

> Cobra sub-command package for invoice collection operations (list collectable gathering invoices, collect for a specific customer, collect all). Uses structured input types from `billingworkercollect` rather than positional primitives to keep the CLI compatible with service API evolution.

## Patterns

**Structured input types from domain packages** — Pass `billingworkercollect.ListCollectableInvoicesInput` and `CollectCustomerInvoiceInput` structs rather than inline primitives; the compiler enforces required fields and the CLI stays compatible when the service signature evolves. (`internal.App.BillingCollector.ListCollectableInvoices(cmd.Context(), billingworkercollect.ListCollectableInvoicesInput{Namespaces: namespaces, CollectionAt: time.Now()})`)
**time.Now() as default AsOf/CollectionAt** — Collection operations default to `time.Now()` for the temporal boundary. Expose a flag only if callers need to override. (`AsOf: time.Now()`)
**StringSliceVar for multi-value filters** — Multi-value filters (namespaces, customerIDs, invoiceIDs) use `cmd.PersistentFlags().StringSliceVar`, accepting comma-separated or repeated flag values. (`cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")`)
**cmd.Context() for context propagation** — Always pass `cmd.Context()` to service calls; never substitute `context.Background()`. (`internal.App.BillingCollector.CollectCustomerInvoice(cmd.Context(), billingworkercollect.CollectCustomerInvoiceInput{...})`)
**Sub-command factory functions** — Each sub-command is a factory var registered in `init()` via `Cmd.AddCommand()`; flags are registered inside the factory body. (`var AllCmd = func() *cobra.Command { cmd := &cobra.Command{Use: "all", ...}; cmd.PersistentFlags()...; return cmd }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collect.go` | Defines parent Cmd plus ListCmd, InvoiceCmd, AllCmd for gathering invoice collection. | InvoiceCmd hardcodes `Namespace: "default"` for CustomerID — a known tech debt. If multi-namespace support is needed, add a --namespace flag and thread it through the input struct. Do not replicate this pattern in new sub-commands. |

## Anti-Patterns

- Hardcoding namespace strings — InvoiceCmd already has this debt; do not replicate it in new commands
- Using context.Background() instead of cmd.Context()
- Constructing billing service instances locally instead of using internal.App
- Passing nil CollectionAt — always supply time.Now() or a flag-driven value
- Registering multi-value filter flags as StringVar instead of StringSliceVar

## Decisions

- **Use billingworkercollect input structs rather than individual parameters** — Input structs evolve independently of the CLI; adding new fields to the struct does not require changing every call site in this package.

## Example: Adding a new collect sub-command with namespace and customer filters

```
var ByCustomerCmd = func() *cobra.Command {
	var ns, cids []string
	cmd := &cobra.Command{
		Use:  "by-customer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.App.BillingCollector.All(cmd.Context(), ns, cids, 0)
		},
	}
	cmd.PersistentFlags().StringSliceVar(&ns, "n", nil, "filter by namespaces")
	cmd.PersistentFlags().StringSliceVar(&cids, "c", nil, "filter by customer ids")
	return cmd
}
```

<!-- archie:ai-end -->
