# service

<!-- archie:ai-start -->

> Concrete implementation of notification.Service, orchestrating channel and rule lifecycle against both the notification.Repository (Ent/PostgreSQL) and the webhook.Handler (Svix). All multi-step mutations use transaction.Run / transaction.RunWithNoValue; Svix webhook operations are performed inside the same transaction scope as adapter writes.

## Patterns

**transaction.Run wraps all multi-step mutations** — CreateChannel, DeleteChannel, UpdateChannel, CreateRule, DeleteRule, and UpdateRule all wrap their bodies in transaction.Run(ctx, s.adapter, fn) or transaction.RunWithNoValue so adapter calls and webhook calls share a transaction context. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*notification.Channel, error) { channel, _ := s.adapter.CreateChannel(ctx, params); _, _ = s.webhook.CreateWebhook(ctx, ...); return channel, nil })`)
**params.Validate() called before every mutation** — Every Service method calls params.Validate() (or params.ValidateWith(...)) before any adapter or webhook call. Returns fmt.Errorf wrapping the validation error without touching the DB. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }`)
**Webhook adapter kept in sync with channel/rule lifecycle** — CreateChannel calls webhook.CreateWebhook then adapter.UpdateChannel (to store SigningSecret). DeleteChannel calls webhook.DeleteWebhook before adapter.DeleteChannel. CreateRule calls webhook.UpdateWebhookChannels(AddChannels) per channel; DeleteRule calls UpdateWebhookChannels(RemoveChannels). (`wb, err = s.webhook.CreateWebhook(ctx, webhook.CreateWebhookInput{Namespace: params.Namespace, ID: &channel.ID, ...}); updateIn.Config.WebHook.SigningSecret = wb.Secret; channel, err = s.adapter.UpdateChannel(ctx, updateIn)`)
**ValidateRuleChannels generic constraint** — ValidateRuleChannels[T] is a generic validator used in CreateRule and UpdateRule that calls s.adapter.ListChannels to verify referenced channels exist in the namespace before saving. (`err = params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s))`)
**ChannelIDsDifference for minimal webhook updates on UpdateRule** — UpdateRule computes channelIDsDiff := notification.NewChannelIDsDifference(params.Channels, oldChannelIDs) and only calls webhook.UpdateWebhookChannels for channels in Additions or Removals — skips unchanged channels. (`channelIDsDiff := notification.NewChannelIDsDifference(params.Channels, oldChannelIDs); if !channelIDsDiff.HasChanged() { return s.adapter.UpdateRule(ctx, params) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct (adapter, webhook, feature, logger), Config, New() constructor, and ListFeature delegating to feature.FeatureConnector. | New dependencies must be added to Config and validated in New(). Service satisfies notification.Service via var _ notification.Service = (*Service)(nil) compile-time assertion. |
| `channel.go (within service package)` | Implements ChannelService methods. CreateChannel stores the channel first, then creates the Svix webhook and updates SigningSecret back into the DB. | DeleteChannel blocks deletion if the channel is assigned to any rule — returns models.NewGenericValidationError with rule IDs. UpdateChannel validates type change is not attempted. |
| `rule.go (within service package)` | Implements RuleService methods. UpdateRule computes channel diffs and calls webhook.UpdateWebhookChannels only for changed channel assignments. | webhook.IsMaxChannelsPerWebhookExceededError must be checked after UpdateWebhookChannels; if hit, return models.NewGenericValidationError (not a 500). |

## Anti-Patterns

- Calling adapter methods for multi-step mutations outside transaction.Run — partial writes if webhook call fails mid-transaction
- Skipping params.Validate() before adapter calls — invalid inputs reach the DB
- Calling webhook.DeleteWebhook after adapter.DeleteChannel — leaves a dangling Svix webhook if DB delete succeeds but webhook delete fails
- Deleting a channel without checking for assigned rules first — leaves rules with references to a deleted channel
- Calling webhook.UpdateWebhookChannels for all channels on UpdateRule instead of only the diff — causes spurious Svix API calls

## Decisions

- **Webhook operations performed inside the same transaction.Run as adapter writes** — Channel and rule lifecycle must stay consistent with Svix state; wrapping both in the same transaction scope means a webhook API failure rolls back the DB change rather than leaving them out of sync.
- **ChannelIDsDifference for minimal Svix calls on UpdateRule** — Svix enforces a MaxChannelsPerWebhook limit; calling UpdateWebhookChannels for every channel on every rule update would hit the limit unnecessarily and cause spurious validation errors.

## Example: Create a new rule with webhook channel sync inside a transaction

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
			_, err = s.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{
				Namespace: params.Namespace, ID: channel.ID, AddChannels: []string{rule.ID},
			})
			if err != nil {
				if webhook.IsMaxChannelsPerWebhookExceededError(err) {
					return nil, models.NewGenericValidationError(fmt.Errorf("max rules per webhook exceeded"))
// ...
```

<!-- archie:ai-end -->
