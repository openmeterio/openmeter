# service

<!-- archie:ai-start -->

> Business-logic implementation of notification.Service. Orchestrates the adapter (persistence) and webhook.Handler (Svix), validates inputs, and keeps webhook channels in sync with rule/channel mutations inside transactions.

## Patterns

**Validate then transaction.Run(ctx, s.adapter, fn)** — Mutating methods call params.Validate() first, then wrap side-effects in transaction.Run / transaction.RunWithNoValue over s.adapter so DB writes and webhook calls commit/roll back together. Pure reads delegate straight to the adapter. (`if err := params.Validate(); err != nil { return ... }; return transaction.Run(ctx, s.adapter, fn)`)
**Channel writes mirror to Svix webhook** — CreateChannel persists, then s.webhook.CreateWebhook, then re-UpdateChannel to store the returned signing secret. UpdateChannel re-syncs via UpdateWebhook including the rule IDs assigned to the channel. Only ChannelTypeWebhook is handled; default returns an error. (`wb, _ := s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{...}); updateIn.Config.WebHook.SigningSecret = wb.Secret`)
**Rule channel-assignment diffing** — UpdateRule computes notification.NewChannelIDsDifference(new, old); returns early if unchanged, otherwise fetches affected channels and calls webhook.UpdateWebhookChannels with AddChannels/RemoveChannels per InAdditions/InRemovals. (`channelIDsDiff := notification.NewChannelIDsDifference(params.Channels, oldChannelIDs)`)
**Config + feature validation** — Create/UpdateRule validate config via params.Config.ValidateWith(notification.ValidateRuleConfigWithFeatures(ctx, s, ns)) and channels via ValidateRuleChannels[I](ctx, s) — a generic over CreateRuleInput|UpdateRuleInput that checks every channel exists. (`err := params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s))`)
**Guard mutations against deleted aggregates** — UpdateChannel/UpdateRule fetch the current row and return notification.UpdateAfterDeleteError when DeletedAt != nil; CreateEvent rejects deleted/disabled rules (NotFoundError / GenericValidationError). (`if rule.DeletedAt != nil { return nil, notification.UpdateAfterDeleteError{Err: errors.New("not allowed to update deleted rule")} }`)
**Max-channels-per-webhook surfaced as validation error** — When webhook.UpdateWebhookChannels returns webhook.IsMaxChannelsPerWebhookExceededError, wrap as models.NewGenericValidationError referencing webhook.MaxChannelsPerWebhook rather than a generic 500. (`if webhook.IsMaxChannelsPerWebhookExceededError(err) { return nil, models.NewGenericValidationError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct (feature, adapter, webhook, logger), Config+New() validation, ChannelIDMetadataKey, ListFeature helper | New requires Adapter, FeatureConnector, Webhook, Logger all non-nil; Service satisfies notification.Service (compile-time _ assertion). |
| `channel.go` | Channel CRUD with Svix sync; DeleteChannel blocks if any rule references the channel | DeleteChannel returns GenericValidationError when rules reference it; UpdateChannel forbids changing channel Type and re-pushes assigned rule IDs to the webhook. |
| `rule.go` | Rule CRUD + ValidateRuleChannels generic + UpdateRule channel diffing | UpdateRule short-circuits webhook sync when channel set unchanged; over-fetch guard uses 2*MaxChannelsPerRule page size. |
| `event.go` | CreateEvent (rejects deleted/disabled rules), ListEvents/GetEvent, ResendEvent state-machine | ResendEvent only resets statuses in Sending/Success/Failed to Resending, skips disabled channels, and stamps AnnotationEventResendTimestamp. |
| `deliverystatus.go` | UpdateEventDeliveryStatus (in tx), List/GetEventsDeliveryStatus | Only the update wraps in transaction.Run; reads delegate directly. |

## Anti-Patterns

- Mutating channels/rules without transaction.Run wrapping both DB write and webhook call
- Skipping params.Validate() / ValidateRuleConfigWithFeatures / ValidateRuleChannels before persisting
- Updating a deleted aggregate instead of returning notification.UpdateAfterDeleteError
- Deleting a channel still referenced by a rule (must return a validation error)
- Surfacing max-channels-exceeded as a 500 instead of a GenericValidationError

## Decisions

- **Webhook side-effects run inside the same transaction as DB writes** — Channel/rule state in Postgres and Svix must stay consistent; a failed Svix call rolls back the DB mutation.
- **Channel deletion is blocked while rules reference it** — Prevents orphaning rules whose only delivery target would vanish; forces explicit rule reassignment first.

## Example: A mutating service method: validate, then run DB + webhook in one transaction

```
func (s Service) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
	fn := func(ctx context.Context) (*notification.Channel, error) {
		channel, err := s.adapter.CreateChannel(ctx, params)
		if err != nil { return nil, err }
		wb, err := s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{Namespace: params.Namespace, ID: &channel.ID, URL: channel.Config.WebHook.URL})
		if err != nil { return nil, err }
		updateIn := notification.UpdateChannelInput{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: channel.ID}, Config: channel.Config}
		updateIn.Config.WebHook.SigningSecret = wb.Secret
		return s.adapter.UpdateChannel(ctx, updateIn)
	}
	return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
