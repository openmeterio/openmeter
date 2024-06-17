package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example2 struct {
	ent.Schema
}

func (Example2) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.TimeMixin{},
	}
}

func (Example2) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("example_value_2"),
	}
}

func (Example2) Indexes() []ent.Index {
	return []ent.Index{}
}

func (Example2) Edges() []ent.Edge {
	return []ent.Edge{}
}
