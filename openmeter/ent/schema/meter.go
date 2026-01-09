package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
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
		// TODO, FIXME:
		//  this should not be a 1:1 relationship, we should allow multiple table engines per meter
		//  thus we can gracefully upgrade to new versions or have hints regarding what table engine is good for
		//  what
		edge.To("table_engine", MeterTableEngine.Type).Unique(),
	}
}

type MeterTableEngine struct {
	ent.Schema
}

func (MeterTableEngine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (MeterTableEngine) Fields() []ent.Field {
	return []ent.Field{
		field.String("meter_id").NotEmpty().Immutable().SchemaType(map[string]string{
			dialect.Postgres: "char(26)",
		}),
		field.String("engine").NotEmpty().Default("events"),
		field.Enum("status").GoType(meter.MeterTableEngineState("")).Default(string(meter.MeterTableEngineStatePreparing)),
		field.String("state").SchemaType(map[string]string{
			dialect.Postgres: "jsonb",
		}),
	}
}

func (MeterTableEngine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("meter_id").Unique(),
	}
}

func (MeterTableEngine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("meter", Meter.Type).
			Ref("table_engine").
			Field("meter_id").
			Required().
			Unique().
			Immutable(),
	}
}
