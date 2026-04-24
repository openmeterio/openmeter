# advance

<!-- archie:ai-start -->

> Cobra sub-command package for invoice advance operations (list, advance single invoice, advance all). Accesses billing auto-advancement exclusively through the `internal.App` singleton which carries a `BillingAutoAdvancer` field wired by app/common.

## Patterns

**internal.App singleton access** — All service calls go through `internal.App.<Field>`, never construct services locally. The singleton is populated by Wire before Cobra executes. (`internal.App.BillingAutoAdvancer.ListInvoicesToAdvance(cmd.Context(), ns, nil)`)
**cmd.Context() for context propagation** — Always pass `cmd.Context()` to service calls — never `context.Background()` — so cancellation and deadlines from Cobra's context chain are respected. (`internal.App.BillingAutoAdvancer.AdvanceInvoice(cmd.Context(), billing.InvoiceID{...})`)
**Sub-command factory functions** — Each sub-command is a `var <Name>Cmd = func() *cobra.Command { ... }` factory, not a top-level `var`. Allows deferred flag registration. (`var ListCmd = func() *cobra.Command { cmd := &cobra.Command{...}; return cmd }`)
**PersistentFlags on parent Cmd** — Flags shared across sub-commands (e.g. `--namespace`) are registered on the parent `Cmd` via `Cmd.PersistentFlags()` in `init()`. (`Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "...")`)
**Nil slice for optional namespace filter** — Namespace filter is a `[]string`; pass nil when empty so the service receives an unfiltered request, not an empty-slice filter. (`var ns []string; if namespace != "" { ns = append(ns, namespace) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Defines parent Cmd plus ListCmd, InvoiceCmd, AllCmd factory functions for invoice advance operations. | InvoiceCmd requires --namespace flag to be non-empty; validate before calling AdvanceInvoice or you get a runtime error with no stack. |

## Anti-Patterns

- Constructing billing services locally instead of using internal.App
- Using context.Background() instead of cmd.Context()
- Registering sub-command-specific flags on the parent Cmd (causes flag pollution across siblings)
- Hardcoding namespace strings instead of accepting them via --namespace flag

## Decisions

- **Thin Cobra wrappers delegating to internal.App service fields** — Keeps cmd/* free of business logic; all wiring happens in app/common via Wire so each sub-command is independently testable and the DI graph is compile-time verified.

## Example: Adding a new advance sub-command that advances invoices for a specific customer

```
var CustomerCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customer [CUSTOMER_ID]",
		Short: "Advance invoices for a customer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if namespace == "" {
				return fmt.Errorf("namespace is required")
			}
			_, err := internal.App.BillingAutoAdvancer.AdvanceInvoice(cmd.Context(), billing.InvoiceID{
				Namespace: namespace,
				ID:        args[0],
			})
			return err
		},
// ...
```

<!-- archie:ai-end -->
