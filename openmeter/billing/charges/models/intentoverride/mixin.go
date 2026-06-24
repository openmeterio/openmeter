package intentoverride

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

const deprecatedInlineOverrideField = "intent overrides are stored in dedicated charge override tables"

type BaseMixin = entutils.RecursiveMixin[overrideBaseMixin]

type overrideBaseMixin struct {
	mixin.Schema
}

func (overrideBaseMixin) Mixin() []ent.Mixin {
	return nil
}

type TaxBehaviorOverride string

const (
	TaxBehaviorOverrideClear TaxBehaviorOverride = ""
)

func (t TaxBehaviorOverride) Validate() error {
	if t == TaxBehaviorOverrideClear {
		return nil
	}

	taxBehavior := productcatalog.TaxBehavior(t)
	if err := taxBehavior.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid tax behavior override: %s", t))
	}

	return nil
}

func (overrideBaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("override_present").
			Default(false).
			Deprecated(deprecatedInlineOverrideField),
		field.String("override_name").
			Optional().
			NotEmpty().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.String("override_description").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.String("override_metadata").
			GoType(&models.Metadata{}).
			ValueScanner(entutils.JSONStringValueScanner[*models.Metadata]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.String("override_tax_behavior").
			GoType(TaxBehaviorOverride("")).
			Validate(func(taxBehavior string) error {
				return TaxBehaviorOverride(taxBehavior).Validate()
			}).
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.String("override_tax_code_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_intent_deleted_at").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_service_period_from").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_service_period_to").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_full_service_period_from").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_full_service_period_to").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_billing_period_from").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
		field.Time("override_billing_period_to").
			Optional().
			Nillable().
			Deprecated(deprecatedInlineOverrideField),
	}
}
