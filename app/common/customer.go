package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entitlementvalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/registry"
)

var Customer = wire.NewSet(
	NewCustomerService,
)

func NewCustomerService(logger *slog.Logger, db *entdb.Client, entRegistry *registry.Entitlement) (customer.Service, error) {
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: db,
		Logger: logger.WithGroup("customer.postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	service, err := customerservice.New(customerservice.Config{
		Adapter:              customerAdapter,
		EntitlementConnector: entRegistry.Entitlement,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer service: %w", err)
	}

	// Create and register the entitlement validator
	validator, err := entitlementvalidator.NewValidator(service, entRegistry.EntitlementRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement validator: %w", err)
	}

	service.RegisterRequestValidator(validator)

	return service, nil
}
