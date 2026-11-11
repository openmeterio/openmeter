package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type NotificationChannel struct {
	ent.Schema
}

func (NotificationChannel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
		entutils.MetadataMixin{},
	}
}

func (NotificationChannel) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").
			GoType(notification.ChannelType("")).
			ValueScanner(ChannelTypeValueScanner).
			NotEmpty().
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.Bool("disabled").
			Default(false).
			Optional(),
		field.String("config").
			GoType(json.RawMessage{}).
			ValueScanner(entutils.JSONStringValueScanner[json.RawMessage]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		field.String("event_type_filter").
			GoType([]string{}).
			ValueScanner(entutils.JSONStringValueScanner[[]string]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Comment("Unique resolved event types for V4 routing; inactive during migration Phase 1."),
		field.String("provider_data").
			GoType(map[string]string{}).
			ValueScanner(entutils.JSONStringValueScanner[map[string]string]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Comment("Provider-owned operational data for V4."),
		// TODO(notification-v4): activate generation tracking and reconciliation
		// only after all writers support them; legacy updates preserve these fields.
		field.Int("generation").
			Min(1).
			Max(math.MaxInt32).
			Default(1).
			SchemaType(map[string]string{
				dialect.Postgres: "integer",
			}).
			Comment("Desired-state generation for V4; inactive during migration Phase 1."),
		field.String("reconcile_status").
			GoType(notification.ChannelReconcileStatus("")).
			Validate(func(status string) error {
				return notification.ChannelReconcileStatus(status).Validate()
			}).
			Default(string(notification.ChannelReconcileStatusUnknown)).
			SchemaType(map[string]string{
				dialect.Postgres: "text",
			}).
			Comment("Channel reconciliation status for V4; unknown until provider adoption."),
		field.Time("reconcile_next_at").
			Optional().
			Nillable().
			Comment("Scheduled Channel reconciliation time for V4; inactive during migration Phase 1."),
	}
}

func (NotificationChannel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("rules", NotificationRule.Type),
	}
}

func (NotificationChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "type"),
	}
}

type NotificationRule struct {
	ent.Schema
}

func (NotificationRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
		entutils.AnnotationsMixin{},
		entutils.MetadataMixin{},
	}
}

func (NotificationRule) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			GoType(notification.EventType("")).
			Immutable().
			Comment("The event type the rule associated with"),
		field.String("name").
			NotEmpty().
			Comment("The name of the rule"),
		field.Bool("disabled").
			Default(false).
			Optional().
			Comment("Whether the rule is disabled or not"),
		field.String("config").
			GoType(notification.RuleConfig{}).
			ValueScanner(RuleConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (NotificationRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channels", NotificationChannel.Type).
			Ref("rules"),
		edge.To("events", NotificationEvent.Type),
	}
}

func (NotificationRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "type"),
	}
}

type NotificationEvent struct {
	ent.Schema
}

func (NotificationEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (NotificationEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Enum("type").
			GoType(notification.EventType("")).
			Immutable().
			Comment("The event type the rule associated with"),
		field.String("rule_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		// TODO(chrisgacsal): add custom value scanner
		field.String("payload").
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (NotificationEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("delivery_statuses", NotificationEventDeliveryStatus.Type).
			Ref("events"),
		edge.From("rules", NotificationRule.Type).
			Ref("events").
			Field("rule_id").
			Required().
			Unique().
			Immutable(),
	}
}

func (NotificationEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "type"),
	}
}

type NotificationEventDeliveryStatus struct {
	ent.Schema
}

func (NotificationEventDeliveryStatus) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
	}
}

func (NotificationEventDeliveryStatus) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Default(clock.Now).
			Immutable(),
		field.Time("updated_at").
			Default(clock.Now).
			UpdateDefault(clock.Now),
		field.String("event_id").
			NotEmpty().
			Immutable(),
		field.String("channel_id").
			NotEmpty().
			Immutable(),
		field.Enum("state").
			GoType(notification.EventDeliveryStatusState("")).
			Default(string(notification.EventDeliveryStatusStatePending)),
		field.String("reason").
			Optional(),
		field.Time("next_attempt_at").
			Optional().
			Nillable(),
		field.JSON("attempts", []notification.EventDeliveryAttempt{}).
			Optional(),
	}
}

func (NotificationEventDeliveryStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("events", NotificationEvent.Type),
	}
}

func (NotificationEventDeliveryStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "event_id", "channel_id"),
		index.Fields("namespace", "state"),
		index.Fields("namespace", "state", "next_attempt_at"),
	}
}

// TODO(notification-v4): keep legacy writes until every deployed and rollback
// version supports the final format; remove legacy reads after data conversion.
var ChannelTypeValueScanner = field.ValueScannerFunc[notification.ChannelType, *sql.NullString]{
	V: func(channelType notification.ChannelType) (driver.Value, error) {
		if channelType == notification.ChannelTypeWebhook {
			return "WEBHOOK", nil
		}

		return string(channelType), nil
	},
	S: func(value *sql.NullString) (notification.ChannelType, error) {
		if value == nil || !value.Valid {
			return "", nil
		}

		if value.String == "WEBHOOK" {
			return notification.ChannelTypeWebhook, nil
		}

		return notification.ChannelType(value.String), nil
	},
}

type ruleConfigSerde[T any] struct {
	notification.RuleConfigMeta

	Data *T `json:"data"`
}

var RuleConfigValueScanner = field.ValueScannerFunc[notification.RuleConfig, *sql.NullString]{
	V: func(config notification.RuleConfig) (driver.Value, error) {
		switch config.Type {
		case notification.EventTypeBalanceThreshold:
			serde := ruleConfigSerde[notification.BalanceThresholdRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: config.Type,
				},
				Data: config.BalanceThreshold,
			}

			return json.Marshal(serde)
		case notification.EventTypeEntitlementReset:
			serde := ruleConfigSerde[notification.EntitlementResetRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: config.Type,
				},
				Data: config.EntitlementReset,
			}

			return json.Marshal(serde)
		case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
			serde := ruleConfigSerde[notification.InvoiceRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: config.Type,
				},
				Data: config.Invoice,
			}

			return json.Marshal(serde)
		default:
			return nil, fmt.Errorf("unknown rule config type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (notification.RuleConfig, error) {
		var ruleConfig notification.RuleConfig
		if ns == nil || !ns.Valid {
			return ruleConfig, nil
		}

		data := []byte(ns.String)

		var meta notification.RuleConfigMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return ruleConfig, err
		}

		switch meta.Type {
		case notification.EventTypeBalanceThreshold:
			serde := ruleConfigSerde[notification.BalanceThresholdRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: meta.Type,
				},
				Data: &notification.BalanceThresholdRuleConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return ruleConfig, err
			}

			ruleConfig = notification.RuleConfig{
				RuleConfigMeta:   serde.RuleConfigMeta,
				BalanceThreshold: serde.Data,
			}

		case notification.EventTypeEntitlementReset:
			serde := ruleConfigSerde[notification.EntitlementResetRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: meta.Type,
				},
				Data: &notification.EntitlementResetRuleConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return ruleConfig, err
			}

			ruleConfig = notification.RuleConfig{
				RuleConfigMeta:   serde.RuleConfigMeta,
				EntitlementReset: serde.Data,
			}
		case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
			serde := ruleConfigSerde[notification.InvoiceRuleConfig]{
				RuleConfigMeta: notification.RuleConfigMeta{
					Type: meta.Type,
				},
				Data: &notification.InvoiceRuleConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return ruleConfig, err
			}

			ruleConfig = notification.RuleConfig{
				RuleConfigMeta: serde.RuleConfigMeta,
				Invoice:        serde.Data,
			}

		default:
			return ruleConfig, fmt.Errorf("unknown rule type: %s", meta.Type)
		}

		return ruleConfig, nil
	},
}

var AnnotationsValueScanner = field.ValueScannerFunc[models.Annotations, *sql.NullString]{
	V: func(annotations models.Annotations) (driver.Value, error) {
		b, err := json.Marshal(annotations)
		if err != nil {
			return nil, err
		}

		return string(b), nil
	},
	S: func(ns *sql.NullString) (models.Annotations, error) {
		var annotations models.Annotations
		if ns == nil || !ns.Valid {
			return annotations, nil
		}

		if err := json.Unmarshal([]byte(ns.String), &annotations); err != nil {
			return nil, err
		}

		return annotations, nil
	},
}
