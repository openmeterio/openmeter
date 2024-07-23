package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/internal/ent/db"
	channeldb "github.com/openmeterio/openmeter/internal/ent/db/notificationchannel"
	ruledb "github.com/openmeterio/openmeter/internal/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type PostgresAdapterConfig struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c PostgresAdapterConfig) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

type postgresRepository interface {
	notification.ChannelRepository
	notification.RuleRepository
}

var _ postgresRepository = (*postgresAdapter)(nil)

type postgresAdapter struct {
	db *entdb.Client

	logger *slog.Logger
}

func newPostgresAdapter(config PostgresAdapterConfig) *postgresAdapter {
	return &postgresAdapter{
		db:     config.Client,
		logger: config.Logger,
	}
}

func (p postgresAdapter) ListChannels(ctx context.Context, params notification.ListChannelsInput) (pagination.PagedResponse[notification.Channel], error) {
	query := p.db.NotificationChannel.Query().
		Where(channeldb.DeletedAtIsNil()) // Do not return deleted channels

	if len(params.Namespaces) > 0 {
		query = query.Where(channeldb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Channels) > 0 {
		query = query.Where(channeldb.IDIn(params.Channels...))
	}

	if !params.IncludeDisabled {
		query = query.Where(channeldb.Disabled(false))
	}

	order := entutils.GetOrdering(sortx.OrderDefault)
	if !params.Order.IsDefaultValue() {
		order = entutils.GetOrdering(params.Order)
	}

	switch params.OrderBy {
	case notification.ChannelOrderByCreatedAt:
		query = query.Order(channeldb.ByCreatedAt(order...))
	case notification.ChannelOrderByUpdatedAt:
		query = query.Order(channeldb.ByUpdatedAt(order...))
	case notification.ChannelOrderByType:
		query = query.Order(channeldb.ByType(order...))
	case notification.ChannelOrderByID:
		fallthrough
	default:
		query = query.Order(channeldb.ByID(order...))
	}

	response := pagination.PagedResponse[notification.Channel]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]notification.Channel, 0, len(paged.Items))
	for _, item := range paged.Items {
		if item == nil {
			p.logger.Warn("received nil channel for list query")
			continue
		}

		result = append(result, *ChannelFromDBEntity(*item))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (p postgresAdapter) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	query := p.db.NotificationChannel.Create().
		SetType(params.Type).
		SetName(params.Name).
		SetNamespace(params.Namespace).
		SetDisabled(params.Disabled).
		SetConfig(params.Config)

	channel, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel in database: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("invalid channel received from database: nil")
	}

	return ChannelFromDBEntity(*channel), nil
}

func (p postgresAdapter) DeleteChannel(ctx context.Context, params notification.DeleteChannelInput) error {
	query := p.db.NotificationChannel.Update().
		Where(channeldb.ID(params.ID)).
		Where(channeldb.Namespace(params.Namespace)).
		SetDeletedAt(clock.Now()).
		SetDisabled(true)

	count, err := query.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete channel in database: %w", err)
	}
	if count == 0 {
		return notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	return nil
}

func (p postgresAdapter) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	query := p.db.NotificationChannel.Query().
		Where(channeldb.ID(params.ID)).
		Where(channeldb.Namespace(params.Namespace))

	channelRow, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to get channel from database: %w", err)
	}

	if channelRow == nil {
		return nil, notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	return ChannelFromDBEntity(*channelRow), nil
}

func (p postgresAdapter) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	query := p.db.NotificationChannel.Update().
		Where(channeldb.ID(params.ID)).
		Where(channeldb.Namespace(params.Namespace)).
		SetUpdatedAt(clock.Now()).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		SetName(params.Name)

	count, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update channel in database: %w", err)
	}

	if count == 0 {
		return nil, notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	channel, err := p.GetChannel(ctx, notification.GetChannelInput{
		ID:        params.ID,
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated channel from database: %w", err)
	}

	return channel, nil
}

func (p postgresAdapter) ListRules(ctx context.Context, params notification.ListRulesInput) (pagination.PagedResponse[notification.Rule], error) {
	query := p.db.NotificationRule.Query().QueryChannels().QueryRules()

	if len(params.Namespaces) > 0 {
		query = query.Where(ruledb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Rules) > 0 {
		query = query.Where(ruledb.IDIn(params.Rules...))
	}

	if !params.IncludeDisabled {
		query = query.Where(ruledb.Disabled(false))
	}

	order := entutils.GetOrdering(sortx.OrderDefault)
	if !params.Order.IsDefaultValue() {
		order = entutils.GetOrdering(params.Order)
	}

	switch params.OrderBy {
	case notification.RuleOrderByCreatedAt:
		query = query.Order(ruledb.ByCreatedAt(order...))
	case notification.RuleOrderByUpdatedAt:
		query = query.Order(ruledb.ByUpdatedAt(order...))
	case notification.RuleOrderByType:
		query = query.Order(ruledb.ByType(order...))
	case notification.RuleOrderByID:
		fallthrough
	default:
		query = query.Order(ruledb.ByID(order...))
	}

	response := pagination.PagedResponse[notification.Rule]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]notification.Rule, 0, len(paged.Items))
	for _, ruleRow := range paged.Items {
		if ruleRow == nil {
			p.logger.Warn("received nil rule for list query")
			continue
		}

		rule := *RuleFromDBEntity(*ruleRow)

		var channelRows []*entdb.NotificationChannel
		channelRows, err = ruleRow.QueryChannels().All(ctx)
		if err != nil {
			return response, err
		}

		rule.Channels = make([]notification.Channel, 0, len(channelRows))
		for _, channel := range channelRows {
			if channel == nil {
				p.logger.Warn("received nil channel for rule")
				continue
			}

			rule.Channels = append(rule.Channels, *ChannelFromDBEntity(*channel))
		}

		result = append(result, rule)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (p postgresAdapter) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	channels, err := p.mustGetChannels(ctx, params.Namespace, params.Channels...)
	if err != nil {
		return nil, err
	}

	query := p.db.NotificationRule.Create().
		SetType(params.Type).
		SetName(params.Name).
		SetNamespace(params.Namespace).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		AddChannelIDs(params.Channels...)

	ruleRow, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule in database: %w", err)
	}

	if ruleRow == nil {
		return nil, fmt.Errorf("invalid rule received from database: nil")
	}

	rule := RuleFromDBEntity(*ruleRow)

	rule.Channels = channels

	return rule, nil
}

func (p postgresAdapter) mustGetChannels(ctx context.Context, namespace string, channels ...string) ([]notification.Channel, error) {
	resp, err := p.ListChannels(ctx, notification.ListChannelsInput{
		Namespaces:      []string{namespace},
		Channels:        channels,
		IncludeDisabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list channels from database: %w", err)
	}

	if len(resp.Items) != len(channels) {
		channelIDs := make(map[string]struct{}, len(resp.Items))
		for _, channel := range resp.Items {
			channelIDs[channel.ID] = struct{}{}
		}

		missingChannels := make([]string, 0)
		for _, channel := range channels {
			if _, ok := channelIDs[channel]; !ok {
				missingChannels = append(missingChannels, channel)
			}
		}

		return nil, notification.ValidationError{
			Err: fmt.Errorf("non-existing channels: %v", missingChannels),
		}
	}

	return resp.Items, err
}

func (p postgresAdapter) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	query := p.db.NotificationRule.Update().
		Where(ruledb.ID(params.ID)).
		Where(ruledb.Namespace(params.Namespace)).
		SetDeletedAt(clock.Now()).
		SetDisabled(true)

	count, err := query.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete rule in database: %w", err)
	}
	if count == 0 {
		return notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	return nil
}

func (p postgresAdapter) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	query := p.db.NotificationRule.Query().
		Where(ruledb.ID(params.ID)).
		Where(ruledb.Namespace(params.Namespace))

	ruleRow, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to get rule from database: %w", err)
	}

	if ruleRow == nil {
		return nil, notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	rule := RuleFromDBEntity(*ruleRow)

	channelRows, err := ruleRow.QueryChannels().All(ctx)

	rule.Channels = make([]notification.Channel, 0, len(channelRows))
	for _, channelRow := range channelRows {
		if channelRow == nil {
			p.logger.Warn("received nil channel for rule")
			continue
		}

		rule.Channels = append(rule.Channels, *ChannelFromDBEntity(*channelRow))
	}

	return rule, nil
}

func (p postgresAdapter) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	query := p.db.NotificationRule.Update().
		Where(ruledb.ID(params.ID)).
		Where(ruledb.Namespace(params.Namespace)).
		SetUpdatedAt(clock.Now()).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		SetName(params.Name)

	count, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update rule in database: %w", err)
	}

	if count == 0 {
		return nil, notification.NotFoundError{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		}
	}

	rule, err := p.GetRule(ctx, notification.GetRuleInput{
		ID:        params.ID,
		Namespace: params.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated rule from database: %w", err)
	}

	return rule, nil
}
