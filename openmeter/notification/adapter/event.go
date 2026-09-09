package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
	eventdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationevent"
	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	ruledb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func EagerLoadRulesWithActiveChannels(at time.Time) func(query *entdb.NotificationRuleQuery) {
	return func(query *entdb.NotificationRuleQuery) {
		query.WithChannels(func(cq *entdb.NotificationChannelQuery) {
			cq.Where(
				channeldb.Disabled(false),
				channeldb.Or(
					channeldb.DeletedAtIsNil(),
					channeldb.DeletedAtGT(at),
				),
			)
		})
	}
}

func (a *adapter) ListEvents(ctx context.Context, params notification.ListEventsInput) (pagination.Result[notification.Event], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[notification.Event], error) {
		query := a.db.NotificationEvent.Query().
			WithRules(EagerLoadRulesWithActiveChannels(clock.Now())).
			WithDeliveryStatuses()

		if len(params.Namespaces) > 0 {
			query = query.Where(eventdb.NamespaceIn(params.Namespaces...))
		}

		query = filter.ApplyToQuery(query, params.ID, eventdb.FieldID)
		query = filter.ApplyToQuery(query, params.Type, eventdb.FieldType)
		query = filter.ApplyToQuery(query, params.CreatedAt, eventdb.FieldCreatedAt)
		query = filter.ApplyToQuery(query, params.RuleID, eventdb.FieldRuleID)

		// Both edge filters are existential: they match events that have at least one
		// matching delivery status / channel. Negation would silently invert to "has some
		// other status/channel", so the API layer rejects it before we get here.
		var statusPreds []predicate.NotificationEventDeliveryStatus
		statusPreds = filter.ApplyToPredicate(statusPreds, params.DeliveryStatus, statusdb.FieldState)
		if len(statusPreds) > 0 {
			query = query.Where(eventdb.HasDeliveryStatusesWith(statusPreds...))
		}

		var channelPreds []predicate.NotificationChannel
		channelPreds = filter.ApplyToPredicate(channelPreds, params.ChannelID, channeldb.FieldID)
		if len(channelPreds) > 0 {
			query = query.Where(eventdb.HasRulesWith(ruledb.HasChannelsWith(channelPreds...)))
		}

		query = filter.ApplyToQueryJSONB(query, params.SubjectKey, eventdb.FieldAnnotations, notification.AnnotationEventSubjectKey)
		query = filter.ApplyToQueryJSONB(query, params.SubjectID, eventdb.FieldAnnotations, notification.AnnotationEventSubjectID)
		query = filter.ApplyToQueryJSONB(query, params.FeatureKey, eventdb.FieldAnnotations, notification.AnnotationEventFeatureKey)
		query = filter.ApplyToQueryJSONB(query, params.FeatureID, eventdb.FieldAnnotations, notification.AnnotationEventFeatureID)

		if len(params.DeduplicationHashes) > 0 {
			query = query.Where(
				entutils.JSONBIn(eventdb.FieldAnnotations, notification.AnnotationBalanceEventDedupeHash, params.DeduplicationHashes),
			)
		}

		if !params.NextAttemptBefore.IsZero() {
			query = query.Where(eventdb.HasDeliveryStatusesWith(
				statusdb.StateNotIn(notification.EventDeliveryStatusStateSuccess, notification.EventDeliveryStatusStateFailed),
				statusdb.Or(
					statusdb.NextAttemptAtIsNil(),
					statusdb.NextAttemptAtLTE(params.NextAttemptBefore.UTC()),
				),
			))
		}

		order := entutils.GetOrdering(sortx.OrderDesc)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case notification.OrderByCreatedAt:
			query = query.Order(eventdb.ByCreatedAt(order...))
		case notification.OrderByType:
			query = query.Order(eventdb.ByType(order...))
		case notification.OrderByID:
			fallthrough
		default:
			query = query.Order(eventdb.ByID(order...))
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
			WithRules(EagerLoadRulesWithActiveChannels(clock.Now()))

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
			Where(ruledb.Or(
				ruledb.DeletedAtIsNil(),
				ruledb.DeletedAtGT(clock.Now()),
			)).
			WithChannels(EagerLoadActiveChannels(clock.Now()))

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

		eventRow.Edges.Rules = ruleRow

		if _, err = ruleRow.Edges.ChannelsOrErr(); err != nil {
			return nil, fmt.Errorf("invalid query result: failed to load notification channels for rule: %w", err)
		}

		// Create delivery statuses for each channel
		if len(ruleRow.Edges.Channels) > 0 {
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
		}

		event, err := EventFromDBEntity(*eventRow)
		if err != nil {
			return nil, fmt.Errorf("failed to cast notification event: %w", err)
		}

		return event, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}
