package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SubscriptionCostBasisPin struct {
	ent.Schema
}

func (SubscriptionCostBasisPin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.IDMixin{},
		entutils.NamespaceMixin{},
		entutils.TimeMixin{},
	}
}

func (SubscriptionCostBasisPin) Fields() []ent.Field {
	return []ent.Field{
		field.String("subscription_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty().
			Immutable(),
		field.String("custom_currency_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty().
			Immutable(),
		field.String("invoice_currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			MinLen(3).
			MaxLen(3).
			Immutable(),
		field.String("cost_basis_id").
			SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
			NotEmpty().
			Immutable(),
	}
}

func (SubscriptionCostBasisPin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("subscription", Subscription.Type).
			Ref("cost_basis_pins").
			Field("subscription_id").
			Unique().
			Required().
			Immutable(),
		edge.From("custom_currency", CustomCurrency.Type).
			Ref("subscription_cost_basis_pins").
			Field("custom_currency_id").
			Unique().
			Required().
			Immutable(),
		edge.From("cost_basis", CurrencyCostBasis.Type).
			Ref("subscription_pins").
			Field("cost_basis_id").
			Unique().
			Required().
			Immutable(),
	}
}

func (SubscriptionCostBasisPin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "subscription_id", "custom_currency_id", "invoice_currency").Unique(),
		index.Fields("subscription_id"),
		index.Fields("custom_currency_id"),
		index.Fields("cost_basis_id"),
	}
}
