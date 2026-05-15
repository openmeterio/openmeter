# advance

<!-- archie:ai-start -->

> Cobra sub-command package for invoice advance operations (list, advance single invoice, advance all). All service access flows exclusively through the `internal.App.BillingAutoAdvancer` singleton wired by app/common; no local service construction is permitted.

## Patterns

**internal.App singleton access** — All service calls go through `internal.App.<Field>`, never construct services locally. The singleton is populated by Wire before Cobra executes. (`internal.App.BillingAutoAdvancer.ListInvoicesToAdvance(cmd.Context(), ns, nil)`)
**cmd.Context() for context propagation** — Always pass `cmd.Context()` to service calls — never `context.Background()` — so cancellation and deadlines from Cobra's context chain are respected. (`internal.App.BillingAutoAdvancer.AdvanceInvoice(cmd.Context(), billing.InvoiceID{Namespace: namespace, ID: invoiceID})`)
**Sub-command factory functions** — Each sub-command is a `var <Name>Cmd = func() *cobra.Command { ... }` factory, not a top-level var. Allows deferred flag registration and avoids init-order issues. (`var ListCmd = func() *cobra.Command { cmd := &cobra.Command{Use: "list", ...}; return cmd }`)
**PersistentFlags on parent Cmd** — Flags shared across sub-commands (e.g. `--namespace`) are registered on the parent `Cmd` via `Cmd.PersistentFlags()` in `init()`, not on individual sub-commands. (`Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace the operation should be performed")`)
**Nil slice for optional namespace filter** — Namespace filter is a `[]string`; build it conditionally so the service receives nil (unfiltered) when no namespace is specified, not an empty slice. (`var ns []string; if namespace != "" { ns = append(ns, namespace) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `advance.go` | Defines parent Cmd plus ListCmd, InvoiceCmd, AllCmd factory functions for invoice advance operations. | InvoiceCmd validates namespace is non-empty before calling AdvanceInvoice — replicate this guard on any new sub-command that requires a scoped namespace; omitting it produces a runtime error with no stack trace. |

## Anti-Patterns

- Constructing billing services locally instead of using internal.App
- Using context.Background() instead of cmd.Context()
- Registering sub-command-specific flags on the parent Cmd (causes flag pollution across siblings)
- Hardcoding namespace strings instead of accepting them via --namespace flag
- Calling AdvanceInvoice without first validating that namespace is non-empty

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
