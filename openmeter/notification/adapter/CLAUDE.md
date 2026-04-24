# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence layer for the notification domain, implementing notification.Repository against four Ent entities: NotificationChannel, NotificationRule, NotificationEvent, NotificationEventDeliveryStatus. Primary constraint: every method body must be wrapped in entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-carried transaction is honored.

## Patterns

**TransactingRepo wrapper on every method** — All adapter methods define an inner fn closure and pass it to entutils.TransactingRepo (for value returns) or entutils.TransactingRepoWithNoValue (for error-only returns). Never call a.db directly outside a fn wrapper. (`fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) { ... }; return entutils.TransactingRepo(ctx, a, fn)`)
**Soft-delete via SetDeletedAt + SetDisabled** — Deletion in channels and rules is a soft-delete: SetDisabled(true) + SetDeletedAt(clock.Now()). All list queries filter with DeletedAtIsNil() OR DeletedAtGT(clock.Now()). (`query := a.db.NotificationChannel.UpdateOneID(params.ID).SetDisabled(true).SetDeletedAt(clock.Now())`)
**EagerLoadActiveChannels / EagerLoadRulesWithActiveChannels helpers** — Shared query modifier functions (defined in rule.go and event.go) filter channels to non-disabled and non-deleted at a given time. Always use these when loading rule.Edges.Channels or event.Edges.Rules. (`WithChannels(EagerLoadActiveChannels(clock.Now()))`)
**EntityMapping functions in entitymapping.go** — All DB-to-domain conversions go through pure functions ChannelFromDBEntity, RuleFromDBEntity, EventFromDBEntity, EventDeliveryStatusFromDBEntity. Never map inline in CRUD methods. (`return ChannelFromDBEntity(*channel), nil`)
**entdb.IsNotFound -> notification.NotFoundError** — Check for entdb.IsNotFound(err) after any query and return notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace:..., ID:...}} — never return raw Ent errors to the service. (`if entdb.IsNotFound(err) { return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}} }`)
**Bulk delivery-status creation on CreateEvent** — CreateEvent saves the event, then re-queries the rule with EagerLoadActiveChannels, then calls CreateBulk to insert one NotificationEventDeliveryStatus per active channel in the same transaction. (`statusBulkQuery := make([]*entdb.NotificationEventDeliveryStatusCreate, 0, len(ruleRow.Edges.Channels)); ... a.db.NotificationEventDeliveryStatus.CreateBulk(statusBulkQuery...).Save(ctx)`)
**Payload serialized as JSON string column** — NotificationEvent.Payload is stored as a string. CreateEvent calls json.Marshal(params.Payload) before SetPayload; EventFromDBEntity calls eventPayloadFromJSON which validates EventPayloadMeta.Version for invoice-type events. (`payloadJSON, err := json.Marshal(params.Payload); query.SetPayload(string(payloadJSON))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config (Client, Logger), New() constructor, adapter struct with Tx/WithTx/Self for entutils transaction plumbing. | WithTx creates a new txClient from raw config — must stay in sync with entutils.TxDriver. Self() is required by TransactingRepo generic constraint. |
| `entitymapping.go` | Pure DB-to-domain mapping functions. eventPayloadFromJSON performs version check for invoice events and rejects unknown versions. | Adding a new EventType must be handled in the eventPayloadFromJSON switch or it silently deserializes without version check. JSON shape tests in entitymapping_test.go guard against embedding regressions. |
| `event.go` | ListEvents and CreateEvent — most complex file. CreateEvent is a multi-step operation: save event, fetch rule+channels, bulk-create delivery statuses — all in one TransactingRepo. | If a new event type is added, delivery status creation logic in CreateEvent must handle nil channel edges. EagerLoadRulesWithActiveChannels is exported for reuse in list queries. |
| `rule.go` | CRUD for NotificationRule. UpdateRule calls ClearChannels().AddChannelIDs() to replace the channel set atomically. | CreateRule and UpdateRule both re-query channels after save to populate Edges.Channels — required because Ent Save does not return edge data. |
| `channel.go` | CRUD for NotificationChannel with orderBy switch and pagination. | IncludeDisabled=false adds channeldb.Disabled(false) filter; missing this would leak disabled channels. |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo/TransactingRepoWithNoValue fn closure — bypasses ctx transaction
- Returning raw entdb errors instead of notification.NotFoundError for 'not found' cases
- Mapping DB rows inline instead of using the entitymapping.go helper functions
- Loading rule channels without EagerLoadActiveChannels — would include disabled/deleted channels
- Manually editing eventPayloadFromJSON to skip version validation for new event types

## Decisions

- **Soft-delete instead of hard-delete for channels and rules** — In-flight events reference rule and channel IDs; hard deletion would break foreign key integrity and orphan delivery status records.
- **Bulk delivery-status insert inside CreateEvent transaction** — Atomicity: an event with no delivery statuses would never be dispatched; creating them in the same TX prevents a window where the event exists but has no pending deliveries.
- **eventPayloadFromJSON version check only for invoice event types** — Invoice payload schema changed; version 0 (missing version field) must be rejected to prevent deserializing stale JSONB into the new InvoicePayload shape.

## Example: Add a new adapter method returning a domain value inside a transaction

```
func (a *adapter) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		row, err := a.db.NotificationChannel.Query().
			Where(channeldb.ID(params.ID)).
			Where(channeldb.Namespace(params.Namespace)).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.ID}}
			}
			return nil, fmt.Errorf("failed to fetch notification channel: %w", err)
		}
		return ChannelFromDBEntity(*row), nil
	}
	return entutils.TransactingRepo(ctx, a, fn)
// ...
```

<!-- archie:ai-end -->
