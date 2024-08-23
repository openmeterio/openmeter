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
	tx *entdb.Tx

	logger *slog.Logger
}

func (r repository) Commit() error {
	if r.tx != nil {
		return r.tx.Commit()
	}

	return nil
}

func (r repository) Rollback() error {
	if r.tx != nil {
		return r.tx.Rollback()
	}

	return nil
}

func (r repository) client() *entdb.Client {
	if r.tx != nil {
		return r.tx.Client()
	}

	return r.db
}

func (r repository) WithTx(ctx context.Context) (notification.TxRepository, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &repository{
		db:     r.db,
		tx:     tx,
		logger: r.logger,
	}, nil
}

func (r repository) ListChannels(ctx context.Context, params notification.ListChannelsInput) (pagination.PagedResponse[notification.Channel], error) {
	db := r.client()

	query := db.NotificationChannel.Query().
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
	db := r.client()

	query := db.NotificationChannel.Create().
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
	db := r.client()

	query := db.NotificationChannel.UpdateOneID(params.ID).
		SetDeletedAt(clock.Now().UTC()).
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
	db := r.client()

	query := db.NotificationChannel.Query().
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
	db := r.client()

	query := db.NotificationChannel.UpdateOneID(params.ID).
		SetUpdatedAt(clock.Now().UTC()).
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
	db := r.client()

	query := db.NotificationRule.Query().
		Where(ruledb.DeletedAtIsNil()) // Do not return deleted Rules

	if len(params.Namespaces) > 0 {
		query = query.Where(ruledb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Rules) > 0 {
		query = query.Where(ruledb.IDIn(params.Rules...))
	}

	if !params.IncludeDisabled {
		query = query.Where(ruledb.Disabled(false))
	}

	if len(params.Types) > 0 {
		query = query.Where(ruledb.TypeIn(params.Types...))
	}

	if len(params.Channels) > 0 {
		query = query.Where(ruledb.HasChannelsWith(channeldb.IDIn(params.Channels...)))
	}

	// Eager load Channels
	query = query.WithChannels()

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

		result = append(result, *RuleFromDBEntity(*ruleRow))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	db := r.client()

	query := db.NotificationRule.Create().
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

	channelsQuery := db.NotificationChannel.Query().
		Where(channeldb.Namespace(params.Namespace)).
		Where(channeldb.IDIn(params.Channels...))

	channelRows, err := channelsQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification channels: %w", err)
	}

	queryRow.Edges.Channels = channelRows

	return RuleFromDBEntity(*queryRow), nil
}

func (r repository) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	db := r.client()

	query := db.NotificationRule.UpdateOneID(params.ID).
		Where(ruledb.Namespace(params.Namespace)).
		SetDeletedAt(clock.Now().UTC()).
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
	db := r.client()

	query := db.NotificationRule.Query().
		Where(ruledb.ID(params.ID)).
		Where(ruledb.Namespace(params.Namespace)).
		WithChannels()

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

	return RuleFromDBEntity(*ruleRow), nil
}

func (r repository) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	db := r.client()

	query := db.NotificationRule.UpdateOneID(params.ID).
		SetUpdatedAt(clock.Now().UTC()).
		SetDisabled(params.Disabled).
		SetConfig(params.Config).
		SetName(params.Name).
		ClearChannels().
		AddChannelIDs(params.Channels...)

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

	channelsQuery := db.NotificationChannel.Query().
		Where(channeldb.Namespace(params.Namespace)).
		Where(channeldb.IDIn(params.Channels...))

	channelRows, err := channelsQuery.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification channels: %w", err)
	}

	queryRow.Edges.Channels = channelRows

	return RuleFromDBEntity(*queryRow), nil
}

func (r repository) ListEvents(ctx context.Context, params notification.ListEventsInput) (pagination.PagedResponse[notification.Event], error) {
	db := r.client()

	query := db.NotificationEvent.Query()

	if len(params.Namespaces) > 0 {
		query = query.Where(eventdb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Events) > 0 {
		query = query.Where(eventdb.IDIn(params.Events...))
	}

	if !params.From.IsZero() {
		query = query.Where(eventdb.CreatedAtGTE(params.From.UTC()))
	}

	if !params.To.IsZero() {
		query = query.Where(eventdb.CreatedAtLTE(params.To.UTC()))
	}

	if len(params.DeduplicationHashes) > 0 {
		query = query.Where(
			entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationEventDedupeHash, params.DeduplicationHashes),
		)
	}

	// Eager load DeliveryStatus, Rules (including Channels)
	if len(params.DeliveryStatusStates) > 0 {
		query = query.WithDeliveryStatuses(func(query *entdb.NotificationEventDeliveryStatusQuery) {
			query.Where(statusdb.StateIn(params.DeliveryStatusStates...))
		})
	} else {
		query = query.WithDeliveryStatuses()
	}

	if len(params.Features) > 0 {
		query = query.Where(
			eventdb.Or(
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationEventFeatureKey, params.Features),
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationEventFeatureID, params.Features),
			),
		)
	}

	if len(params.Subjects) > 0 {
		query = query.Where(
			eventdb.Or(
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationEventSubjectKey, params.Subjects),
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationEventSubjectID, params.Subjects),
			),
		)
	}

	if len(params.Rules) > 0 {
		query = query.Where(eventdb.RuleIDIn(params.Rules...))
	}

	if len(params.Channels) > 0 {
		query = query.Where(eventdb.HasRulesWith(ruledb.HasChannelsWith(channeldb.IDIn(params.Channels...))))
	}

	query = query.WithRules(func(query *entdb.NotificationRuleQuery) {
		query.WithChannels()
	})

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

		event, err := EventFromDBEntity(*eventRow)
		if err != nil {
			return response, fmt.Errorf("failed to get notification events: %w", err)
		}

		result = append(result, *event)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

func (r repository) GetEvent(ctx context.Context, params notification.GetEventInput) (*notification.Event, error) {
	db := r.client()

	query := db.NotificationEvent.Query().
		Where(eventdb.Namespace(params.Namespace)).
		Where(eventdb.ID(params.ID)).
		WithDeliveryStatuses().
		WithRules()

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

	event, err := EventFromDBEntity(*eventRow)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification event: %w", err)
	}

	return event, nil
}

func (r repository) CreateEvent(ctx context.Context, params notification.CreateEventInput) (*notification.Event, error) {
	payloadJSON, err := json.Marshal(params.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize notification event payload: %w", err)
	}

	db := r.client()

	query := db.NotificationEvent.Create().
		SetType(params.Type).
		SetNamespace(params.Namespace).
		SetRuleID(params.RuleID).
		SetPayload(string(payloadJSON)).
		SetAnnotations(params.Annotations)

	eventRow, err := query.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification event: %w", err)
	}

	if eventRow == nil {
		return nil, errors.New("invalid query response: nil notification event received")
	}

	ruleQuery := db.NotificationRule.Query().
		Where(ruledb.Namespace(params.Namespace)).
		Where(ruledb.ID(params.RuleID)).
		Where(ruledb.DeletedAtIsNil()).
		WithChannels()

	ruleRow, err := ruleQuery.First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.RuleID,
				},
			}
		}

		return nil, fmt.Errorf("failed to fetch notification rule: %w", err)
	}
	if ruleRow == nil {
		return nil, errors.New("invalid query result: nil notification rule received")
	}

	if _, err = ruleRow.Edges.ChannelsOrErr(); err != nil {
		return nil, fmt.Errorf("invalid query result: failed to load notification chnnaels for rule: %w", err)
	}

	eventRow.Edges.Rules = ruleRow

	statusBulkQuery := make([]*entdb.NotificationEventDeliveryStatusCreate, 0, len(ruleRow.Edges.Channels))
	for _, channel := range ruleRow.Edges.Channels {
		if channel == nil {
			r.logger.Warn("invalid query result: nil channel received")
			continue
		}

		q := db.NotificationEventDeliveryStatus.Create().
			SetNamespace(params.Namespace).
			SetEventID(eventRow.ID).
			SetChannelID(channel.ID).
			SetState(notification.EventDeliveryStatusStatePending).
			AddEvents(eventRow)

		statusBulkQuery = append(statusBulkQuery, q)
	}

	statusQuery := db.NotificationEventDeliveryStatus.CreateBulk(statusBulkQuery...)

	statusRows, err := statusQuery.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to save notification event: %w", err)
	}

	eventRow.Edges.DeliveryStatuses = statusRows

	event, err := EventFromDBEntity(*eventRow)
	if err != nil {
		return nil, fmt.Errorf("failed to cast notification event: %w", err)
	}

	return event, nil
}

func (r repository) ListEventsDeliveryStatus(ctx context.Context, params notification.ListEventsDeliveryStatusInput) (pagination.PagedResponse[notification.EventDeliveryStatus], error) {
	db := r.client()

	query := db.NotificationEventDeliveryStatus.Query()

	if len(params.Namespaces) > 0 {
		query = query.Where(statusdb.NamespaceIn(params.Namespaces...))
	}

	if len(params.Events) > 0 {
		query = query.Where(statusdb.EventIDIn(params.Events...))
	}

	if len(params.Channels) > 0 {
		query = query.Where(statusdb.ChannelIDIn(params.Channels...))
	}

	if len(params.States) > 0 {
		query = query.Where(statusdb.StateIn(params.States...))
	}

	if !params.From.IsZero() {
		query = query.Where(statusdb.UpdatedAtGTE(params.From.UTC()))
	}

	if !params.To.IsZero() {
		query = query.Where(statusdb.UpdatedAtLTE(params.To.UTC()))
	}

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
	db := r.client()

	query := db.NotificationEventDeliveryStatus.Query().
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

func (r repository) UpdateEventDeliveryStatus(ctx context.Context, params notification.UpdateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	var updateQuery *entdb.NotificationEventDeliveryStatusUpdateOne

	db := r.client()

	if params.ID != "" {
		updateQuery = db.NotificationEventDeliveryStatus.UpdateOneID(params.ID).SetState(params.State)
	} else {
		getQuery := db.NotificationEventDeliveryStatus.Query().
			Where(statusdb.Namespace(params.Namespace)).
			Where(statusdb.EventID(params.EventID)).
			Where(statusdb.ChannelID(params.ChannelID))

		statusRow, err := getQuery.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{
					NamespacedID: models.NamespacedID{
						Namespace: params.Namespace,
						ID:        params.EventID,
					},
				}
			}

			return nil, fmt.Errorf("failed to udpate notification event delivery status: %w", err)
		}

		updateQuery = db.NotificationEventDeliveryStatus.UpdateOne(statusRow).
			SetState(params.State).
			SetReason(params.Reason)
	}

	updateRow, err := updateQuery.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.EventID,
				},
			}
		}

		return nil, fmt.Errorf("failed to create notification event delivery status: %w", err)
	}

	if updateRow == nil {
		return nil, fmt.Errorf("invalid query response: no delivery status received")
	}

	return EventDeliveryStatusFromDBEntity(*updateRow), nil
}
