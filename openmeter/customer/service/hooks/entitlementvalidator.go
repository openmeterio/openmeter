package hooks

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	EntitlementValidatorHook     = models.ServiceHook[customer.Customer]
	NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer]
)

var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil)

type entitlementValidatorHook struct {
	NoopEntitlementValidatorHook

	entitlementService entitlement.Connector
}

type EntitlementValidatorHookConfig struct {
	EntitlementService entitlement.Connector
}

func (e EntitlementValidatorHookConfig) Validate() error {
	if e.EntitlementService == nil {
		return fmt.Errorf("entitlement service is required")
	}

	return nil
}

func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid entitlement validator hook config: %w", err)
	}

	return &entitlementValidatorHook{
		entitlementService: config.EntitlementService,
	}, nil
}

func (e *entitlementValidatorHook) PreDelete(ctx context.Context, customer *customer.Customer) error {
	entitlements, err := e.entitlementService.GetEntitlementsOfCustomer(ctx, customer.Namespace, customer.ID, clock.Now())
	if err != nil {
		return fmt.Errorf("failed to get entitlements of customer: %w", err)
	}

	if len(entitlements) > 0 {
		return models.NewGenericValidationError(fmt.Errorf("customer has entitlements, please remove them before deleting the customer"))
	}

	return nil
}
