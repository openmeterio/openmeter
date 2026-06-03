# service

<!-- archie:ai-start -->

> Concrete implementation of notification.Service, orchestrating channel and rule lifecycle against both notification.Repository (Ent/PostgreSQL) and webhook.Handler (Svix). All multi-step mutations use transaction.Run / RunWithNoValue; Svix webhook operations run inside the same transaction scope as adapter writes.

## Patterns

**transaction.Run wraps all multi-step mutations** — CreateChannel, DeleteChannel, UpdateChannel, CreateRule, DeleteRule, UpdateRule wrap their bodies in transaction.Run(ctx, s.adapter, fn) so adapter and webhook calls share a transaction. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*notification.Channel, error) { channel, _ := s.adapter.CreateChannel(ctx, params); _, _ = s.webhook.CreateWebhook(ctx, ...); return channel, nil })`)
**params.Validate() before every mutation** — Every Service method calls params.Validate() (or ValidateWith) before any adapter/webhook call, wrapping the error without touching the DB. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }`)
**Webhook adapter kept in sync with channel/rule lifecycle** — CreateChannel calls webhook.CreateWebhook then adapter.UpdateChannel (store SigningSecret). DeleteChannel calls webhook.DeleteWebhook before adapter.DeleteChannel. CreateRule/DeleteRule call UpdateWebhookChannels per channel. (`wb, _ := s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{...}); updateIn.Config.WebHook.SigningSecret = wb.Secret; channel, _ = s.adapter.UpdateChannel(ctx, updateIn)`)
**ValidateRuleChannels generic constraint** — ValidateRuleChannels[T] used in CreateRule/UpdateRule calls s.adapter.ListChannels to verify referenced channels exist in the namespace before saving. (`err = params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s))`)
**ChannelIDsDifference for minimal webhook updates on UpdateRule** — UpdateRule computes notification.NewChannelIDsDifference(params.Channels, oldChannelIDs) and only calls UpdateWebhookChannels for Additions or Removals. (`channelIDsDiff := notification.NewChannelIDsDifference(params.Channels, oldChannelIDs); if !channelIDsDiff.HasChanged() { return s.adapter.UpdateRule(ctx, params) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct (adapter, webhook, feature, logger), Config, New(), ListFeature delegating to feature.FeatureConnector. | New dependencies must be added to Config and validated in New(). Compile-time assertion var _ notification.Service = (*Service)(nil). |
| `channel.go` | ChannelService methods. CreateChannel stores the channel first, then creates the Svix webhook and updates SigningSecret back into the DB. | DeleteChannel blocks deletion if the channel is assigned to any rule (returns NewGenericValidationError with rule IDs). UpdateChannel forbids type change. |
| `rule.go` | RuleService methods. UpdateRule computes channel diffs and calls UpdateWebhookChannels only for changed assignments. | webhook.IsMaxChannelsPerWebhookExceededError must be checked after UpdateWebhookChannels; if hit, return NewGenericValidationError (not a 500). |

## Anti-Patterns

- Calling adapter methods for multi-step mutations outside transaction.Run — partial writes if a webhook call fails mid-transaction
- Skipping params.Validate() before adapter calls — invalid inputs reach the DB
- Calling webhook.DeleteWebhook after adapter.DeleteChannel — leaves a dangling Svix webhook if DB delete succeeds but webhook delete fails
- Deleting a channel without checking for assigned rules first — leaves rules referencing a deleted channel
- Calling UpdateWebhookChannels for all channels on UpdateRule instead of only the diff — causes spurious Svix API calls

## Decisions

- **Webhook operations performed inside the same transaction.Run as adapter writes** — Channel/rule lifecycle must stay consistent with Svix; same-transaction scope means a webhook API failure rolls back the DB change rather than leaving them out of sync.
- **ChannelIDsDifference for minimal Svix calls on UpdateRule** — Svix enforces MaxChannelsPerWebhook; calling UpdateWebhookChannels for every channel on every update would hit the limit and cause spurious validation errors.

## Example: Create a rule with webhook channel sync inside a transaction

```
func (s Service) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
	fn := func(ctx context.Context) (*notification.Rule, error) {
		if err := params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s)); err != nil {
			return nil, fmt.Errorf("invalid channels: %w", err)
		}
		rule, err := s.adapter.CreateRule(ctx, params)
		if err != nil { return nil, fmt.Errorf("failed to create rule: %w", err) }
		for _, channel := range rule.Channels {
			_, err = s.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{Namespace: params.Namespace, ID: channel.ID, AddChannels: []string{rule.ID}})
			if err != nil {
				if webhook.IsMaxChannelsPerWebhookExceededError(err) {
					return nil, models.NewGenericValidationError(fmt.Errorf("max rules per webhook exceeded"))
				}
				return nil, err
// ...
```

<!-- archie:ai-end -->
