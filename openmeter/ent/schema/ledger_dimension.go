package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
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
		field.String("dimension_display_value").Immutable(),
	}
}

func (LedgerDimension) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "id").Unique(),
		index.Fields("namespace", "dimension_key", "dimension_value").Unique(),
	}
}

func (LedgerDimension) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sub_accounts", LedgerSubAccount.Type),
		edge.To("currency_sub_accounts", LedgerSubAccount.Type),
		edge.To("tax_code_sub_accounts", LedgerSubAccount.Type),
		edge.To("features_sub_accounts", LedgerSubAccount.Type),
		edge.To("credit_priority_sub_accounts", LedgerSubAccount.Type),
	}
}
