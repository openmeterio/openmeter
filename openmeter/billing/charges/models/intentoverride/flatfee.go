package intentoverride

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type FlatFeeMixin = entutils.RecursiveMixin[flatFeeMixin]

type flatFeeMixin struct {
	mixin.Schema
}

func (flatFeeMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

func (flatFeeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("override_feature_key").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Enum("override_payment_term").
			GoType(productcatalog.PaymentTermType("")).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_pro_rating").
			GoType(&productcatalog.ProRatingConfig{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.ProRatingConfig]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Other("override_amount_before_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_percentage_discounts").
			GoType(&PercentageDiscountsOverride{}).
			ValueScanner(entutils.JSONStringValueScanner[*PercentageDiscountsOverride]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
	}
}

type PercentageDiscountsOverride struct {
	Value *productcatalog.PercentageDiscount `json:"value,omitempty"`
}

func (p *PercentageDiscountsOverride) Validate() error {
	if p == nil || p.Value == nil {
		return nil
	}

	return p.Value.Validate()
}
