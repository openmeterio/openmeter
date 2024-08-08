package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/internal/ent/db"
	channeldb "github.com/openmeterio/openmeter/internal/ent/db/notificationchannel"
	eventdb "github.com/openmeterio/openmeter/internal/ent/db/notificationevent"
	statusdb "github.com/openmeterio/openmeter/internal/ent/db/notificationeventdeliverystatus"
	ruledb "github.com/openmeterio/openmeter/internal/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/internal/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (notification.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &repository{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ notification.Repository = (*repository)(nil)

type repository struct {
	db *entdb.Client

	logger *slog.Logger
}

func (r repository) ListChannels(ctx context.Context, params notification.ListChannelsInput) (pagination.PagedResponse[notification.Channel], error) {
	query := r.db.NotificationChannel.Query().
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
			r.logger.Warn("invalid query result: nil notification channel received")
			continue
		}

		result = append(result, *ChannelFromDBEntity(*item))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	query := r.db.NotificationChannel.Create().
		SetType(params.Type).
		SetName(params.Name).
		SetNamespace(params.Namespace).
		SetDisabled(params.Disabled).
		SetConfig(params.Config)

	channel, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification channel: %w", err)
	}

	if channel == nil {
		return nil, fmt.Errorf("invalid query result: nil notification channel received")
	}

	return ChannelFromDBEntity(*channel), nil
}

func (r repository) DeleteChannel(ctx context.Context, params notification.DeleteChannelInput) error {
	query := r.db.NotificationChannel.UpdateOneID(params.ID).
		SetDeletedAt(clock.Now()).
		SetDisabled(true)

	_, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	return nil
}

func (r repository) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	query := r.db.NotificationChannel.Query().
		Where(channeldb.ID(params.ID)).
		Where(channeldb.Namespace(params.Namespace))

	queryRow, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to fetch notification channel: %w", err)
	}

	if queryRow == nil {
		return nil, fmt.Errorf("invalid query result: nil notification channel received")
	}

	return ChannelFromDBEntity(*queryRow), nil
}

func (r repository) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	query := r.db.NotificationChannel.UpdateOneID(params.ID).
		SetUpdatedAt(clock.Now()).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		SetName(params.Name)

	queryRow, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to update notification channel: %w", err)
	}

	if queryRow == nil {
		return nil, fmt.Errorf("invalid query result: nil notification channel received")
	}

	return ChannelFromDBEntity(*queryRow), nil
}

func (r repository) ListRules(ctx context.Context, params notification.ListRulesInput) (pagination.PagedResponse[notification.Rule], error) {
	query := r.db.NotificationRule.Query().QueryChannels().QueryRules()

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
			r.logger.Warn("invalid query result: nil notification rule received")
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
				r.logger.Warn("invalid query result: nil notification channel received")
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

func (r repository) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	// FIXME: this needs to be reworked
	channels, err := r.mustGetChannels(ctx, params.Namespace, params.Channels...)
	if err != nil {
		return nil, err
	}

	query := r.db.NotificationRule.Create().
		SetType(params.Type).
		SetName(params.Name).
		SetNamespace(params.Namespace).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		AddChannelIDs(params.Channels...)

	queryRow, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification rule: %w", err)
	}

	if queryRow == nil {
		return nil, fmt.Errorf("invalid query result: nil notification rule received")
	}

	rule := RuleFromDBEntity(*queryRow)

	rule.Channels = channels

	return rule, nil
}

func (r repository) mustGetChannels(ctx context.Context, namespace string, channels ...string) ([]notification.Channel, error) {
	resp, err := r.ListChannels(ctx, notification.ListChannelsInput{
		Namespaces:      []string{namespace},
		Channels:        channels,
		IncludeDisabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failedto fetch notification channels: %w", err)
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
			Err: fmt.Errorf("unkown notification channels: %v", missingChannels),
		}
	}

	return resp.Items, err
}

func (r repository) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	query := r.db.NotificationRule.UpdateOneID(params.ID).
		Where(ruledb.Namespace(params.Namespace)).
		SetDeletedAt(clock.Now()).
		SetDisabled(true)

	_, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return fmt.Errorf("failed top delete notification rule: %w", err)
	}

	return nil
}

func (r repository) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	query := r.db.NotificationRule.Query().
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

		return nil, fmt.Errorf("failed to fetch notification rule: %w", err)
	}

	if ruleRow == nil {
		return nil, fmt.Errorf("invalid query result: nil notification rule received")
	}

	rule := RuleFromDBEntity(*ruleRow)

	channelRows, err := ruleRow.QueryChannels().All(ctx)

	rule.Channels = make([]notification.Channel, 0, len(channelRows))
	for _, channelRow := range channelRows {
		if channelRow == nil {
			r.logger.Warn("invalid query result: nil notification channel received")
			continue
		}

		rule.Channels = append(rule.Channels, *ChannelFromDBEntity(*channelRow))
	}

	return rule, nil
}

func (r repository) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	query := r.db.NotificationRule.UpdateOneID(params.ID).
		SetUpdatedAt(clock.Now()).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		SetName(params.Name)

	queryRow, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to update notification rule: %w", err)
	}

	if queryRow == nil {
		return nil, fmt.Errorf("invalid query result: nil notification rule received")
	}

	return RuleFromDBEntity(*queryRow), nil
}

func (r repository) ListEvents(ctx context.Context, params notification.ListEventsInput) (pagination.PagedResponse[notification.Event], error) {
	query := r.db.NotificationEvent.Query().QueryDeliveryStatuses().QueryEvents()

	if len(params.Namespaces) > 0 {
		query = query.Where(eventdb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Events) > 0 {
		query = query.Where(eventdb.IDIn(params.Events...))
	}

	order := entutils.GetOrdering(sortx.OrderDefault)
	if !params.Order.IsDefaultValue() {
		order = entutils.GetOrdering(params.Order)
	}

	switch params.OrderBy {
	case notification.EventOrderByCreatedAt:
		query = query.Order(eventdb.ByCreatedAt(order...))
	case notification.EventOrderByID:
		fallthrough
	default:
		query = query.Order(eventdb.ByID(order...))
	}

	response := pagination.PagedResponse[notification.Event]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]notification.Event, 0, len(paged.Items))
	for _, eventRow := range paged.Items {
		if eventRow == nil {
			r.logger.Warn("invalid query result: nil notification event received")
			continue
		}

		event := *EventFromDBEntity(*eventRow)

		var statusRows []*entdb.NotificationEventDeliveryStatus
		statusRows, err = eventRow.QueryDeliveryStatuses().All(ctx)
		if err != nil {
			return response, err
		}

		event.DeliveryStatus = make([]notification.EventDeliveryStatus, 0, len(statusRows))
		for _, status := range statusRows {
			if status == nil {
				r.logger.Warn("invalid query result: nil notification event delivery status received")
				continue
			}

			event.DeliveryStatus = append(event.DeliveryStatus, *EventDeliveryStatusFromDBEntity(*status))
		}

		result = append(result, event)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) GetEvent(ctx context.Context, params notification.GetEventInput) (*notification.Event, error) {
	query := r.db.NotificationEvent.Query().
		Where(eventdb.Namespace(params.Namespace)).
		Where(eventdb.ID(params.ID))

	eventRow, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to get notification event: %w", err)
	}

	if eventRow == nil {
		return nil, errors.New("invalid query response: nil notification event received")
	}

	return EventFromDBEntity(*eventRow), nil
}

func (r repository) CreateEvent(ctx context.Context, params notification.CreateEventInput) (*notification.Event, error) {
	ruleJSON, err := json.Marshal(params.Rule)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize notification rule: %w", err)
	}

	payloadJSON, err := json.Marshal(params.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize notification event payload: %w", err)
	}

	query := r.db.NotificationEvent.Create().
		SetType(params.Type).
		SetNamespace(params.Namespace).
		SetRule(string(ruleJSON)).
		SetPayload(string(payloadJSON))

	eventRow, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification event: %w", err)
	}

	if eventRow == nil {
		return nil, errors.New("invalid query response: nil notification event received")
	}

	return EventFromDBEntity(*eventRow), nil
}

func (r repository) ListEventsDeliveryStatus(ctx context.Context, params notification.ListEventsDeliveryStatusInput) (pagination.PagedResponse[notification.EventDeliveryStatus], error) {
	query := r.db.NotificationEventDeliveryStatus.Query()

	if len(params.Namespaces) > 0 {
		query = query.Where(statusdb.NamespaceIn(params.Namespaces...))
	}

	if len(params.EventIDs) > 0 {
		query = query.Where(statusdb.EventIDIn(params.EventIDs...))
	}

	if len(params.ChannelIDs) > 0 {
		query = query.Where(statusdb.ChannelIDIn(params.ChannelIDs...))
	}

	// FIXME: add from-to

	response := pagination.PagedResponse[notification.EventDeliveryStatus]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]notification.EventDeliveryStatus, 0, len(paged.Items))
	for _, statusRow := range paged.Items {
		if statusRow == nil {
			r.logger.Warn("invalid query response: nil notification event delivery status received")
			continue
		}

		result = append(result, *EventDeliveryStatusFromDBEntity(*statusRow))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) GetEventDeliveryStatus(ctx context.Context, params notification.GetEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	query := r.db.NotificationEventDeliveryStatus.Query().
		Where(statusdb.Namespace(params.Namespace)).
		Where(statusdb.ID(params.ID))

	queryRow, err := query.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to get notification event delivery status: %w", err)
	}
	if queryRow == nil {
		return nil, errors.New("invalid query response: no delivery status received")
	}

	return EventDeliveryStatusFromDBEntity(*queryRow), nil
}

func (r repository) CreateEventDeliveryStatus(ctx context.Context, params notification.CreateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	query := r.db.NotificationEventDeliveryStatus.Create().
		SetNamespace(params.Namespace).
		SetEventID(params.EventID).
		SetState(notification.EventDeliveryStatusStateSending).
		AddEventIDs(params.EventID)

	queryRow, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification event delivery status: %w", err)
	}
	if queryRow == nil {
		return nil, fmt.Errorf("invalid query response: no delivery status received")
	}

	return EventDeliveryStatusFromDBEntity(*queryRow), nil
}

func (r repository) UpdateEventDeliveryStatus(ctx context.Context, params notification.UpdateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	query := r.db.NotificationEventDeliveryStatus.UpdateOneID(params.ID).
		SetState(notification.EventDeliveryStatusStateSending)

	queryRow, err := query.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.ID,
				},
			}
		}

		return nil, fmt.Errorf("failed to create notification event delivery status: %w", err)
	}

	if queryRow == nil {
		return nil, fmt.Errorf("invalid query response: no delivery status received")
	}

	return EventDeliveryStatusFromDBEntity(*queryRow), nil
}
