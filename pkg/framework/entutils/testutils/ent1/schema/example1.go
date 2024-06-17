package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Example1 struct {
	ent.Schema
}

func (Example1) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.TimeMixin{},
	}
}

func (Example1) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("example_value_1"),
	}
}

func (Example1) Indexes() []ent.Index {
	return []ent.Index{}
}

func (Example1) Edges() []ent.Edge {
	return []ent.Edge{}
}
