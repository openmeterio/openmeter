package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
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
			query = query.Where(statusdb.StateIn(toDBDeliveryStates(params.States...)...))
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

			status, err := EventDeliveryStatusFromDBEntity(*statusRow)
			if err != nil {
				return response, err
			}

			result = append(result, *status)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

// TODO(notification-v4): retire the legacy spellings once all writers and stored
// rows use lowercase and the service accepts the final state representation.
func deliveryStateSpellings(state notification.EventDeliveryStatusState) (legacy, lowercase notification.EventDeliveryStatusState) {
	switch state {
	case notification.EventDeliveryStatusStatePending, "pending":
		return notification.EventDeliveryStatusStatePending, "pending"
	case notification.EventDeliveryStatusStateSending, "sending":
		return notification.EventDeliveryStatusStateSending, "sending"
	case notification.EventDeliveryStatusStateResending, "resending":
		return notification.EventDeliveryStatusStateResending, "resending"
	case notification.EventDeliveryStatusStateSuccess, "success":
		return notification.EventDeliveryStatusStateSuccess, "success"
	case notification.EventDeliveryStatusStateFailed, "failed":
		return notification.EventDeliveryStatusStateFailed, "failed"
	default:
		return state, state
	}
}

// The Event filter uses strings, while Ent predicates use EventDeliveryStatusState.
func toDBDeliveryStates[T ~string](states ...T) []T {
	values := make([]T, 0, 2*len(states))
	for _, state := range states {
		legacy, lowercase := deliveryStateSpellings(notification.EventDeliveryStatusState(state))
		values = append(values, T(legacy))

		if legacy != lowercase {
			values = append(values, T(lowercase))
		}
	}

	return values
}

func toDBDeliveryStateFilter(input *filter.FilterString) *filter.FilterString {
	if input == nil {
		return nil
	}

	switch {
	case input.Eq != nil:
		values := toDBDeliveryStates(*input.Eq)
		return &filter.FilterString{In: &values}
	case input.In != nil:
		values := toDBDeliveryStates((*input.In)...)
		return &filter.FilterString{In: &values}
	default:
		return input
	}
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

		return EventDeliveryStatusFromDBEntity(*queryRow)
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

		attempts, err := json.Marshal(params.Attempts)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize notification event delivery attempts: %w", err)
		}

		query := a.db.NotificationEventDeliveryStatus.UpdateOneID(params.ID).
			Where(statusdb.Namespace(params.Namespace)).
			SetState(params.State).
			SetReason(params.Reason).
			SetAnnotations(params.Annotations).
			SetOrClearNextAttemptAt(nextAttempt).
			SetAttempts(attempts)

		row, err := query.Save(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, notification.NotFoundError{
					NamespacedID: params.NamespacedID,
				}
			}

			return nil, fmt.Errorf("failed to update notification event delivery status: %w", err)
		}

		if row == nil {
			return nil, fmt.Errorf("invalid query response: no delivery status received")
		}

		return EventDeliveryStatusFromDBEntity(*row)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}
