package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
	return []ent.Index{}
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
