package openmetersandbox

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/provider/models"
)

type Config struct {
	Supplier models.SupplierConfig `json:"supplier"`
}

func (c *Config) Validate() error {
	if err := c.Supplier.Validate(); err != nil {
		return fmt.Errorf("failed to validate supplier configuration: %w", err)
	}

	return nil
}
