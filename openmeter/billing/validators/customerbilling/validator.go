package customerbilling

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerapp "github.com/openmeterio/openmeter/openmeter/customer/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ValidateCustomerInvoicingApp(ctx context.Context, billingService billing.Service, customerID customer.CustomerID, capabilities []app.CapabilityType) error {
	if billingService == nil {
		return fmt.Errorf("billing service is required")
	}

	customerProfile, err := billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customerID,
		Expand: billing.CustomerOverrideExpand{
			Apps:     true,
			Customer: true,
		},
	})
	if err != nil {
		return err
	}

	appBase := customerProfile.MergedProfile.Apps.Invoicing
	if appBase == nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer with id %s has no invoicing app configured in namespace %s", customerID.ID, customerID.Namespace),
		)
	}

	customerApp, err := customerapp.AsCustomerApp(appBase)
	if err != nil {
		return err
	}

	return customerApp.ValidateCustomer(ctx, customerProfile.Customer, capabilities)
}
