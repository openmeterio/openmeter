package adapter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (a *adapter) ListEventsDeliveryStatus(ctx context.Context, params notification.ListEventsDeliveryStatusInput) (pagination.Result[notification.EventDeliveryStatus], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[notification.EventDeliveryStatus], error) {
		query := a.db.NotificationEventDeliveryStatus.Query()

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

		response := pagination.Result[notification.EventDeliveryStatus]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, err
		}

		result := make([]notification.EventDeliveryStatus, 0, len(paged.Items))
		for _, statusRow := range paged.Items {
			if statusRow == nil {
				a.logger.WarnContext(ctx, "invalid query response: nil notification event delivery status received")
				continue
			}

			result = append(result, *EventDeliveryStatusFromDBEntity(*statusRow))
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) GetEventDeliveryStatus(ctx context.Context, params notification.GetEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.EventDeliveryStatus, error) {
		query := a.db.NotificationEventDeliveryStatus.Query().
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

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) UpdateEventDeliveryStatus(ctx context.Context, params notification.UpdateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.EventDeliveryStatus, error) {
		nextAttempt := func() *time.Time {
			if params.NextAttempt == nil {
				return nil
			}

			return lo.ToPtr(lo.FromPtr(params.NextAttempt).UTC())
		}()

		query := a.db.NotificationEventDeliveryStatus.UpdateOneID(params.ID).
			Where(statusdb.Namespace(params.Namespace)).
			SetState(params.State).
			SetReason(params.Reason).
			SetAnnotations(params.Annotations).
			SetOrClearNextAttemptAt(nextAttempt).
			SetAttempts(params.Attempts)

		row, err := query.Save(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{
					NamespacedID: params.NamespacedID,
				}
			}

			return nil, fmt.Errorf("failed to create notification event delivery status: %w", err)
		}

		if row == nil {
			return nil, fmt.Errorf("invalid query response: no delivery status received")
		}

		return EventDeliveryStatusFromDBEntity(*row), nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}
