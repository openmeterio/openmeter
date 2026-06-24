package intentoverride

import (
	"fmt"
	"slices"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BaseMixin = entutils.RecursiveMixin[overrideBaseMixin]

type overrideBaseMixin struct {
	mixin.Schema
}

const legacyInlineOverrideColumnDeprecation = "legacy inline override column; kept for compatibility only"

func (overrideBaseMixin) Mixin() []ent.Mixin {
	return nil
}

type Kind string

const (
	KindEdit   Kind = "edit"
	KindDelete Kind = "delete"
)

func (k Kind) Values() []string {
	return []string{
		string(KindEdit),
		string(KindDelete),
	}
}

func (k Kind) Validate() error {
	if !slices.Contains(k.Values(), string(k)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid intent override kind: %s", k))
	}

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
		field.Enum("override_kind").
			GoType(Kind("")).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_name").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			NotEmpty().
			Nillable(),
		field.String("override_description").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_metadata").
			GoType(&models.Metadata{}).
			ValueScanner(entutils.JSONStringValueScanner[*models.Metadata]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_tax_behavior").
			GoType(TaxBehaviorOverride("")).
			Validate(func(taxBehavior string) error {
				return TaxBehaviorOverride(taxBehavior).Validate()
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.String("override_tax_code_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_service_period_from").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_service_period_to").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_full_service_period_from").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_full_service_period_to").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_billing_period_from").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
		field.Time("override_billing_period_to").
			Deprecated(legacyInlineOverrideColumnDeprecation).
			Optional().
			Nillable(),
	}
}
