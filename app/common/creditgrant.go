package common

import (
	"fmt"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var CreditGrant = wire.NewSet(
	NewCreditGrantService,
)

func NewCreditGrantService(
	billingRegistry BillingRegistry,
	customerService customer.Service,
) (creditgrant.Service, error) {
	if billingRegistry.Charges == nil {
		return nil, nil
	}

	svc, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: billingRegistry.Charges.CreditPurchaseService,
		ChargesService:        billingRegistry.Charges.Service,
		CustomerService:       customerService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create credit grant service: %w", err)
	}

	return svc, nil
}
