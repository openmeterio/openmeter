# billing

<!-- archie:ai-start -->

> Cobra parent command that aggregates four billing sub-commands (advance, advancecharges, collect, subscriptionsync) under a single 'billing' namespace. Acts as a pure aggregator — contains no business logic itself.

## Patterns

**Aggregator-only parent command** — billing.go registers sub-commands in init() and exposes a single Cmd var. No RunE, no flags, no logic. (`Cmd.AddCommand(advance.Cmd); Cmd.AddCommand(collect.Cmd)`)
**Each sub-command lives in its own sub-package** — advance, advancecharges, collect, subscriptionsync are separate packages imported and registered in billing.go's init(). (`import "github.com/openmeterio/openmeter/cmd/jobs/billing/advance"`)
**internal.App singleton for service access** — All sub-commands access billing services exclusively through internal.App fields (BillingAutoAdvancer, ChargesAutoAdvancer, BillingSubscriptionReconciler) — never construct services locally. (`app.BillingAutoAdvancer.AdvanceInvoices(cmd.Context(), input)`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() — never context.Background() or context.TODO(). (`func(cmd *cobra.Command, args []string) error { return svc.Do(cmd.Context(), ...) }`)
**Nil guard for optional features** — advancecharges guards every execution path against app.ChargesAutoAdvancer == nil because charges are optional. (`if app.ChargesAutoAdvancer == nil { return errors.New("charges feature disabled") }`)
**PersistentFlags on parent Cmd for shared filters** — --namespace and similar shared flags belong on the parent (billing) Cmd or on the sub-command that owns them, never duplicated across siblings. (`Cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace filter")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Registers the four sub-commands in init(); exposes Cmd for the parent jobs command to mount. | Never add RunE or flags here — this is a pure aggregator. |
| `advance/advance.go` | List, advance single, advance-all sub-commands via BillingAutoAdvancer. | Uses nil slice for optional namespace filter — don't default to empty string. |
| `advancecharges/advancecharges.go` | Charge advance sub-commands; nil-guards ChargesAutoAdvancer before every call. | Omitting the nil guard causes a panic when the charges feature is disabled. |
| `collect/collect.go` | Invoice collection sub-commands using billingworkercollect structured input types. | Always supply time.Now() for CollectionAt — passing nil is a known debt in InvoiceCmd, do not replicate. |
| `subscriptionsync/sync.go` | Subscription-to-invoice sync reconciliation sub-commands via BillingSubscriptionReconciler. | Always apply lookback filter; omitting it scans all historical subscriptions. |

## Anti-Patterns

- Constructing billing or charges service instances locally instead of using internal.App fields
- Using context.Background() or context.TODO() instead of cmd.Context()
- Registering sub-command-specific flags on the billing parent Cmd (causes flag pollution across siblings)
- Hardcoding namespace strings instead of accepting them via --namespace flag
- Omitting the nil guard on ChargesAutoAdvancer — panics when charges feature is disabled

## Decisions

- **billing.go is a pure aggregator with no logic** — Keeps sub-command concerns isolated; billing.go only wires them together so each can evolve independently.
- **All service access goes through internal.App** — Prevents duplicate wiring and ensures the same Wire-provisioned instances (with correct feature flags) are used in all sub-commands.

## Example: Adding a new billing sub-command

```
// In billing.go init():
import "github.com/openmeterio/openmeter/cmd/jobs/billing/mynewcmd"
func init() {
    Cmd.AddCommand(mynewcmd.Cmd)
}
// In billing/mynewcmd/mynewcmd.go:
var Cmd = &cobra.Command{
    Use:   "my-new-cmd",
    Short: "Does X",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := internal.MustGetApp(cmd.Context())
        return app.BillingAutoAdvancer.DoX(cmd.Context(), input)
    },
}
```

<!-- archie:ai-end -->
