package adapter

import (
	"encoding/json"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func ChannelFromDBEntity(e db.NotificationChannel) (*notification.Channel, error) {
	config, err := fromDBChannelConfig(e.Type, e.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid notification channel config [namespace=%s channel.id=%s]: %w", e.Namespace, e.ID, err)
	}

	return &notification.Channel{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt.UTC(),
			UpdatedAt: e.UpdatedAt.UTC(),
			DeletedAt: func() *time.Time {
				if e.DeletedAt == nil {
					return nil
				}

				deletedAt := e.DeletedAt.UTC()

				return &deletedAt
			}(),
		},
		Type:     e.Type,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   config,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}, nil
}

func RuleFromDBEntity(e db.NotificationRule) (*notification.Rule, error) {
	eventType := fromDBEventType(e.Type)
	config, err := fromDBRuleConfig(eventType, e.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid notification rule config [namespace=%s rule.id=%s]: %w", e.Namespace, e.ID, err)
	}

	var channels []notification.Channel
	if len(e.Edges.Channels) > 0 {
		for _, channel := range e.Edges.Channels {
			if channel == nil {
				continue
			}

			mapped, err := ChannelFromDBEntity(*channel)
			if err != nil {
				return nil, fmt.Errorf("failed to map notification rule channel [rule.id=%s]: %w", e.ID, err)
			}

			channels = append(channels, *mapped)
		}
	}

	return &notification.Rule{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt.UTC(),
			UpdatedAt: e.UpdatedAt.UTC(),
			DeletedAt: func() *time.Time {
				if e.DeletedAt == nil {
					return nil
				}

				deletedAt := e.DeletedAt.UTC()

				return &deletedAt
			}(),
		},
		Type:     eventType,
		Name:     e.Name,
		Disabled: e.Disabled,
		Config:   config,
		Channels: channels,

		Annotations: e.Annotations,
		Metadata:    e.Metadata,
	}, nil
}

func EventFromDBEntity(e db.NotificationEvent) (*notification.Event, error) {
	payload, err := eventPayloadFromJSON(e.Payload)
	if err != nil {
		return nil, err
	}

	var statuses []notification.EventDeliveryStatus
	if len(e.Edges.DeliveryStatuses) > 0 {
		statuses = make([]notification.EventDeliveryStatus, 0, len(e.Edges.DeliveryStatuses))
		for _, status := range e.Edges.DeliveryStatuses {
			if status == nil {
				continue
			}

			mapped, err := EventDeliveryStatusFromDBEntity(*status)
			if err != nil {
				return nil, fmt.Errorf("failed to map notification event delivery status [event.id=%s]: %w", e.ID, err)
			}

			statuses = append(statuses, *mapped)
		}
	}

	ruleRow, err := e.Edges.RulesOrErr()
	if err != nil {
		return nil, err
	}

	rule, err := RuleFromDBEntity(*ruleRow)
	if err != nil {
		return nil, fmt.Errorf("failed to map notification event rule [event.id=%s]: %w", e.ID, err)
	}

	return &notification.Event{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		Type:           fromDBEventType(e.Type),
		CreatedAt:      e.CreatedAt.UTC(),
		Payload:        payload,
		Rule:           *rule,
		DeliveryStatus: statuses,
		Annotations:    e.Annotations,
	}, nil
}

func eventPayloadFromJSON(data []byte) (notification.EventPayload, error) {
	var meta notification.EventPayloadMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload meta: %w", err)
	}

	switch meta.Type {
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		if meta.Version != notification.EventPayloadVersionCurrent {
			return notification.EventPayload{}, fmt.Errorf("unsupported notification event payload version: %d", meta.Version)
		}
	}

	var payload notification.EventPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return notification.EventPayload{}, fmt.Errorf("failed to deserialize notification event payload: %w", err)
	}

	return payload, nil
}

func EventDeliveryStatusFromDBEntity(e db.NotificationEventDeliveryStatus) (*notification.EventDeliveryStatus, error) {
	var attempts []notification.EventDeliveryAttempt
	if len(e.Attempts) > 0 {
		if err := json.Unmarshal(e.Attempts, &attempts); err != nil {
			return nil, fmt.Errorf("invalid notification event delivery attempts [namespace=%s delivery_status.id=%s]: %w", e.Namespace, e.ID, err)
		}
	}

	state, _ := deliveryStateSpellings(e.State)

	return &notification.EventDeliveryStatus{
		NamespacedID: models.NamespacedID{
			Namespace: e.Namespace,
			ID:        e.ID,
		},
		ChannelID: e.ChannelID,
		EventID:   e.EventID,
		State:     state,
		Reason:    e.Reason,
		CreatedAt: e.CreatedAt.UTC(),
		UpdatedAt: e.UpdatedAt.UTC(),

		NextAttempt: func() *time.Time {
			if e.NextAttemptAt == nil {
				return nil
			}

			return lo.ToPtr(lo.FromPtr(e.NextAttemptAt).UTC())
		}(),
		Attempts: attempts,

		Annotations: e.Annotations,
	}, nil
}

// TODO(notification-v4): retire mixed-storage predicates once all writers and rows
// use versioned names. V1/V3 still need the public name mapping after cutover.
var eventTypeNames = [...]struct {
	legacy    notification.EventType
	versioned string
}{
	{notification.EventTypeBalanceThreshold, "v1.entitlement.balance_threshold.triggered"},
	{notification.EventTypeEntitlementReset, "v1.entitlement.usage_period.reset"},
	{notification.EventTypeInvoiceCreated, "v1.billing.invoice.created"},
	{notification.EventTypeInvoiceUpdated, "v1.billing.invoice.updated"},
}

func fromDBEventType(value string) notification.EventType {
	for _, names := range eventTypeNames {
		if value == names.versioned {
			return names.legacy
		}
	}

	return notification.EventType(value)
}

func toDBEventType(value notification.EventType) string {
	for _, names := range eventTypeNames {
		if value == names.legacy {
			return names.versioned
		}
	}

	return string(value)
}

func storedEventTypeNames(values ...string) []string {
	names := make([]string, 0, 2*len(values))
	for _, value := range values {
		names = append(names, value)

		for _, pair := range eventTypeNames {
			if value == string(pair.legacy) {
				names = append(names, pair.versioned)
				break
			}
		}
	}

	return names
}

func eventTypePredicate(field string, f *filter.FilterString) func(*sql.Selector) {
	switch {
	case f.Eq != nil:
		return sql.FieldIn(field, storedEventTypeNames(*f.Eq)...)
	case f.Ne != nil:
		return sql.FieldNotIn(field, storedEventTypeNames(*f.Ne)...)
	case f.In != nil:
		return sql.FieldIn(field, storedEventTypeNames((*f.In)...)...)
	case f.Nin != nil:
		return sql.FieldNotIn(field, storedEventTypeNames((*f.Nin)...)...)
	case f.Exists != nil:
		return f.Select(field)
	default:
		// Internal callers can also supply pattern, range, or compound filters.
		// Let PostgreSQL apply those to the same name that callers see on reads.
		return func(s *sql.Selector) {
			d := sql.Dialect(s.Dialect())
			names := d.Select().AppendSelectExprAs(legacyEventTypeExpr(s.C(field)), field).As("event_type_names")
			matches := d.Select().From(names)
			f.Select(names.C(field))(matches)

			s.Where(sql.Exists(matches))
		}
	}
}

func legacyEventTypeExpr(column string) sql.Querier {
	return sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("CASE ").Ident(column)

		for _, names := range eventTypeNames {
			b.WriteString(" WHEN ").Arg(names.versioned).WriteString(" THEN ").Arg(names.legacy)
		}

		b.WriteString(" ELSE ").Ident(column).WriteString(" END")
	})
}

func byLegacyEventType(field string, order sortx.Order) func(*sql.Selector) {
	return func(s *sql.Selector) {
		name := legacyEventTypeExpr(s.C(field))
		if order == sortx.OrderDesc {
			name = sql.DescExpr(name)
		}

		s.OrderExpr(name)
	}
}
