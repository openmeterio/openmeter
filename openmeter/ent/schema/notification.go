package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

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
		field.Enum("type").
			GoType(notification.ChannelType("")).
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.Bool("disabled").
			Default(false).
			Optional(),
		field.String("config").
			GoType(notification.ChannelConfig{}).
			ValueScanner(ChannelConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
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

type channelConfigSerde[T any] struct {
	notification.ChannelConfigMeta

	Data T `json:"data"`
}

var ChannelConfigValueScanner = field.ValueScannerFunc[notification.ChannelConfig, *sql.NullString]{
	V: func(config notification.ChannelConfig) (driver.Value, error) {
		switch config.Type {
		case notification.ChannelTypeWebhook:
			serde := channelConfigSerde[notification.WebHookChannelConfig]{
				ChannelConfigMeta: notification.ChannelConfigMeta{
					Type: config.Type,
				},
				Data: config.WebHook,
			}
			return json.Marshal(serde)
		default:
			return nil, fmt.Errorf("unknown channel type: %s", config.Type)
		}
	},
	S: func(ns *sql.NullString) (notification.ChannelConfig, error) {
		var channelConfig notification.ChannelConfig
		if ns == nil || !ns.Valid {
			return channelConfig, nil
		}

		data := []byte(ns.String)

		var meta notification.ChannelConfigMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return channelConfig, err
		}

		switch meta.Type {
		case notification.ChannelTypeWebhook:
			serde := channelConfigSerde[notification.WebHookChannelConfig]{
				ChannelConfigMeta: notification.ChannelConfigMeta{
					Type: meta.Type,
				},
				Data: notification.WebHookChannelConfig{},
			}

			if err := json.Unmarshal(data, &serde); err != nil {
				return channelConfig, err
			}

			channelConfig = notification.ChannelConfig{
				ChannelConfigMeta: serde.ChannelConfigMeta,
				WebHook:           serde.Data,
			}

		default:
			return channelConfig, fmt.Errorf("unknown channel type: %s", meta.Type)
		}

		return channelConfig, nil
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
