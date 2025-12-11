package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Meter holds the schema definition for the Meter entity.
type Meter struct {
	ent.Schema
}

// Mixin of the Meter.
func (Meter) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.UniqueResourceMixin{},
		entutils.AnnotationsMixin{},
	}
}

// Fields of the Meter.
func (Meter) Fields() []ent.Field {
	return []ent.Field{
		field.String("event_type").NotEmpty().Immutable(),
		// Optional for count
		field.String("value_property").Nillable().Optional(),
		field.JSON("group_by", map[string]string{}).Optional(),
		field.Enum("aggregation").GoType(meter.MeterAggregation("")).Immutable(),
		// If set, only events since this time will be included.
		field.Time("event_from").Optional().Nillable(),
	}
}

// Indexes of the Meter.
func (Meter) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "key").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("namespace", "event_type"),
	}
}

// Edges of the Meter.
func (Meter) Edges() []ent.Edge {
	return []ent.Edge{
		// FIXME: enable foreign key constraints
		// Ent doesn't support foreign key constraints on non ID fields (key)
		// https://github.com/ent/ent/issues/2549
		// edge.To("feature", Feature.Type),
	}
}
