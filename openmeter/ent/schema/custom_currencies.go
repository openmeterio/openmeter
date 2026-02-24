package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type CustomCurrency struct {
	ent.Schema
}

func (CustomCurrency) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (CustomCurrency) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			NotEmpty().
			MinLen(3).
			MaxLen(24).
			Immutable(),
		field.String("name").
			NotEmpty(),
		field.String("symbol").
			NotEmpty(),
	}
}

func (CustomCurrency) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cost_basis_history", CurrencyCostBasis.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (CustomCurrency) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "code").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}

type CurrencyCostBasis struct {
	ent.Schema
}

func (CurrencyCostBasis) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.NamespaceMixin{},
		entutils.IDMixin{},
		entutils.TimeMixin{},
	}
}

func (CurrencyCostBasis) Fields() []ent.Field {
	return []ent.Field{
		field.String("custom_currency_id").
			Immutable(),
		field.String("fiat_code").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable(),
		field.Other("rate", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Immutable(),
		field.Time("effective_from").
			Immutable(),
	}
}

func (CurrencyCostBasis) Edges() []ent.Edge {
	return []ent.Edge{
		// Many cost basis entries belong to one currency
		edge.From("currency", CustomCurrency.Type).
			Ref("cost_basis_history").
			Field("custom_currency_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (CurrencyCostBasis) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "custom_currency_id", "fiat_code", "effective_from").
			Annotations(
				entsql.IndexWhere("deleted_at IS NULL"),
			).
			Unique(),
	}
}
