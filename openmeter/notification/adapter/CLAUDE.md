# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence layer for the notification domain, implementing notification.Repository against four Ent entities (NotificationChannel, NotificationRule, NotificationEvent, NotificationEventDeliveryStatus). Every method body is wrapped in entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-carried Ent transaction is honored.

## Patterns

**TransactingRepo wrapper on every method** — Every adapter method defines an inner fn closure and passes it to entutils.TransactingRepo (value returns) or TransactingRepoWithNoValue (error-only). Never call a.db outside a fn wrapper. (`fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) { ... }; return entutils.TransactingRepo(ctx, a, fn)`)
**Soft-delete via SetDisabled + SetDeletedAt** — Channels and rules are soft-deleted: SetDisabled(true) + SetDeletedAt(clock.Now()). All list queries filter with Or(DeletedAtIsNil(), DeletedAtGT(clock.Now())). (`a.db.NotificationChannel.UpdateOneID(params.ID).SetDisabled(true).SetDeletedAt(clock.Now()).Save(ctx)`)
**EagerLoadActiveChannels / EagerLoadRulesWithActiveChannels** — Shared query-modifier functions (rule.go, event.go) filter channels to non-disabled and non-deleted at a given time. Use these whenever loading rule/event channel edges. (`query.WithChannels(EagerLoadActiveChannels(clock.Now()))`)
**EntityMapping functions in entitymapping.go** — All DB-to-domain conversions go through pure functions: ChannelFromDBEntity, RuleFromDBEntity, EventFromDBEntity, EventDeliveryStatusFromDBEntity. Never map inline in CRUD methods. (`return ChannelFromDBEntity(*channel), nil`)
**entdb.IsNotFound -> notification.NotFoundError** — After any query, check entdb.IsNotFound(err) and return notification.NotFoundError{NamespacedID:...}; never surface raw Ent errors. (`if entdb.IsNotFound(err) { return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}} }`)
**Bulk delivery-status creation inside CreateEvent transaction** — CreateEvent saves the event, re-queries the rule with EagerLoadActiveChannels, then CreateBulk inserts one NotificationEventDeliveryStatus per active channel — all in one TransactingRepo fn. (`a.db.NotificationEventDeliveryStatus.CreateBulk(statusBulkQuery...).Save(ctx)`)
**Payload serialized as JSON string column** — NotificationEvent.Payload is a string. CreateEvent json.Marshal(params.Payload) before SetPayload; EventFromDBEntity calls eventPayloadFromJSON which validates EventPayloadMeta.Version for invoice events. (`payloadJSON, _ := json.Marshal(params.Payload); query.SetPayload(string(payloadJSON))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config (Client, Logger), New() constructor, adapter struct with Tx/WithTx/Self for entutils transaction plumbing. | WithTx creates a txClient via entdb.NewTxClientFromRawConfig — must stay in sync with entutils.TxDriver contract. Self() is required by the TransactingRepo generic constraint. |
| `entitymapping.go` | Pure DB-to-domain mappers; eventPayloadFromJSON does version check for invoice events. | A new EventType must be handled in the eventPayloadFromJSON switch or it silently deserializes without version check. entitymapping_test.go guards JSON shape. |
| `event.go` | ListEvents and CreateEvent; CreateEvent saves event, fetches rule+channels, bulk-creates delivery statuses in one TransactingRepo. | New event types must handle nil channel edges in CreateEvent. EagerLoadRulesWithActiveChannels is exported for reuse. |
| `rule.go` | NotificationRule CRUD; UpdateRule ClearChannels().AddChannelIDs() replaces channel set atomically. Defines EagerLoadActiveChannels. | CreateRule and UpdateRule re-query channels after Save — Ent Save does not return edge data. |
| `channel.go` | NotificationChannel CRUD with orderBy switch and pagination. | IncludeDisabled=false adds Disabled(false) filter; omitting leaks disabled channels. |
| `deliverystatus.go` | ListEventsDeliveryStatus, GetEventDeliveryStatus, UpdateEventDeliveryStatus — all in TransactingRepo. | UpdateEventDeliveryStatus uses SetOrClearNextAttemptAt for nullable NextAttemptAt — SetNextAttemptAt would panic on nil. |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo/TransactingRepoWithNoValue fn closure — bypasses ctx transaction
- Returning raw entdb errors instead of notification.NotFoundError for not-found cases
- Mapping DB rows inline in CRUD methods instead of using entitymapping.go helpers
- Loading rule channels without EagerLoadActiveChannels — includes disabled or deleted channels
- Editing eventPayloadFromJSON to skip version validation for new invoice event types

## Decisions

- **Soft-delete instead of hard-delete for channels and rules** — In-flight events reference rule and channel IDs; hard deletion breaks FK integrity and orphans delivery status records.
- **Bulk delivery-status insert inside CreateEvent transaction** — Atomicity: an event with no delivery statuses would never dispatch; same-TX creation prevents a window where the event exists with no pending deliveries.
- **Version check only for invoice event types in eventPayloadFromJSON** — Invoice payload schema changed; version 0 (missing field) must be rejected to avoid deserializing stale JSONB into the new InvoicePayload shape.

## Example: Add an adapter method returning a domain value inside a transaction

```
func (a *adapter) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		row, err := a.db.NotificationChannel.Query().Where(channeldb.ID(params.ID), channeldb.Namespace(params.Namespace)).First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}}
			}
			return nil, fmt.Errorf("failed to fetch notification channel: %w", err)
		}
		return ChannelFromDBEntity(*row), nil
	}
	return entutils.TransactingRepo(ctx, a, fn)
}
```

<!-- archie:ai-end -->
