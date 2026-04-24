# service

<!-- archie:ai-start -->

> Business logic layer for the notification domain, implementing notification.Service by composing notification.Repository (Ent adapter) with webhook.Handler (Svix). Orchestrates channel/rule/event lifecycle, validates cross-entity constraints (rule channels must exist in same namespace), and maintains Svix webhook registrations in sync with DB state.

## Patterns

**params.Validate() before any adapter call** — Every service method calls params.Validate() (or the equivalent domain input validator) at the top before entering a transaction or calling the adapter. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }`)
**transaction.Run for multi-step mutations** — Channel and rule create/update/delete use transaction.Run(ctx, s.adapter, fn) to wrap adapter calls + Svix webhook calls in a single transaction. Read-only methods call adapter directly. (`return transaction.Run(ctx, s.adapter, fn)`)
**ValidateRuleChannels generic validator** — ValidateRuleChannels[I notification.CreateRuleInput | notification.UpdateRuleInput] is a generic function that validates all channel IDs exist in the namespace before create/update. Called via params.ValidateWith(...). (`err = params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s))`)
**Svix webhook sync on channel/rule mutations** — CreateChannel calls s.webhook.CreateWebhook; UpdateChannel calls s.webhook.UpdateWebhook. CreateRule/UpdateRule call s.webhook.UpdateWebhookChannels for each channel in the rule. Svix calls happen inside the same transaction.Run fn as the adapter calls. (`s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{Namespace: params.Namespace, ID: &channel.ID, URL: channel.Config.WebHook.URL, ...})`)
**ChannelIDMetadataKey constant for Svix metadata** — Svix webhooks are tagged with Metadata{ChannelIDMetadataKey: channel.ID} to allow reverse lookup. Always set this when creating a webhook. (`Metadata: map[string]string{ChannelIDMetadataKey: channel.ID}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config, New() constructor, ListFeature() implementation. Holds adapter (notification.Repository), webhook (webhook.Handler), feature (feature.FeatureConnector). | All three dependencies are required; nil check is in New(). New service methods must satisfy notification.Service interface (var _ notification.Service = (*Service)(nil) compile check). |
| `channel.go` | Channel CRUD. CreateChannel: DB create + Svix create. UpdateChannel: DB update + Svix update. DeleteChannel: soft-delete DB + Svix disable. All inside transaction.Run. | DeleteChannel disables the webhook in Svix — the webhook ID matches the channel ID. |
| `rule.go` | Rule CRUD with ValidateRuleChannels and Svix channel registration. ValidateRuleConfigWithFeatures called to check feature references in rule config. | webhook.MaxChannelsPerWebhook is enforced in UpdateRule; exceeding it returns a specific error that must be mapped in the HTTP layer. |
| `event.go` | ListEvents, GetEvent, CreateEvent, ResendEvent. CreateEvent also calls s.webhook.SendWebhookMessage to dispatch via Svix. | ResendEvent re-dispatches an existing event through the webhook — it does not create a new DB record. |
| `deliverystatus.go` | Thin pass-through to adapter for delivery status list/get/update operations. | No webhook calls here — delivery status is purely a DB tracking concern. |

## Anti-Patterns

- Calling adapter methods without wrapping multi-step mutations in transaction.Run
- Skipping params.Validate() before adapter/webhook calls
- Creating a Svix webhook outside the transaction.Run fn (Svix call and DB call must be atomic)
- Calling s.webhook.CreateWebhook without setting ChannelIDMetadataKey in Metadata
- Adding business logic to the adapter layer instead of the service layer

## Decisions

- **Svix webhook operations co-located with DB mutations inside transaction.Run** — Channel state (enabled/disabled, URL, signing secret) must stay in sync between PostgreSQL and Svix; running both inside the same logical transaction prevents divergence on partial failure.
- **Generic ValidateRuleChannels[I] validator** — Both CreateRule and UpdateRule share identical channel-existence validation logic; the generic constraint avoids duplication while keeping type safety.

## Example: Add a new service method that mutates both DB and Svix

```
func (s Service) NewMutation(ctx context.Context, params notification.NewMutationInput) (*notification.Channel, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	fn := func(ctx context.Context) (*notification.Channel, error) {
		result, err := s.adapter.SomeAdapterMethod(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to ...: %w", err)
		}
		_, err = s.webhook.UpdateWebhook(ctx, webhook.UpdateWebhookInput{...})
		if err != nil {
			return nil, fmt.Errorf("failed to update webhook: %w", err)
		}
		return result, nil
	}
// ...
```

<!-- archie:ai-end -->
