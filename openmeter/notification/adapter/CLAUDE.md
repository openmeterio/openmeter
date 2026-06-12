# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for notification channels, rules, events, and delivery statuses. Implements notification.Repository against entdb.Client; all reads/writes are namespace-scoped and transaction-aware.

## Patterns

**TransactingRepo wrapper on every method** — Each repository method defines an inner fn(ctx, *adapter) and returns entutils.TransactingRepo(ctx, a, fn) (or TransactingRepoWithNoValue for void). This rebinds the adapter to the tx carried in ctx. (`func (a *adapter) GetChannel(...) (*notification.Channel, error) { fn := func(ctx, a *adapter)(...){...}; return entutils.TransactingRepo(ctx, a, fn) }`)
**Tx/WithTx/Self transaction triad** — adapter implements Tx (HijackTx + entutils.NewTxDriver), WithTx (rebuilds adapter from raw tx config via entdb.NewTxClientFromRawConfig), and Self — the contract entutils.TransactingRepo expects. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Soft-delete filtering on list/get** — Channels and rules use SetDeletedAt(clock.Now())+SetDisabled(true) for deletes; queries filter with Or(DeletedAtIsNil(), DeletedAtGT(now)). EagerLoadActiveChannels / EagerLoadRulesWithActiveChannels apply the same predicate to edges. (`channeldb.Or(channeldb.DeletedAtIsNil(), channeldb.DeletedAtGT(clock.Now()))`)
**DB<->domain mapping via *FromDBEntity functions** — All conversion from entdb rows to domain lives in entitymapping.go: ChannelFromDBEntity, RuleFromDBEntity, EventFromDBEntity, EventDeliveryStatusFromDBEntity. Every timestamp is forced .UTC(). (`result = append(result, *ChannelFromDBEntity(*item))`)
**NotFound mapped to notification.NotFoundError** — On entdb.IsNotFound(err), return notification.NotFoundError{NamespacedID:{Namespace, ID}} rather than the raw ent error so the httpdriver error encoder can map it to 404. (`if entdb.IsNotFound(err) { return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}} }`)
**Annotation JSONB filtering for event queries** — ListEvents filters on annotation keys via entutils.JSONBIn(eventdb.FieldAnnotations, key, values) — used for dedupe hashes, feature/subject key+id, etc. Add new filterable annotations the same way. (`entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationBalanceEventDedupeHash, params.DeduplicationHashes)`)
**Event creation also fans out delivery statuses** — CreateEvent inserts the event, re-fetches the rule with active channels, then CreateBulk a NotificationEventDeliveryStatus (state Pending) per channel, wiring eventRow.Edges manually before mapping. (`q := a.db.NotificationEventDeliveryStatus.Create().SetEventID(eventRow.ID).SetChannelID(channel.ID).SetState(notification.EventDeliveryStatusStatePending)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client,Logger}+Validate, New() returning notification.Repository, and the Tx/WithTx/Self triad | New requires both Client and Logger non-nil; do not fall back to slog.Default(). |
| `entitymapping.go` | All DB->domain mappers + eventPayloadFromJSON which version-gates invoice payloads | eventPayloadFromJSON rejects invoice payloads whose Version != EventPayloadVersionCurrent; preserve UTC() on all timestamps. |
| `event.go` | ListEvents/GetEvent/CreateEvent + EagerLoadRulesWithActiveChannels | CreateEvent must keep delivery-status CreateBulk inside the same fn so it shares the tx; eager-loads must use clock.Now(). |
| `rule.go` | Rule CRUD + EagerLoadActiveChannels; UpdateRule does ClearChannels().AddChannelIDs() | UpdateRule replaces (not merges) channel edges; channel edges are re-queried separately to populate the returned domain object. |
| `channel.go` | Channel CRUD; DeleteChannel is a soft delete (SetDisabled+SetDeletedAt) | Deletes are soft; list excludes deleted channels but GetChannel does not filter deleted. |
| `deliverystatus.go` | List/Get/UpdateEventDeliveryStatus; time bounds forced to UTC | NextAttempt is normalized via lo.ToPtr(...UTC()); uses SetOrClearNextAttemptAt for nil-able field. |
| `entitymapping_test.go` | Guards EventPayload JSON shape and version gating | TestEventPayloadV1JSONShape asserts api.Invoice fields sit flat under invoice.* — embedding/tag regressions break stored JSONB. |

## Anti-Patterns

- Calling a.db directly in a method without wrapping in entutils.TransactingRepo/TransactingRepoWithNoValue — loses tx propagation
- Returning raw entdb errors instead of notification.NotFoundError on not-found
- Hard-deleting channels/rules instead of soft-delete (SetDeletedAt+SetDisabled)
- Mapping DB rows inline instead of via the *FromDBEntity functions in entitymapping.go
- Storing non-UTC timestamps or skipping .UTC() in mappers

## Decisions

- **Soft-delete with DeletedAt + Disabled rather than physical delete** — Events reference rules/channels historically; physical delete would orphan delivery history and break audit/resend.
- **Annotations stored as JSONB and queried via entutils.JSONBIn** — Lets the consumer dedupe and filter events by feature/subject/dedupe-hash without dedicated columns per dimension.

## Example: Adding a namespace-scoped repository read with tx propagation and NotFound mapping

```
func (a *adapter) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Rule, error) {
		row, err := a.db.NotificationRule.Query().
			Where(ruledb.ID(params.ID), ruledb.Namespace(params.Namespace)).
			WithChannels(EagerLoadActiveChannels(clock.Now())).First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}}
			}
			return nil, fmt.Errorf("failed to fetch notification rule: %w", err)
		}
		return RuleFromDBEntity(*row), nil
	}
	return entutils.TransactingRepo(ctx, a, fn)
}
```

<!-- archie:ai-end -->
