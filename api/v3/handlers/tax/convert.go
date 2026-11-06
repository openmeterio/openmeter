// Package tax holds the converter helpers for the v3 tax config shape, shared by the v3 handler
// packages whose rate cards embed it, so that no handler package needs to import another.
package tax

import (
	"errors"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ToAPITaxCodeConfig maps a lean tax config to the v3 shape, and nil or an empty config to nil.
func ToAPITaxCodeConfig(c *productcatalog.TaxCodeConfig) *api.TaxCodeConfig {
	if c == nil || (c.Behavior == nil && c.TaxCodeID == "") {
		return nil
	}

	result := &api.TaxCodeConfig{}

	if c.TaxCodeID != "" {
		result.Code = &api.TaxCodeReference{Id: c.TaxCodeID}
	}

	if c.Behavior != nil {
		result.Behavior = (*api.BillingTaxBehavior)(c.Behavior)
	}

	return result
}

// FromAPITaxCodeConfig maps the v3 shape back to a lean tax config. A nil input is absent; an
// explicit code reference without an ID, or a config without either field, is a validation error.
func FromAPITaxCodeConfig(tc *api.TaxCodeConfig) (*productcatalog.TaxCodeConfig, error) {
	if tc == nil {
		return nil, nil
	}

	if tc.Code != nil && tc.Code.Id == "" {
		return nil, models.NewGenericValidationError(errors.New("tax_config.code.id must be set when tax_config.code is present"))
	}

	if tc.Code == nil && tc.Behavior == nil {
		return nil, models.NewGenericValidationError(errors.New("tax_config.code.id or tax_config.behavior must be set"))
	}

	result := &productcatalog.TaxCodeConfig{}

	if tc.Code != nil {
		result.TaxCodeID = tc.Code.Id
	}

	if tc.Behavior != nil {
		result.Behavior = (*productcatalog.TaxBehavior)(tc.Behavior)
	}

	return result, nil
}
