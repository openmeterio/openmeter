package adapter

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func (a *adapter) ListChannels(ctx context.Context, params notification.ListChannelsInput) (pagination.Result[notification.Channel], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[notification.Channel], error) {
		query := a.db.NotificationChannel.Query().
			Where(channeldb.Or(
				channeldb.DeletedAtIsNil(),
				channeldb.DeletedAtGT(clock.Now()),
			)) // Do not return deleted channels

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
		case notification.OrderByCreatedAt:
			query = query.Order(channeldb.ByCreatedAt(order...))
		case notification.OrderByUpdatedAt:
			query = query.Order(channeldb.ByUpdatedAt(order...))
		case notification.OrderByType:
			query = query.Order(channeldb.ByType(order...))
		case notification.OrderByID:
			fallthrough
		default:
			query = query.Order(channeldb.ByID(order...))
		}

		response := pagination.Result[notification.Channel]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, err
		}

		result := make([]notification.Channel, 0, len(paged.Items))
		for _, item := range paged.Items {
			if item == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil notification channel received")
				continue
			}

			result = append(result, *ChannelFromDBEntity(*item))
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		query := a.db.NotificationChannel.Create().
			SetType(params.Type).
			SetName(params.Name).
			SetNamespace(params.Namespace).
			SetDisabled(params.Disabled).
			SetConfig(params.Config).
			SetAnnotations(params.Annotations).
			SetMetadata(params.Metadata)

		channel, err := query.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create notification channel: %w", err)
		}

		if channel == nil {
			return nil, fmt.Errorf("invalid query result: nil notification channel received")
		}

		return ChannelFromDBEntity(*channel), nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) DeleteChannel(ctx context.Context, params notification.DeleteChannelInput) error {
	fn := func(ctx context.Context, a *adapter) error {
		query := a.db.NotificationChannel.UpdateOneID(params.ID).
			Where(channeldb.Namespace(params.Namespace)).
			SetDisabled(true).
			SetDeletedAt(clock.Now())

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

	return entutils.TransactingRepoWithNoValue(ctx, a, fn)
}

func (a *adapter) GetChannel(ctx context.Context, params notification.GetChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		query := a.db.NotificationChannel.Query().
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

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		query := a.db.NotificationChannel.UpdateOneID(params.ID).
			Where(channeldb.Namespace(params.Namespace)).
			SetDisabled(params.Disabled).
			SetConfig(params.Config).
			SetName(params.Name).
			SetMetadata(params.Metadata).
			SetAnnotations(params.Annotations)

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

	return entutils.TransactingRepo(ctx, a, fn)
}
