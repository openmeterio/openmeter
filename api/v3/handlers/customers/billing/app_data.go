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

// buildAppData returns the customer's current data for the resolved payment
// app, so update responses match a subsequent get. Stripe data is omitted when
// none exists; custom invoicing cannot distinguish missing data from empty
// labels, so its field is always present.
func buildAppData(ctx context.Context, application app.App, customerID customer.CustomerID) (*api.BillingAppCustomerData, error) {
	appData := api.BillingAppCustomerData{}

	switch application.GetType() {
	case app.AppTypeStripe:
		data, err := application.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
			CustomerID: customerID,
		})
		if err != nil {
			if app.IsAppCustomerPreConditionError(err) {
				return &appData, nil
			}

			return nil, err
		}

		if stripeData, ok := data.(appstripe.CustomerData); ok {
			appData.Stripe = &api.BillingAppCustomerDataStripe{
				CustomerId:             &stripeData.StripeCustomerID,
				DefaultPaymentMethodId: stripeData.StripeDefaultPaymentMethodID,
			}
		}
	case app.AppTypeCustomInvoicing:
		data, err := application.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
			CustomerID: customerID,
		})
		if err != nil {
			return nil, err
		}

		if customInvoicingData, ok := data.(appcustominvoicing.CustomerData); ok {
			appData.ExternalInvoicing = &api.BillingAppCustomerDataExternalInvoicing{
				Labels: (*api.Labels)(lo.ToPtr(customInvoicingData.Metadata.ToMap())),
			}
		}
	case app.AppTypeSandbox:
		// Sandbox apps have no customer data
	default:
		return nil, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
	}

	return &appData, nil
}

// validateAppData rejects structurally invalid values regardless of which app
// the resolved billing profile uses, so a bad value never silently passes as
// an ignored field. fieldPrefix scopes reported error fields to the calling
// endpoint's body shape ("" for app-data, "app_data." for customer billing).
func validateAppData(ctx context.Context, fieldPrefix string, data api.UpsertAppCustomerDataRequest) error {
	if data.Stripe.IsSpecified() && !data.Stripe.IsNull() {
		stripeData, err := data.Stripe.Get()
		if err != nil {
			return err
		}

		if stripeData.CustomerId == nil {
			return apierrors.NewBadRequestError(ctx, fmt.Errorf("stripe customer id is required"), apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  fieldPrefix + "stripe.customer_id",
					Rule:   "required",
					Reason: "Stripe Customer ID is required",
					Source: apierrors.InvalidParamSourceBody,
				},
			})
		}
	}

	return nil
}

// applyAppData replaces the customer's data for the resolved payment app: a
// provided value is validated and stored, while an omitted or explicitly null
// field deletes the existing data. Valid fields for apps other than the
// resolved one are ignored.
func applyAppData(ctx context.Context, application app.App, customerID customer.CustomerID, fieldPrefix string, data api.UpsertAppCustomerDataRequest) error {
	if err := validateAppData(ctx, fieldPrefix, data); err != nil {
		return err
	}

	switch application.GetType() {
	case app.AppTypeStripe:
		if !data.Stripe.IsSpecified() || data.Stripe.IsNull() {
			return application.DeleteCustomerData(ctx, app.DeleteAppInstanceCustomerDataInput{
				CustomerID: customerID,
			})
		}

		stripeData, err := data.Stripe.Get()
		if err != nil {
			return err
		}

		return application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
			CustomerID: customerID,
			Data: appstripe.CustomerData{
				StripeCustomerID:             *stripeData.CustomerId,
				StripeDefaultPaymentMethodID: stripeData.DefaultPaymentMethodId,
			},
		})
	case app.AppTypeCustomInvoicing:
		if !data.ExternalInvoicing.IsSpecified() || data.ExternalInvoicing.IsNull() {
			return application.DeleteCustomerData(ctx, app.DeleteAppInstanceCustomerDataInput{
				CustomerID: customerID,
			})
		}

		externalInvoicingData, err := data.ExternalInvoicing.Get()
		if err != nil {
			return err
		}

		customerData := appcustominvoicing.CustomerData{}
		if externalInvoicingData.Labels != nil {
			customerData.Metadata = models.Metadata(*externalInvoicingData.Labels)
		}

		return application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
			CustomerID: customerID,
			Data:       customerData,
		})
	case app.AppTypeSandbox:
		// The upsert maintains the app-customer relationship even though
		// sandbox apps have no customer data.
		return application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
			CustomerID: customerID,
			Data:       appsandbox.CustomerData{},
		})
	default:
		return apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
	}
}
