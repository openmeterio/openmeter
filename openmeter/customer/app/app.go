package customerapp

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type App interface {
	// ValidateCustomer validates if the app can run for the given customer
	ValidateCustomer(ctx context.Context, customer *customer.Customer, capabilities []app.CapabilityType) error
}

// AsCustomerApp returns the app from the app entity
func AsCustomerApp(customerAppCandidate app.App) (App, error) {
	customerApp, ok := customerAppCandidate.(App)
	if !ok {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("is not a customer app [id=%s, type=%s]", customerAppCandidate.GetID(), customerAppCandidate.GetType()),
		)
	}

	return customerApp, nil
}
