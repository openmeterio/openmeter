# billing

<!-- archie:ai-start -->

> Cobra parent command aggregating four billing sub-commands (advance, advancecharges, collect, subscriptionsync) under the 'billing' namespace of the jobs CLI. A pure aggregator — no business logic; all service access flows through internal.App fields wired by app/common.

## Patterns

**Aggregator-only parent command** — billing.go registers sub-commands in init() and exposes a single Cmd var — no RunE, no flags, no logic. (`func init() { Cmd.AddCommand(advance.Cmd); Cmd.AddCommand(collect.Cmd) }`)
**Sub-commands in separate packages** — Each sub-command (advance, advancecharges, collect, subscriptionsync) lives in its own sub-package imported and registered in billing.go's init(). (`import "github.com/openmeterio/openmeter/cmd/jobs/billing/advance"`)
**internal.App singleton for service access** — Sub-commands access billing services only through internal.App fields (BillingAutoAdvancer, ChargesAutoAdvancer, BillingSubscriptionReconciler) — never construct services locally. (`app := internal.MustGetApp(cmd.Context()); app.BillingAutoAdvancer.AdvanceInvoices(cmd.Context(), input)`)
**cmd.Context() for context propagation** — Every RunE uses cmd.Context() — never context.Background()/TODO(). (`func(cmd *cobra.Command, args []string) error { return svc.Do(cmd.Context(), ...) }`)
**Nil guard for optional features** — advancecharges guards every execution path against app.ChargesAutoAdvancer == nil because charges are wired conditionally (credits.enabled). (`if app.ChargesAutoAdvancer == nil { return errors.New("charges feature disabled") }`)
**Flags on the owning sub-command, not the parent** — Shared filter flags like --namespace belong on the owning sub-command (PersistentFlags), never duplicated across siblings or pushed to the billing parent Cmd. (`advanceCmd.PersistentFlags().StringVar(&namespace, "namespace", "", "namespace filter")`)
**Structured input types from domain packages** — collect and subscriptionsync use typed input structs (billingworkercollect, reconciler) rather than loose primitives to track service API evolution. (`input := billingworkercollect.CollectInput{AsOf: time.Now(), Namespace: ns}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Pure aggregator — registers the four sub-commands in init() and exposes Cmd for the parent jobs command. | Never add RunE, flags, or business logic here; this file must remain a zero-logic aggregator. |
| `advance/advance.go` | List, advance-single, and advance-all sub-commands via BillingAutoAdvancer. | Uses a nil slice (not empty string) for the optional namespace filter — do not default to empty string. |
| `advancecharges/advancecharges.go` | Charge advance sub-commands; nil-guards ChargesAutoAdvancer before every call. | Omitting the nil guard panics when the charges feature is disabled (credits.enabled=false). |
| `collect/collect.go` | Invoice collection sub-commands using billingworkercollect structured input types. | Always supply time.Now() for CollectionAt — passing nil is a known debt in InvoiceCmd; do not replicate it. Use StringSliceVar for multi-value filters. |
| `subscriptionsync/sync.go` | Subscription-to-invoice sync reconciliation via BillingSubscriptionReconciler — the manual recovery path for missed billing-worker Kafka events. | Always apply a lookback filter; omitting it scans all historical subscriptions. AllCmd needs --force to bypass sync-state guards. |

## Anti-Patterns

- Constructing billing or charges service instances locally instead of using internal.App fields
- Using context.Background()/TODO() instead of cmd.Context() in any RunE
- Registering flags on the billing parent Cmd that belong only to a specific sub-command (flag pollution)
- Hardcoding namespace strings instead of accepting them via a --namespace flag
- Omitting the nil guard on app.ChargesAutoAdvancer — panics when the charges feature is disabled

## Decisions

- **billing.go is a pure aggregator with no logic** — Keeps sub-command concerns isolated so each evolves independently without touching siblings.
- **All service access goes through internal.App** — Prevents duplicate wiring and ensures the same Wire-provisioned instances (with correct feature flags) are used everywhere.
- **Each sub-command lives in its own package** — Allows independent flag registration, testability, and import without pulling in sibling Cobra state.

<!-- archie:ai-end -->
