// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s Service) ListRules(ctx context.Context, params notification.ListRulesInput) (notification.ListRulesResult, error) {
	if err := params.Validate(ctx, s); err != nil {
		return notification.ListRulesResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.repo.ListRules(ctx, params)
}

func (s Service) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := s.logger.WithGroup("rule").With(
		"operation", "create",
		"namespace", params.Namespace,
	)

	logger.Debug("creating rule", "type", params.Type)

	txFunc := func(ctx context.Context, repo notification.TxRepository) (*notification.Rule, error) {
		rule, err := repo.CreateRule(ctx, params)
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

	return notification.WithTx[*notification.Rule](ctx, s.repo, txFunc)
}

func (s Service) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	if err := params.Validate(ctx, s); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	txFunc := func(ctx context.Context, repo notification.TxRepository) error {
		rule, err := s.repo.GetRule(ctx, notification.GetRuleInput(params))
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

		return s.repo.DeleteRule(ctx, params)
	}

	return notification.WithTxNoValue(ctx, s.repo, txFunc)
}

func (s Service) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.repo.GetRule(ctx, params)
}

func (s Service) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	if err := params.Validate(ctx, s); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	logger := s.logger.WithGroup("rule").With(
		"operation", "update",
		"id", params.ID,
		"namespace", params.Namespace,
	)

	logger.Debug("updating rule")

	rule, err := s.repo.GetRule(ctx, notification.GetRuleInput{
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
		return s.repo.UpdateRule(ctx, params)
	}

	txFunc := func(ctx context.Context, repo notification.TxRepository) (*notification.Rule, error) {
		// Fetch all the channels from repo which are either added or removed from rule
		channels, err := repo.ListChannels(ctx, notification.ListChannelsInput{
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

		return s.repo.UpdateRule(ctx, params)
	}

	return notification.WithTx[*notification.Rule](ctx, s.repo, txFunc)
}
