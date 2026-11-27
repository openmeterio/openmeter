package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (s Service) ListRules(ctx context.Context, params notification.ListRulesInput) (notification.ListRulesResult, error) {
	if err := params.Validate(); err != nil {
		return notification.ListRulesResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.ListRules(ctx, params)
}

func ValidateRuleChannels[I notification.CreateRuleInput | notification.UpdateRuleInput](ctx context.Context, s Service) func(input I) error {
	return func(params I) error {
		var (
			channels  []string
			namespace string
		)

		switch i := any(params).(type) {
		case notification.CreateRuleInput:
			channels = i.Channels
			namespace = i.Namespace
		case notification.UpdateRuleInput:
			channels = i.Channels
			namespace = i.Namespace
		}

		if len(channels) == 0 {
			return errors.New("rule must have at least one channel")
		}

		channelsOut, err := s.adapter.ListChannels(ctx, notification.ListChannelsInput{
			Page: pagination.Page{
				PageSize:   len(channels),
				PageNumber: 1,
			},
			Namespaces: []string{namespace},
			ID:         &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr(channels)}},
			OrderBy:    notification.OrderByCreatedAt,
			Order:      sortx.OrderAsc,
		})
		if err != nil {
			return fmt.Errorf("failed to validate channels: %w", err)
		}

		channelIDs := lo.SliceToMap(channelsOut.Items, func(item notification.Channel) (string, struct{}) {
			return item.ID, struct{}{}
		})

		var missingChannels []string

		for _, channelID := range channels {
			if _, ok := channelIDs[channelID]; !ok {
				missingChannels = append(missingChannels, channelID)
			}
		}

		if len(missingChannels) > 0 {
			return fmt.Errorf("non-existing channels: %v", missingChannels)
		}

		return nil
	}
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

		config, err := s.resolveRuleConfigFeatures(ctx, resolveRuleConfigFeaturesInput{
			Namespace: params.Namespace,
			Config:    params.Config,
		})
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		params.Config = config

		err = params.ValidateWith(ValidateRuleChannels[notification.CreateRuleInput](ctx, s))
		if err != nil {
			return nil, fmt.Errorf("invalid channels: %w", err)
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
					if webhook.IsMaxChannelsPerWebhookExceededError(err) {
						return nil, models.NewGenericValidationError(
							fmt.Errorf("maximum number (%d) of rules for webhook channels exceeded [notification.channel.id=%s notification.channel.name=%s]",
								webhook.MaxChannelsPerWebhook, channel.ID, channel.Name),
						)
					}

					return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		if err := s.adapter.RefreshChannelEventTypeFilters(ctx, notification.RefreshChannelEventTypeFiltersInput{
			Namespace:  params.Namespace,
			ChannelIDs: params.Channels,
		}); err != nil {
			return nil, err
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
		rule, err := s.adapter.GetRuleForUpdate(ctx, params)
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

		if err := s.adapter.DeleteRule(ctx, params); err != nil {
			return err
		}

		return s.adapter.RefreshChannelEventTypeFilters(ctx, notification.RefreshChannelEventTypeFiltersInput{
			Namespace:  params.Namespace,
			ChannelIDs: lo.Map(rule.Channels, func(c notification.Channel, _ int) string { return c.ID }),
		})
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

		rule, err := s.adapter.GetRuleForUpdate(ctx, notification.GetRuleInput{
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

		config, err := s.resolveRuleConfigFeatures(ctx, resolveRuleConfigFeaturesInput{
			Namespace: params.Namespace,
			Config:    params.Config,
		})
		if err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		params.Config = config

		err = params.ValidateWith(func(i notification.UpdateRuleInput) error {
			if i.Type != rule.Type {
				return fmt.Errorf("cannot change rule type: %s to %s", rule.Type, i.Type)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("invalid rule type: %w", err)
		}

		err = params.ValidateWith(ValidateRuleChannels[notification.UpdateRuleInput](ctx, s))
		if err != nil {
			return nil, fmt.Errorf("invalid channels: %w", err)
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

		persist := func() (*notification.Rule, error) {
			updated, err := s.adapter.UpdateRule(ctx, params)
			if err != nil {
				return nil, err
			}

			if updated == nil || updated.Revision != rule.Revision+1 {
				return nil, fmt.Errorf("failed to advance rule revision [namespace=%s rule.id=%s]", params.Namespace, params.ID)
			}

			if err := s.adapter.RefreshChannelEventTypeFilters(ctx, notification.RefreshChannelEventTypeFiltersInput{
				Namespace:  params.Namespace,
				ChannelIDs: channelIDsDiff.All(),
			}); err != nil {
				return nil, err
			}

			return updated, nil
		}

		// We can return early ff there is no change in the list of channels assigned to rule.
		if !channelIDsDiff.HasChanged() {
			return persist()
		}

		// Fetch all the channels from repo which are either added or removed from rule
		channels, err := s.adapter.ListChannels(ctx, notification.ListChannelsInput{
			Page: pagination.Page{
				// In order to avoid under-fetching. There cannot be more affected channels than
				// twice as the maximum number of allowed channels per rule.
				PageSize:   2 * notification.MaxChannelsPerRule,
				PageNumber: 1,
			},
			Namespaces: []string{params.Namespace},
			ID:         &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr(channelIDsDiff.All())}},
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
					if webhook.IsMaxChannelsPerWebhookExceededError(err) {
						return nil, models.NewGenericValidationError(
							fmt.Errorf("maximum number (%d) of rules for webhook channels exceeded [notification.channel.id=%s notification.channel.name=%s]",
								webhook.MaxChannelsPerWebhook, channel.ID, channel.Name),
						)
					}

					return nil, fmt.Errorf("failed to update webhook for channel: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid channel type: %s", channel.Type)
			}
		}

		return persist()
	}

	return transaction.Run(ctx, s.adapter, fn)
}

type resolveRuleConfigFeaturesInput struct {
	Namespace string
	Config    notification.RuleConfig
}

func (i resolveRuleConfigFeaturesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Config.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid config: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Resolve before persistence so a later reuse of a feature key cannot retarget a
// Rule. Copy the config before replacing features: the caller owns its pointers.
func (s Service) resolveRuleConfigFeatures(ctx context.Context, input resolveRuleConfigFeaturesInput) (notification.RuleConfig, error) {
	if err := input.Validate(); err != nil {
		return notification.RuleConfig{}, err
	}

	var references []string

	switch input.Config.Type {
	case notification.EventTypeBalanceThreshold:
		references = input.Config.BalanceThreshold.Features
	case notification.EventTypeEntitlementReset:
		references = input.Config.EntitlementReset.Features
	}

	if len(references) == 0 {
		return input.Config, nil
	}

	features, err := s.featureResolver.BatchResolve(ctx, input.Namespace, references...)
	if err != nil {
		return notification.RuleConfig{}, fmt.Errorf("failed to resolve rule features: %w", err)
	}

	ids := make([]string, 0, len(references))
	var missingFeatures []string

	for _, reference := range references {
		feature := features[reference]
		if feature == nil {
			missingFeatures = append(missingFeatures, reference)
			continue
		}

		ids = append(ids, feature.ID)
	}

	if len(missingFeatures) > 0 {
		return notification.RuleConfig{}, models.NewGenericValidationError(fmt.Errorf("non-existing features: %v", missingFeatures))
	}

	ids = lo.Uniq(ids)

	config := input.Config
	switch config.Type {
	case notification.EventTypeBalanceThreshold:
		balance := *config.BalanceThreshold
		balance.Features = ids
		config.BalanceThreshold = &balance
	case notification.EventTypeEntitlementReset:
		reset := *config.EntitlementReset
		reset.Features = ids
		config.EntitlementReset = &reset
	}

	return config, nil
}
