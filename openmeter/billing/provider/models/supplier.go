package models

import (
	"fmt"

	"github.com/invopop/gobl/l10n"
)

type SupplierConfig struct {
	// Name is the name of the supplier
	Name string `json:"name"`
	// Country is the country of the supplier
	TaxCountry l10n.TaxCountryCode `json:"taxCountry"`
}

func (c *SupplierConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("supplier name is required")
	}

	if c.TaxCountry == "" {
		return fmt.Errorf("supplier tax country is required")
	}

	return nil
}
