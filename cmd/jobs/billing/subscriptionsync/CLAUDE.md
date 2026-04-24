# subscriptionsync

<!-- archie:ai-start -->

> Cobra sub-command package for subscription-to-invoice sync reconciliation (list syncable subscriptions, sync all). Delegates entirely to `internal.App.BillingSubscriptionReconciler` using typed input from the `reconciler` package.

## Patterns

**Typed reconciler input structs** — Use `reconciler.ReconcilerListSubscriptionsInput` and embed it inside `reconciler.ReconcilerAllInput`; never pass primitives directly. (`internal.App.BillingSubscriptionReconciler.All(cmd.Context(), reconciler.ReconcilerAllInput{ReconcilerListSubscriptionsInput: reconciler.ReconcilerListSubscriptionsInput{...}, Force: force})`)
**Shared filter flags across sub-commands via package-level vars** — Filter vars (`namespaces`, `customerIDs`, `lookback`) are package-level and registered per sub-command via `cmd.PersistentFlags()`, not on the parent Cmd. (`cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")`)
**Default lookback constant** — Define a `defaultLookback` constant (`24 * time.Hour`) and use it as the flag default so behavior is explicit and consistent. (`cmd.PersistentFlags().DurationVar(&lookback, "l", defaultLookback, "lookback period")`)
**Force flag for idempotent re-runs** — AllCmd exposes a `--force` (`-f`) bool flag mapped to `ReconcilerAllInput.Force` to bypass sync-state checks for manual recovery. (`cmd.PersistentFlags().BoolVar(&force, "f", false, "force reconciliation")`)
**cmd.Context() for context propagation** — Pass `cmd.Context()` to all reconciler calls. (`internal.App.BillingSubscriptionReconciler.ListSubscriptions(cmd.Context(), ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | Defines parent Cmd plus ListCmd and AllCmd for subscription sync. AllCmd is the primary recovery tool for missed billing-worker events. | Package-level vars for filters are shared across sub-commands; re-registering them on both sub-commands means the last flag registration wins — avoid duplicate registration. |

## Anti-Patterns

- Calling reconciler methods without the lookback filter — results in scanning all historical subscriptions
- Using context.Background() instead of cmd.Context()
- Constructing a reconciler instance locally instead of using internal.App.BillingSubscriptionReconciler
- Omitting the --force flag on AllCmd — manual recovery runs need it

## Decisions

- **Embed ReconcilerListSubscriptionsInput inside ReconcilerAllInput** — Keeps list and all-run filters structurally identical so the CLI flags are reusable and the reconciler API stays DRY.

<!-- archie:ai-end -->
