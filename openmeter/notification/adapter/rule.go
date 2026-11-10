package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
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

func EagerLoadActiveChannels(at time.Time) func(query *entdb.NotificationChannelQuery) {
	return func(query *entdb.NotificationChannelQuery) {
		query.Where(
			channeldb.Disabled(false),
			channeldb.Or(
				channeldb.DeletedAtIsNil(),
				channeldb.DeletedAtGT(at),
			),
		)
	}
}

func (a *adapter) ListRules(ctx context.Context, params notification.ListRulesInput) (pagination.Result[notification.Rule], error) {
	fn := func(ctx context.Context, a *adapter) (pagination.Result[notification.Rule], error) {
		now := clock.Now()

		query := a.db.NotificationRule.Query().
			Where(ruledb.Or(
				ruledb.DeletedAtIsNil(),
				ruledb.DeletedAtGT(now),
			)).                                        // Do not return deleted Rules
			WithChannels(EagerLoadActiveChannels(now)) // Eager load active Channels

		query = query.Where(ruledb.Namespace(params.Namespace))

		if !params.IncludeDisabled {
			query = query.Where(ruledb.Disabled(false))
		}

		if len(params.Channels) > 0 {
			query = query.Where(ruledb.HasChannelsWith(channeldb.IDIn(params.Channels...)))
		}

		query = filter.ApplyToQuery(query, params.ID, ruledb.FieldID)
		query = filter.ApplyToQuery(query, params.Name, ruledb.FieldName)
		query = filter.ApplyToQuery(query, params.Type, ruledb.FieldType)
		query = filter.ApplyToQuery(query, params.Disabled, ruledb.FieldDisabled)
		query = filter.ApplyToQuery(query, params.CreatedAt, ruledb.FieldCreatedAt)
		query = filter.ApplyToQuery(query, params.UpdatedAt, ruledb.FieldUpdatedAt)

		var channelPreds []predicate.NotificationChannel
		channelPreds = filter.ApplyToPredicate(channelPreds, params.ChannelID, channeldb.FieldID)
		if len(channelPreds) > 0 {
			query = query.Where(ruledb.HasChannelsWith(channelPreds...))
		}

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}

		switch params.OrderBy {
		case notification.OrderByCreatedAt:
			query = query.Order(ruledb.ByCreatedAt(order...))
		case notification.OrderByUpdatedAt:
			query = query.Order(ruledb.ByUpdatedAt(order...))
		case notification.OrderByType:
			query = query.Order(ruledb.ByType(order...))
		case notification.OrderByID:
			fallthrough
		default:
			query = query.Order(ruledb.ByID(order...))
		}

		response := pagination.Result[notification.Rule]{
			Page: params.Page,
		}

		paged, err := query.Paginate(ctx, params.Page)
		if err != nil {
			return response, err
		}

		result := make([]notification.Rule, 0, len(paged.Items))
		for _, ruleRow := range paged.Items {
			if ruleRow == nil {
				a.logger.WarnContext(ctx, "invalid query result: nil notification rule received")
				continue
			}

			rule, err := RuleFromDBEntity(*ruleRow)
			if err != nil {
				return response, err
			}

			result = append(result, *rule)
		}

		response.TotalCount = paged.TotalCount
		response.Items = result

		return response, nil
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) CreateRule(ctx context.Context, params notification.CreateRuleInput) (*notification.Rule, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Rule, error) {
		config, err := toDBRuleConfig(params.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to encode notification rule config: %w", err)
		}

		query := a.db.NotificationRule.Create().
			SetType(params.Type).
			SetName(params.Name).
			SetNamespace(params.Namespace).
			SetDisabled(params.Disabled).
			SetConfig(config).
			SetAnnotations(params.Annotations).
			SetMetadata(params.Metadata).
			AddChannelIDs(params.Channels...)

		queryRow, err := query.Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create notification rule: %w", err)
		}

		if queryRow == nil {
			return nil, fmt.Errorf("invalid query result: nil notification rule received")
		}

		channelsQuery := a.db.NotificationChannel.Query().
			Where(channeldb.Namespace(params.Namespace)).
			Where(channeldb.IDIn(params.Channels...))

		channelRows, err := channelsQuery.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query notification channels: %w", err)
		}

		queryRow.Edges.Channels = channelRows

		return RuleFromDBEntity(*queryRow)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) DeleteRule(ctx context.Context, params notification.DeleteRuleInput) error {
	fn := func(ctx context.Context, a *adapter) error {
		query := a.db.NotificationRule.UpdateOneID(params.ID).
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

	return entutils.TransactingRepoWithNoValue(ctx, a, fn)
}

func (a *adapter) GetRule(ctx context.Context, params notification.GetRuleInput) (*notification.Rule, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Rule, error) {
		query := a.db.NotificationRule.Query().
			Where(ruledb.ID(params.ID)).
			Where(ruledb.Namespace(params.Namespace)).
			WithChannels(EagerLoadActiveChannels(clock.Now()))

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

		return RuleFromDBEntity(*ruleRow)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

func (a *adapter) UpdateRule(ctx context.Context, params notification.UpdateRuleInput) (*notification.Rule, error) {
	fn := func(ctx context.Context, a *adapter) (*notification.Rule, error) {
		config, err := toDBRuleConfig(params.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to encode notification rule config: %w", err)
		}

		query := a.db.NotificationRule.UpdateOneID(params.ID).
			Where(ruledb.Namespace(params.Namespace)).
			SetDisabled(params.Disabled).
			SetConfig(config).
			SetName(params.Name).
			SetAnnotations(params.Annotations).
			SetMetadata(params.Metadata).
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

		channelsQuery := a.db.NotificationChannel.Query().
			Where(channeldb.Namespace(params.Namespace)).
			Where(channeldb.IDIn(params.Channels...))

		channelRows, err := channelsQuery.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query notification channels: %w", err)
		}

		queryRow.Edges.Channels = channelRows

		return RuleFromDBEntity(*queryRow)
	}

	return entutils.TransactingRepo(ctx, a, fn)
}

// fromDBRuleConfig uses the row's type for unwrapped configs. Decoding does not
// revalidate historical rule conditions; validation belongs to the write path.
func fromDBRuleConfig(eventType notification.EventType, data json.RawMessage) (notification.RuleConfig, error) {
	config := notification.RuleConfig{
		RuleConfigMeta: notification.RuleConfigMeta{Type: eventType},
	}

	var err error

	switch eventType {
	case notification.EventTypeBalanceThreshold:
		config.BalanceThreshold, err = fromDBRuleConfigPayload[notification.BalanceThresholdRuleConfig](eventType, data)
	case notification.EventTypeEntitlementReset:
		config.EntitlementReset, err = fromDBRuleConfigPayload[notification.EntitlementResetRuleConfig](eventType, data)
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		config.Invoice, err = fromDBRuleConfigPayload[notification.InvoiceRuleConfig](eventType, data)
	default:
		return notification.RuleConfig{}, fmt.Errorf("invalid rule type: %s", eventType)
	}

	if err != nil {
		return notification.RuleConfig{}, err
	}

	return config, nil
}

// fromDBRuleConfigPayload tries the legacy envelope before the flat contract.
// Strict flat decoding prevents malformed envelopes from becoming empty invoice
// configs, since InvoiceRuleConfig has no fields.
func fromDBRuleConfigPayload[T any](eventType notification.EventType, data json.RawMessage) (*T, error) {
	// TODO(notification-v4): remove envelope decoding after all writers and
	// stored configs use the final representation.
	var legacy struct {
		Type notification.EventType `json:"type"`
		Data *T                     `json:"data"`
	}

	legacyErr := json.Unmarshal(data, &legacy)
	if legacyErr == nil {
		switch {
		case legacy.Type != eventType:
			legacyErr = fmt.Errorf("invalid rule config discriminator")
		case legacy.Data == nil:
			legacyErr = fmt.Errorf("invalid rule config envelope: missing data")
		default:
			return legacy.Data, nil
		}
	}

	var flat *T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	flatErr := decoder.Decode(&flat)
	if flatErr == nil {
		switch flat {
		case nil:
			flatErr = fmt.Errorf("invalid flat rule config: expected an object")
		default:
			if err := decoder.Decode(new(any)); err != io.EOF {
				flatErr = fmt.Errorf("invalid flat rule config: expected a single JSON object")
			}
		}
	}

	if flatErr != nil {
		return nil, fmt.Errorf("invalid rule config: envelope: %w; flat: %w", legacyErr, flatErr)
	}

	return flat, nil
}

func toDBRuleConfig(config notification.RuleConfig) (json.RawMessage, error) {
	// Preserve the validation previously performed by Ent's typed config field.
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rule config: %w", err)
	}

	var data any

	switch config.Type {
	case notification.EventTypeBalanceThreshold:
		data = config.BalanceThreshold
	case notification.EventTypeEntitlementReset:
		data = config.EntitlementReset
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		data = config.Invoice
	}

	// TODO(notification-v4): keep legacy envelope writes until all deployed and
	// supported rollback versions can read the final config contract.
	encoded, err := json.Marshal(struct {
		Type notification.EventType `json:"type"`
		Data any                    `json:"data"`
	}{
		Type: config.Type,
		Data: data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode rule config: %w", err)
	}

	return encoded, nil
}
