package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/convert"
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
		field.String("feature_id").Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("subject_key").Immutable(),
		field.Time("measure_usage_from").Immutable(),
		field.Time("usage_period_anchor"),
		field.Enum("usage_period_interval").Values(convert.Transform(credit.RecurrancePeriodValues, func(v credit.RecurrencePeriod) string {
			return string(v)
		})...).Immutable(),
		field.Time("usage_period_next_reset"),
	}
}

func (Entitlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id"),
		index.Fields("namespace", "subject_key"),
		index.Fields("namespace", "id", "subject_key"),
		index.Fields("namespace", "feature_id", "id"),
		index.Fields("namespace", "usage_period_next_reset"),
	}
}

func (Entitlement) Edges() []ent.Edge {
	return []ent.Edge{
		// link to usage_reset as that references entitlement
		edge.To("usage_reset", UsageReset.Type),
	}
}
