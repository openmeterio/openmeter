package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type TimeMixin struct {
	mixin.Schema
}

func (TimeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

type CustomCurrency struct {
	ent.Schema
}

func (CustomCurrency) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		TimeMixin{},
	}
}

func (CustomCurrency) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").NotEmpty().MinLen(3).MaxLen(3).Unique().Immutable(),
		field.String("name").NotEmpty().MaxLen(100).Immutable(),
		field.String("symbol").NotEmpty().MaxLen(10).Immutable(),
		field.Int8("smallest_denomination").NonNegative().Default(2).Immutable(),
	}
}

func (CustomCurrency) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cost_basis_history", CurrencyCostBasis.Type),
	}
}

type CurrencyCostBasis struct {
	ent.Schema
}

func (CurrencyCostBasis) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		TimeMixin{},
	}
}

func (CurrencyCostBasis) Fields() []ent.Field {
	return []ent.Field{
		field.String("fiat_code").NotEmpty().MinLen(3).MaxLen(3).Immutable(),
		field.Other("rate", alpacadecimal.Decimal{}).SchemaType(map[string]string{
			dialect.Postgres: "numeric",
		}).Immutable(),
		field.Time("effective_from").Default(time.Now).Immutable(),
	}
}

func (CurrencyCostBasis) Edges() []ent.Edge {
	return []ent.Edge{
		// Many cost basis entries belong to one currency
		edge.From("currency", CustomCurrency.Type).
			Ref("cost_basis_history").
			Unique().
			Required(),
	}
}

func (CurrencyCostBasis) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("currency").Fields("fiat_code", "effective_from").Unique(),
	}
}
