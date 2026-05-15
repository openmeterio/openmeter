# subscriptionsync

<!-- archie:ai-start -->

> Cobra sub-command package for subscription-to-invoice sync reconciliation (list syncable subscriptions, sync all). Delegates entirely to `internal.App.BillingSubscriptionReconciler` using typed input structs from the `reconciler` package; provides the manual recovery path for missed billing-worker Kafka events.

## Patterns

**Typed reconciler input structs with embedding** — Use `reconciler.ReconcilerListSubscriptionsInput` and embed it inside `reconciler.ReconcilerAllInput` so list and all-run filters stay structurally identical and CLI flags are reusable. (`internal.App.BillingSubscriptionReconciler.All(cmd.Context(), reconciler.ReconcilerAllInput{ReconcilerListSubscriptionsInput: reconciler.ReconcilerListSubscriptionsInput{Namespaces: namespaces, Customers: customerIDs, Lookback: lookback}, Force: force})`)
**Default lookback constant** — Define a `defaultLookback` constant (`24 * time.Hour`) and use it as the flag default so behavior is explicit and consistent across sub-commands. (`const defaultLookback = 24 * time.Hour; cmd.PersistentFlags().DurationVar(&lookback, "l", defaultLookback, "lookback period")`)
**Force flag for idempotent re-runs** — AllCmd exposes a `--force` (`-f`) bool flag mapped to `ReconcilerAllInput.Force` to bypass sync-state checks for manual crash recovery runs. (`cmd.PersistentFlags().BoolVar(&force, "f", false, "force reconciliation (even if the sync state would not necessarily require it)")`)
**Per-sub-command flag registration** — Filter flags (namespaces, customerIDs, lookback) are registered on each sub-command individually via `cmd.PersistentFlags()`, not on the parent Cmd, because they are not universally needed. (`cmd.PersistentFlags().StringSliceVar(&namespaces, "n", nil, "filter by namespaces")`)
**cmd.Context() for context propagation** — Pass `cmd.Context()` to all reconciler calls; never substitute `context.Background()`. (`internal.App.BillingSubscriptionReconciler.ListSubscriptions(cmd.Context(), reconciler.ReconcilerListSubscriptionsInput{...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sync.go` | Defines parent Cmd plus ListCmd and AllCmd for subscription sync. AllCmd is the primary recovery tool for missed billing-worker events. | Package-level vars for filters are shared across sub-commands; registering the same flag on both sub-commands means the last flag-parse wins during a single invocation — avoid duplicate registration of the same var on both ListCmd and AllCmd if they execute in the same process run. |

## Anti-Patterns

- Calling reconciler methods without a lookback filter — results in scanning all historical subscriptions
- Using context.Background() instead of cmd.Context()
- Constructing a reconciler instance locally instead of using internal.App.BillingSubscriptionReconciler
- Omitting the --force flag on AllCmd — manual recovery runs need it to bypass sync-state guards
- Passing primitives directly to reconciler instead of using the typed input structs

## Decisions

- **Embed ReconcilerListSubscriptionsInput inside ReconcilerAllInput** — Keeps list and all-run filter structs structurally identical so the CLI flags are reusable and the reconciler API stays DRY. Adding a new filter field to the list input automatically becomes available to the all-run command.

## Example: Adding a new subscription-sync sub-command with namespace and customer filters using the reconciler pattern

```
var DryRunCmd = func() *cobra.Command {
	var ns, cids []string
	var lb time.Duration
	cmd := &cobra.Command{
		Use:  "dryrun",
		RunE: func(cmd *cobra.Command, args []string) error {
			subs, err := internal.App.BillingSubscriptionReconciler.ListSubscriptions(
				cmd.Context(),
				reconciler.ReconcilerListSubscriptionsInput{
					Namespaces: ns,
					Customers:  cids,
					Lookback:   lb,
				})
			if err != nil {
				return err
// ...
```

<!-- archie:ai-end -->
