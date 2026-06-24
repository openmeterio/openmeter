package intentoverride

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type UsageBasedMixin = entutils.RecursiveMixin[usageBasedMixin]

type usageBasedMixin struct {
	mixin.Schema
}

func (usageBasedMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

func (usageBasedMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("override_feature_key").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			NotEmpty().
			Nillable(),
		field.String("override_price").
			GoType(&productcatalog.Price{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.Price]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.Discounts]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
	}
}
