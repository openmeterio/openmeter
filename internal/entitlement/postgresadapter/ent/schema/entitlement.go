package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Entitlement struct {
	ent.Schema
}

func (Entitlement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.MetadataAnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (Entitlement) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("entitlement_type").Values(entitlement.EntitlementType("").StrValues()...).Immutable(),
		field.String("feature_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("feature_key").Immutable(),
		field.String("subject_key").Immutable(),
		field.Time("measure_usage_from").Optional().Nillable().Immutable(),
		field.Float("issue_after_reset").Optional().Nillable().Immutable(),
		field.Bool("is_soft_limit").Optional().Nillable().Immutable(),
		field.JSON("config", map[string]interface{}{}).Optional(),
		field.Enum("usage_period_interval").Values(entitlement.UsagePeriodInterval("").StrValues()...).Optional().Nillable().Immutable(),
		field.Time("usage_period_anchor").Optional().Nillable().Immutable(),
	}
}

func (Entitlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subject_key"),
		index.Fields("namespace", "id", "subject_key"),
		index.Fields("namespace", "feature_id", "id"),
	}
}

func (Entitlement) Edges() []ent.Edge {
	return []ent.Edge{
		// link to usage_reset as that references entitlement
		edge.To("usage_reset", UsageReset.Type),
	}
}
