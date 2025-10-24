package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s Service) ListRules(ctx context.Context, params notification.ListRulesInput) (notification.ListRulesResult, error) {
	if err := params.Validate(); err != nil {
		return notification.ListRulesResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.ListRules(ctx, params)
}

func (s Service) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.Rule, error) {
		logger := s.logger.WithGroup("rule").With(
			"operation", "create",
			"namespace", params.Namespace,
		)

		err := params.Config.ValidateWith(notification.ValidateRuleConfigWithFeatures(ctx, s, params.Namespace))
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		logger.Debug("creating rule", "type", params.Type)

		rule, err := s.adapter.CreateRule(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule: %w", err)
		}

		for _, channel := range rule.Channels {
			switch channel.Type {
			case notification.ChannelTypeWebhook:
				_, err = s.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{
					Namespace: params.Namespace,
					ID:        channel.ID,
					AddChannels: []string{
						rule.ID,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		return rule, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s Service) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) error {
		rule, err := s.adapter.GetRule(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to get rule: %w", err)
		}

		for _, channel := range rule.Channels {
			switch channel.Type {
			case notification.ChannelTypeWebhook:
				_, err = s.webhook.UpdateWebhookChannels(ctx, webhook.UpdateWebhookChannelsInput{
					Namespace: params.Namespace,
					ID:        channel.ID,
					RemoveChannels: []string{
						rule.ID,
					},
				})
				if err != nil {
					return fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		return s.adapter.DeleteRule(ctx, params)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, fn)
}

func (s Service) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.GetRule(ctx, params)
}

func (s Service) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.Rule, error) {
		logger := s.logger.WithGroup("rule").With(
			"operation", "update",
			"id", params.ID,
			"namespace", params.Namespace,
		)

		rule, err := s.adapter.GetRule(ctx, notification.GetRuleInput{
			ID:        params.ID,
			Namespace: params.Namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get rule: %w", err)
		}

		if rule.DeletedAt != nil {
			return nil, notification.UpdateAfterDeleteError{
				Err: errors.New("not allowed to update deleted rule"),
			}
		}

		err = params.Config.ValidateWith(notification.ValidateRuleConfigWithFeatures(ctx, s, params.Namespace))
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		err = params.ValidateWith(func(i notification.UpdateRuleInput) error {
			if i.Type != rule.Type {
				return fmt.Errorf("cannot change rule type: %s to %s", rule.Type, i.Type)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		logger.Debug("updating rule")

		// Get list of channel IDs currently assigned to rule
		oldChannelIDs := lo.Map(rule.Channels, func(channel notification.Channel, _ int) string {
			return channel.ID
		})
		logger.Debug("currently assigned channels", "channels", oldChannelIDs)

		// Calculate channels diff for the update
		channelIDsDiff := notification.NewChannelIDsDifference(params.Channels, oldChannelIDs)

		logger.WithGroup("channels").Debug("difference in channels assignment",
			"changed", channelIDsDiff.HasChanged(),
			"additions", channelIDsDiff.Additions(),
			"removals", channelIDsDiff.Removals(),
		)

		// We can return early ff there is no change in the list of channels assigned to rule.
		if !channelIDsDiff.HasChanged() {
			return s.adapter.UpdateRule(ctx, params)
		}

		// Fetch all the channels from repo which are either added or removed from rule
		channels, err := s.adapter.ListChannels(ctx, notification.ListChannelsInput{
			Page: pagination.Page{
				// In order to avoid under-fetching. There cannot be more affected channels than
				// twice as the maximum number of allowed channels per rule.
				PageSize:   2 * notification.MaxChannelsPerRule,
				PageNumber: 1,
			},
			Namespaces:      []string{params.Namespace},
			Channels:        channelIDsDiff.All(),
			IncludeDisabled: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list channels for rule: %w", err)
		}
		logger.Debug("fetched all affected channels", "channels", channels.Items)

		// Update affected channels
		for _, channel := range channels.Items {
			switch channel.Type {
			case notification.ChannelTypeWebhook:
				input := webhook.UpdateWebhookChannelsInput{
					Namespace: params.Namespace,
					ID:        channel.ID,
				}

				if channelIDsDiff.InAdditions(channel.ID) {
					input.AddChannels = []string{rule.ID}
				}

				if channelIDsDiff.InRemovals(channel.ID) {
					input.RemoveChannels = []string{rule.ID}
				}

				logger.Debug("updating webhook for channel", "id", channel.ID, "input", input)

				_, err = s.webhook.UpdateWebhookChannels(ctx, input)
				if err != nil {
					return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		return s.adapter.UpdateRule(ctx, params)
	}

	return transaction.Run(ctx, s.adapter, fn)
}
