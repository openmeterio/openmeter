package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
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

		typeFilter, err := toDBChannelTypeFilter(params.Type)
		if err != nil {
			return pagination.Result[notification.Channel]{}, fmt.Errorf("invalid channel type filter: %w", err)
		}

		query = filter.ApplyToQuery(query, params.ID, channeldb.FieldID)
		query = filter.ApplyToQuery(query, params.Name, channeldb.FieldName)
		query = filter.ApplyToQuery(query, typeFilter, channeldb.FieldType)

		query = filter.ApplyToQuery(query, params.Disabled, channeldb.FieldDisabled)
		query = filter.ApplyToQuery(query, params.CreatedAt, channeldb.FieldCreatedAt)
		query = filter.ApplyToQuery(query, params.UpdatedAt, channeldb.FieldUpdatedAt)

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

			channel, err := ChannelFromDBEntity(*item)
			if err != nil {
				return response, err
			}

			result = append(result, *channel)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) CreateChannel(ctx context.Context, params notification.CreateChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		config, err := toDBChannelConfig(params.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to encode notification channel config: %w", err)
		}

		query := a.db.NotificationChannel.Create().
			SetType(params.Type).
			SetName(params.Name).
			SetNamespace(params.Namespace).
			SetDisabled(params.Disabled).
			SetConfig(config).
			SetAnnotations(params.Annotations).
			SetMetadata(params.Metadata)

		channel, err := query.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create notification channel: %w", err)
		}

		if channel == nil {
			return nil, fmt.Errorf("invalid query result: nil notification channel received")
		}

		return ChannelFromDBEntity(*channel)
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

		return ChannelFromDBEntity(*queryRow)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) UpdateChannel(ctx context.Context, params notification.UpdateChannelInput) (*notification.Channel, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Channel, error) {
		config, err := toDBChannelConfig(params.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to encode notification channel config: %w", err)
		}

		query := a.db.NotificationChannel.UpdateOneID(params.ID).
			Where(channeldb.Namespace(params.Namespace)).
			SetDisabled(params.Disabled).
			SetConfig(config).
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

		return ChannelFromDBEntity(*queryRow)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

// fromDBChannelConfig uses the row's type to decode flat configs, which carry no
// discriminator. Keep legacy envelope support until Phase 2 conversion is complete.
func fromDBChannelConfig(channelType notification.ChannelType, data json.RawMessage) (notification.ChannelConfig, error) {
	config := notification.ChannelConfig{
		ChannelConfigMeta: notification.ChannelConfigMeta{Type: channelType},
	}

	if channelType != notification.ChannelTypeWebhook {
		return notification.ChannelConfig{}, fmt.Errorf("invalid channel type: %s", channelType)
	}

	// TODO(notification-v4): remove envelope decoding after all writers and
	// stored configs use the flat representation.
	var legacy struct {
		Type notification.ChannelType           `json:"type"`
		Data *notification.WebHookChannelConfig `json:"data"`
	}

	legacyErr := json.Unmarshal(data, &legacy)
	if legacyErr == nil {
		switch {
		case legacy.Type != channelType && legacy.Type != "WEBHOOK":
			legacyErr = fmt.Errorf("invalid channel config discriminator")
		case legacy.Data == nil:
			legacyErr = fmt.Errorf("invalid channel config envelope: missing data")
		default:
			config.WebHook = *legacy.Data

			return config, nil
		}
	}

	// Unmarshal ignores missing fields, so a flat config also needs its URL
	// to distinguish it from an empty object or an invalid envelope.
	var webhook struct {
		URL           string            `json:"url"`
		CustomHeaders map[string]string `json:"custom_headers"`
		SigningSecret string            `json:"signing_secret"`
	}

	flatErr := json.Unmarshal(data, &webhook)
	if flatErr == nil && webhook.URL == "" {
		flatErr = fmt.Errorf("invalid flat webhook config: missing URL")
	}

	if flatErr != nil {
		return notification.ChannelConfig{}, fmt.Errorf("invalid channel config: envelope: %w; flat: %w", legacyErr, flatErr)
	}

	config.WebHook = notification.WebHookChannelConfig{
		URL:           webhook.URL,
		CustomHeaders: webhook.CustomHeaders,
		SigningSecret: webhook.SigningSecret,
	}

	return config, nil
}

func toDBChannelConfig(config notification.ChannelConfig) (json.RawMessage, error) {
	if config.Type != notification.ChannelTypeWebhook {
		return nil, fmt.Errorf("invalid channel type: %s", config.Type)
	}

	// TODO(notification-v4): switch to flat config writes in Phase 2. Old
	// backends still require this envelope and its camelCase field names.
	encoded, err := json.Marshal(struct {
		Type notification.ChannelType          `json:"type"`
		Data notification.WebHookChannelConfig `json:"data"`
	}{
		Type: "WEBHOOK",
		Data: config.WebHook,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode webhook config: %w", err)
	}

	return encoded, nil
}

// toDBChannelTypeFilter expands canonical types into their stored spellings.
// TODO(notification-v4): remove this mapping after all writers and rows use the final format.
func toDBChannelTypeFilter(input *filter.FilterString) (*filter.FilterString, error) {
	if input == nil {
		return nil, nil
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	expand := func(values []string) ([]string, error) {
		result := make([]string, 0, len(values)*2)
		for _, value := range values {
			switch value {
			case string(notification.ChannelTypeWebhook):
				result = append(result, value, "WEBHOOK")
			case "WEBHOOK":
				return nil, models.NewGenericValidationError(fmt.Errorf("invalid channel type filter value: use %q", notification.ChannelTypeWebhook))
			default:
				result = append(result, value)
			}
		}

		return result, nil
	}

	mapped := &filter.FilterString{}
	var err error

	switch {
	case input.IsEmpty():
		return mapped, nil
	case input.Eq != nil:
		values, expandErr := expand([]string{*input.Eq})
		mapped.In, err = &values, expandErr
	case input.Ne != nil:
		values, expandErr := expand([]string{*input.Ne})
		mapped.Nin, err = &values, expandErr
	case input.In != nil:
		values, expandErr := expand(*input.In)
		mapped.In, err = &values, expandErr
	case input.Nin != nil:
		values, expandErr := expand(*input.Nin)
		mapped.Nin, err = &values, expandErr
	case input.Exists != nil:
		mapped.Exists = new(*input.Exists)
	case input.And != nil, input.Or != nil:
		children := input.And
		if children == nil {
			children = input.Or
		}

		converted := make([]filter.FilterString, 0, len(*children))
		for _, child := range *children {
			mappedChild, childErr := toDBChannelTypeFilter(&child)
			if childErr != nil {
				return nil, childErr
			}

			converted = append(converted, *mappedChild)
		}

		if input.And != nil {
			mapped.And = &converted
		} else {
			mapped.Or = &converted
		}
	default:
		return nil, models.NewGenericValidationError(fmt.Errorf("invalid channel type filter: only exact and existence operators are supported"))
	}

	if err != nil {
		return nil, err
	}

	return mapped, nil
}
