package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type LedgerDimension struct {
	ent.Schema
}

func (LedgerDimension) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.AnnotationsMixin{},
		entutils.TimeMixin{},
	}
}

func (LedgerDimension) Fields() []ent.Field {
	return []ent.Field{
		field.String("dimension_key").Immutable(),
		field.String("dimension_value").Immutable(),
	}
}

func (LedgerDimension) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "dimension_key", "dimension_value"),
	}
}
