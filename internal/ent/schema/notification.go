package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/internal/notification"
	notificationpostgres "github.com/openmeterio/openmeter/internal/notification/repository/postgres"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type NotificationChannel struct {
	ent.Schema
}

func (NotificationChannel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
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
			ValueScanner(notificationpostgres.ChannelConfigValueScanner).
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
		index.Fields("namespace", "id", "type"),
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
	}
}

func (NotificationRule) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			GoType(notification.RuleType("")).
			Immutable().
			Comment("The event type the rule associated with"),
		field.String("name").
			NotEmpty().
			Comment("The name of the rule"),
		field.Bool("disabled").Default(false).Optional().
			Comment("Whether the rule is disabled or not"),
		field.String("config").
			GoType(notification.RuleConfig{}).
			ValueScanner(notificationpostgres.RuleConfigValueScanner).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
	}
}

func (NotificationRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channels", NotificationChannel.Type).
			Ref("rules"),
	}
}

type NotificationEvent struct {
	ent.Schema
}

func (NotificationEvent) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
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
		// FIXME: add custom value scanner
		field.String("rule").
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}),
		// FIXME: add custom value scanner
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
	}
}

type NotificationEventDeliveryStatus struct {
	ent.Schema
}

func (NotificationEventDeliveryStatus) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
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
		field.Enum("type").
			GoType(notification.EventType("")).
			Immutable().
			Comment("The event type the rule associated with"),
		field.Enum("State").
			GoType(notification.EventDeliveryStatusState("")).
			Comment("The event type the rule associated with"),
	}
}

func (NotificationEventDeliveryStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("events", NotificationEvent.Type),
	}
}
