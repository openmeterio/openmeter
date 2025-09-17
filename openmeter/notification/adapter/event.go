package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
	eventdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationevent"
	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	ruledb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListEvents(ctx context.Context, params notification.ListEventsInput) (pagination.Result[notification.Event], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[notification.Event], error) {
		query := a.db.NotificationEvent.Query()

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
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationBalanceEventDedupeHash, params.DeduplicationHashes),
			)
		}

		if len(params.DeliveryStatusStates) > 0 {
			query = query.Where(eventdb.HasDeliveryStatusesWith(statusdb.StateIn(params.DeliveryStatusStates...)))
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

		query = query.
			WithRules(func(query *entdb.NotificationRuleQuery) {
				query.WithChannels()
			}).
			WithDeliveryStatuses()

		order := entutils.GetOrdering(sortx.OrderDesc)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case notification.OrderByID:
			query = query.Order(eventdb.ByID(order...))
		case notification.OrderByCreatedAt:
			fallthrough
		default:
			query = query.Order(eventdb.ByCreatedAt(order...))
		}

		response := pagination.Result[notification.Event]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, err
		}

		result := make([]notification.Event, 0, len(paged.Items))
		for _, eventRow := range paged.Items {
			if eventRow == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil notification event received")
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

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) GetEvent(ctx context.Context, params notification.GetEventInput) (*notification.Event, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Event, error) {
		query := a.db.NotificationEvent.Query().
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

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) CreateEvent(ctx context.Context, params notification.CreateEventInput) (*notification.Event, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Event, error) {
		payloadJSON, err := json.Marshal(params.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize notification event payload: %w", err)
		}

		query := a.db.NotificationEvent.Create().
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

		ruleQuery := a.db.NotificationRule.Query().
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
			return nil, fmt.Errorf("invalid query result: failed to load notification channels for rule: %w", err)
		}

		eventRow.Edges.Rules = ruleRow

		statusBulkQuery := make([]*entdb.NotificationEventDeliveryStatusCreate, 0, len(ruleRow.Edges.Channels))
		for _, channel := range ruleRow.Edges.Channels {
			if channel == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil channel received")
				continue
			}

			q := a.db.NotificationEventDeliveryStatus.Create().
				SetNamespace(params.Namespace).
				SetEventID(eventRow.ID).
				SetChannelID(channel.ID).
				SetState(notification.EventDeliveryStatusStatePending).
				AddEvents(eventRow)

			statusBulkQuery = append(statusBulkQuery, q)
		}

		statusQuery := a.db.NotificationEventDeliveryStatus.CreateBulk(statusBulkQuery...)

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

	return entutils.TransactingRepo(ctx, a, fn)
}
