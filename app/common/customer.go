package common

import (
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

func NewCustomerService(logger *slog.Logger, db *entdb.Client) (customer.Service, error) {
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: db,
		Logger: logger.WithGroup("customer.postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	return customerservice.New(customerservice.Config{
		Adapter: customerAdapter,
	})
}
