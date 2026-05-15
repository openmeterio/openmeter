# billing

<!-- archie:ai-start -->

> Cobra parent command that aggregates four billing sub-commands (advance, advancecharges, collect, subscriptionsync) under a single 'billing' namespace. Acts as a pure aggregator — contains no business logic itself; all service access flows through internal.App fields wired by app/common.

## Patterns

**Aggregator-only parent command** — billing.go registers sub-commands in init() and exposes a single Cmd var. No RunE, no flags, no logic. (`func init() { Cmd.AddCommand(advance.Cmd); Cmd.AddCommand(collect.Cmd) }`)
**Sub-commands in separate packages** — Each sub-command (advance, advancecharges, collect, subscriptionsync) lives in its own sub-package imported and registered in billing.go's init(). (`import "github.com/openmeterio/openmeter/cmd/jobs/billing/advance"`)
**internal.App singleton for service access** — All sub-commands access billing services exclusively through internal.App fields (BillingAutoAdvancer, ChargesAutoAdvancer, BillingSubscriptionReconciler) — never construct services locally. (`app := internal.MustGetApp(cmd.Context()); app.BillingAutoAdvancer.AdvanceInvoices(cmd.Context(), input)`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() — never context.Background() or context.TODO(). (`func(cmd *cobra.Command, args []string) error { return svc.Do(cmd.Context(), ...) }`)
**Nil guard for optional features** — advancecharges guards every execution path against app.ChargesAutoAdvancer == nil because charges are optional and wired conditionally. (`if app.ChargesAutoAdvancer == nil { return errors.New("charges feature disabled") }`)
**PersistentFlags on owning Cmd for shared filters** — Shared flags like --namespace belong on the sub-command that owns them, never duplicated across siblings or pushed up to the billing parent Cmd. (`advanceCmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace filter")`)
**Structured input types from domain packages** — Sub-commands (collect, subscriptionsync) use typed input structs from domain packages rather than loose primitives to stay compatible with service API evolution. (`input := billingworkercollect.CollectInput{AsOf: time.Now(), Namespace: ns}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Pure aggregator — registers the four sub-commands in init() and exposes Cmd for the parent jobs command to mount. | Never add RunE, flags, or business logic here; this file must remain a zero-logic aggregator. |
| `advance/advance.go` | List, advance-single, and advance-all sub-commands via BillingAutoAdvancer. | Uses nil slice (not empty string) for optional namespace filter — do not default to empty string. |
| `advancecharges/advancecharges.go` | Charge advance sub-commands; nil-guards ChargesAutoAdvancer before every call. | Omitting the nil guard causes a panic when the charges feature is disabled (credits.enabled=false). |
| `collect/collect.go` | Invoice collection sub-commands using billingworkercollect structured input types. | Always supply time.Now() for CollectionAt — passing nil is a known debt in InvoiceCmd, do not replicate it. |
| `subscriptionsync/sync.go` | Subscription-to-invoice sync reconciliation sub-commands via BillingSubscriptionReconciler. | Always apply a lookback filter; omitting it scans all historical subscriptions and is extremely slow. |

## Anti-Patterns

- Constructing billing or charges service instances locally instead of using internal.App fields
- Using context.Background() or context.TODO() instead of cmd.Context() in any RunE
- Registering flags on the billing parent Cmd that belong only to a specific sub-command (causes flag pollution)
- Hardcoding namespace strings instead of accepting them via a --namespace flag
- Omitting the nil guard on app.ChargesAutoAdvancer — panics when the charges feature is disabled

## Decisions

- **billing.go is a pure aggregator with no logic** — Keeps sub-command concerns isolated; billing.go only wires them together so each can evolve independently without touching siblings.
- **All service access goes through internal.App** — Prevents duplicate wiring and ensures the same Wire-provisioned instances (with correct feature flags, including credits.enabled guard) are used in all sub-commands.
- **Each sub-command lives in its own package** — Allows independent flag registration, testability, and import without pulling in unrelated Cobra command state from siblings.

## Example: Adding a new billing sub-command with optional feature guard

```
// billing.go init():
import "github.com/openmeterio/openmeter/cmd/jobs/billing/mynewcmd"
func init() { Cmd.AddCommand(mynewcmd.Cmd) }

// billing/mynewcmd/mynewcmd.go:
var Cmd = &cobra.Command{
    Use:   "my-new-cmd",
    Short: "Does X",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := internal.MustGetApp(cmd.Context())
        if app.ChargesAutoAdvancer == nil {
            return errors.New("charges feature disabled")
        }
        return app.ChargesAutoAdvancer.DoX(cmd.Context(), input)
    },
// ...
```

<!-- archie:ai-end -->
