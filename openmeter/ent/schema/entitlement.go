package schema

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Entitlement struct {
	ent.Schema
}

func (Entitlement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataMixin{},
		entutils.TimeMixin{},
	}
}

func (Entitlement) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("entitlement_type").Values(entitlement.EntitlementType("").StrValues()...).Immutable(),
		field.String("feature_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Time("active_from").Optional().Nillable().Immutable(),
		field.Time("active_to").Optional().Nillable(),
		field.String("feature_key").NotEmpty().Validate(func(fK string) error {
			if _, err := ulid.Parse(fK); err == nil {
				return fmt.Errorf("selected feature key cannot be a valid ULID")
			}
			return nil
		}).Immutable(),
		field.String("customer_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.Time("measure_usage_from").Optional().Nillable().Immutable(),
		field.Float("issue_after_reset").Optional().Nillable().Immutable(),
		field.Uint8("issue_after_reset_priority").Optional().Nillable().Immutable(),
		field.Bool("is_soft_limit").Optional().Nillable().Immutable(),
		field.Bool("preserve_overage_at_reset").Optional().Nillable().Immutable(),
		field.String("config").
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("usage_period_interval").GoType(datetime.ISODurationString("")).Optional().Nillable().Immutable(),
		field.Time("usage_period_anchor").Optional().Nillable().Comment("Historically this field had been overwritten with each anchor reset, now we keep the original anchor time and the value is populated from the last reset which is queried dynamically"),
		// TODO: get rid of current_usage_period in the db and make it calculated
		field.Time("current_usage_period_start").Optional().Nillable(),
		field.Time("current_usage_period_end").Optional().Nillable(),
		field.String("annotations").
			GoType(models.Annotations{}).
			ValueScanner(entutils.JSONStringValueScanner[models.Annotations]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
	}
}

func (Entitlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "customer_id"),
		index.Fields("namespace", "id", "customer_id"),
		index.Fields("namespace", "feature_id", "id"),
		index.Fields("namespace", "current_usage_period_end"),
		// Index for collecting entitlements with due resets
		index.Fields("current_usage_period_end", "deleted_at"),
		index.Fields("created_at", "id").Unique(),
	}
}

func (Entitlement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("usage_reset", UsageReset.Type).Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("grant", Grant.Type).Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("balance_snapshot", BalanceSnapshot.Type).Annotations(entsql.Annotation{
			OnDelete: entsql.Cascade,
		}),
		edge.To("subscription_item", SubscriptionItem.Type),
		edge.From("feature", Feature.Type).
			Ref("entitlement").
			Field("feature_id").
			Required().
			Unique().
			Immutable(),
		edge.From("customer", Customer.Type).
			Ref("entitlements").
			Field("customer_id").
			Required().
			Unique().
			Immutable(),
	}
}
