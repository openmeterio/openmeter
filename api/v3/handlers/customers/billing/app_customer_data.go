package customersbilling

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// applyAppCustomerData applies the app data section of a billing update to the
// profile's payment app. An explicit null deletes the app's customer data and
// an omitted custom invoicing field leaves it unchanged; everything else is
// mapped faithfully (zero values where absent) and upserted so the app service
// reports validation issues. The returned value echoes the applied app data.
func (h *handler) applyAppCustomerData(ctx context.Context, application app.App, customerID customer.CustomerID, data api.BillingAppCustomerData) (*api.BillingAppCustomerData, error) {
	resp := &api.BillingAppCustomerData{}
	var appData app.CustomerData

	switch application.GetType() {
	case app.AppTypeStripe:
		if data.Stripe.IsNull() {
			return resp, h.deleteAppCustomerData(ctx, application, customerID)
		}

		if !data.Stripe.IsSpecified() {
			return resp, nil
		}

		stripeData, err := data.Stripe.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to read stripe data: %w", err)
		}

		resp.Stripe = data.Stripe
		appData = appstripe.CustomerData{
			StripeCustomerID:             lo.FromPtr(stripeData.CustomerId),
			StripeDefaultPaymentMethodID: stripeData.DefaultPaymentMethodId,
		}
	case app.AppTypeCustomInvoicing:
		if data.ExternalInvoicing.IsNull() {
			return resp, h.deleteAppCustomerData(ctx, application, customerID)
		}

		if !data.ExternalInvoicing.IsSpecified() {
			return resp, nil
		}

		externalInvoicing, err := data.ExternalInvoicing.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to read external invoicing data: %w", err)
		}

		resp.ExternalInvoicing = data.ExternalInvoicing
		appData = appcustominvoicing.CustomerData{}
		if externalInvoicing.Labels != nil {
			appData = appcustominvoicing.CustomerData{
				Metadata: models.Metadata(*externalInvoicing.Labels),
			}
		}
	case app.AppTypeSandbox:
		appData = appsandbox.CustomerData{}
	default:
		return nil, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
	}

	if err := application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
		CustomerID: customerID,
		Data:       appData,
	}); err != nil {
		return nil, fmt.Errorf("failed to update customer data: %w", err)
	}

	return resp, nil
}

func (h *handler) deleteAppCustomerData(ctx context.Context, application app.App, customerID customer.CustomerID) error {
	if err := application.DeleteCustomerData(ctx, app.DeleteAppInstanceCustomerDataInput{
		CustomerID: customerID,
	}); err != nil {
		return fmt.Errorf("failed to delete customer data: %w", err)
	}

	return nil
}
