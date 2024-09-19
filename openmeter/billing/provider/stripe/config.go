package stripe

import "github.com/openmeterio/openmeter/openmeter/billing/provider/models"

type Config struct {
	// Supplier specifies the minimum required configuration for our billing stack, unfortunately with
	// Stripe we cannot easily fetch the registration details, so we need to store it here.
	Supplier models.SupplierConfig `json:"supplier"`
}

func (c *Config) Validate() error {
	return nil
}
