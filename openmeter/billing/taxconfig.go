package billing

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TaxConfig is the billing-layer tax configuration. It extends productcatalog.TaxConfig with
// TaxCode, the resolved entity snapshot stamped at invoice snapshot time.
type TaxConfig struct {
	Behavior  *productcatalog.TaxBehavior    `json:"behavior,omitempty"`
	Stripe    *productcatalog.StripeTaxConfig `json:"stripe,omitempty"`
	TaxCodeID *string                         `json:"tax_code_id,omitempty"`
	// TaxCode is the resolved TaxCode entity, stamped at invoice snapshot time.
	TaxCode *taxcode.TaxCode `json:"tax_code,omitempty"`
}

// FromProductCatalog promotes a productcatalog.TaxConfig into a billing.TaxConfig.
// TaxCode is left nil; stamp it after resolving the entity.
func FromProductCatalog(c *productcatalog.TaxConfig) *TaxConfig {
	if c == nil {
		return nil
	}
	return &TaxConfig{
		Behavior:  c.Behavior,
		Stripe:    c.Stripe,
		TaxCodeID: c.TaxCodeID,
	}
}

// ToProductCatalog strips the billing-only TaxCode field and returns the intent-level config.
func (c *TaxConfig) ToProductCatalog() *productcatalog.TaxConfig {
	if c == nil {
		return nil
	}
	return &productcatalog.TaxConfig{
		Behavior:  c.Behavior,
		Stripe:    c.Stripe,
		TaxCodeID: c.TaxCodeID,
	}
}

func (c *TaxConfig) Equal(v *TaxConfig) bool {
	if c == nil && v == nil {
		return true
	}

	if c == nil || v == nil {
		return false
	}

	if (c.Behavior != nil && v.Behavior == nil) || (c.Behavior == nil && v.Behavior != nil) {
		return false
	}

	if c.Behavior != nil && *c.Behavior != *v.Behavior {
		return false
	}

	if (c.TaxCodeID != nil && v.TaxCodeID == nil) || (c.TaxCodeID == nil && v.TaxCodeID != nil) {
		return false
	}

	if c.TaxCodeID != nil && *c.TaxCodeID != *v.TaxCodeID {
		return false
	}

	if !c.TaxCode.Equal(v.TaxCode) {
		return false
	}

	return c.Stripe.Equal(v.Stripe)
}

func (c *TaxConfig) Validate() error {
	var errs []error

	if c.Behavior != nil {
		if err := c.Behavior.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.Stripe != nil {
		if err := c.Stripe.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid stripe config: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c TaxConfig) Clone() TaxConfig {
	out := TaxConfig{}

	if c.Behavior != nil {
		out.Behavior = lo.ToPtr(*c.Behavior)
	}

	if c.Stripe != nil {
		out.Stripe = lo.ToPtr(c.Stripe.Clone())
	}

	if c.TaxCodeID != nil {
		out.TaxCodeID = lo.ToPtr(*c.TaxCodeID)
	}

	if c.TaxCode != nil {
		tc := *c.TaxCode
		tc.AppMappings = append(taxcode.TaxCodeAppMappings(nil), c.TaxCode.AppMappings...)
		if c.TaxCode.Description != nil {
			tc.Description = lo.ToPtr(*c.TaxCode.Description)
		}
		out.TaxCode = &tc
	}

	return out
}

// MergeTaxConfigs merges two billing TaxConfigs with overrides taking precedence.
//
// Stripe and TaxCodeID are two encodings of the same intent-level tax-code identity, so they
// merge as a unit: a config that overrides only the Stripe code must not inherit the base's
// (different) TaxCodeID, which would leave the result pointing at two different tax entities.
//
// TaxCode (resolved entity) is intentionally excluded: merge operates on intent-level configs,
// not snapshotted invoice lines.
func MergeTaxConfigs(base, overrides *TaxConfig) *TaxConfig {
	if base != nil && overrides != nil {
		stripe, taxCodeID := base.Stripe, base.TaxCodeID
		if overrides.Stripe != nil || overrides.TaxCodeID != nil {
			stripe, taxCodeID = overrides.Stripe, overrides.TaxCodeID
		}

		return &TaxConfig{
			Behavior:  lo.CoalesceOrEmpty(overrides.Behavior, base.Behavior),
			Stripe:    stripe,
			TaxCodeID: taxCodeID,
		}
	}

	if overrides != nil {
		c := overrides.Clone()
		return &c
	}

	if base != nil {
		c := base.Clone()
		return &c
	}

	return nil
}
