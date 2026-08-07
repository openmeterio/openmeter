package detailedline

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Mixin = entutils.RecursiveMixin[mixinBase]

type mixinBase struct {
	mixin.Schema
}

func (mixinBase) Mixin() []ent.Mixin {
	return []ent.Mixin{
		stddetailedline.Mixin{},
	}
}

func (mixinBase) Fields() []ent.Field {
	return []ent.Field{
		field.JSON("amount_discounts", AmountDiscounts{}).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional(),
	}
}
