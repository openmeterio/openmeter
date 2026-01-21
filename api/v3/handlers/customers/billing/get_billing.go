package customersbilling

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerBillingRequest struct {
		CustomerID customer.CustomerID
	}
	GetCustomerBillingResponse = api.BillingCustomerData
	GetCustomerBillingParams   = string
	GetCustomerBillingHandler  httptransport.HandlerWithArgs[GetCustomerBillingRequest, GetCustomerBillingResponse, GetCustomerBillingParams]
)

func (h *handler) GetCustomerBilling() GetCustomerBillingHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID GetCustomerBillingParams) (GetCustomerBillingRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerBillingRequest{}, err
			}

			return GetCustomerBillingRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
			}, nil
		},
		func(ctx context.Context, request GetCustomerBillingRequest) (GetCustomerBillingResponse, error) {
			override, err := h.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: request.CustomerID,
				Expand: billing.CustomerOverrideExpand{
					Apps: true,
				},
			})
			if err != nil {
				return GetCustomerBillingResponse{}, err
			}

			appData := api.BillingAppCustomerData{}

			// TODO: Only one app ID can be in the billing profile right now.
			// We pick the payment app for now.
			application := override.MergedProfile.Apps.Payment
			data, err := application.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
				CustomerID: request.CustomerID,
			})
			if err != nil {
				return GetCustomerBillingResponse{}, err
			}

			switch override.MergedProfile.Apps.Payment.GetType() {
			case app.AppTypeStripe:
				if data, ok := data.(appstripeentity.CustomerData); ok {
					// TODO: we don't have metadata on the stripe customer data yet
					appData.Stripe = &api.BillingAppCustomerDataStripe{
						CustomerId:             &data.StripeCustomerID,
						DefaultPaymentMethodId: data.StripeDefaultPaymentMethodID,
					}
				}
			case app.AppTypeCustomInvoicing:
				if data, ok := data.(appcustominvoicing.CustomerData); ok {
					appData.ExternalInvoicing = &api.BillingAppCustomerDataExternalInvoicing{
						Labels: (*api.Labels)(lo.ToPtr(data.Metadata.ToMap())),
					}
				}
			case app.AppTypeSandbox:
				// No app data
			default:
				return GetCustomerBillingResponse{}, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
			}

			return GetCustomerBillingResponse{
					BillingProfile: &api.BillingProfileReference{
						Id: override.MergedProfile.ID,
					},
					AppData: &appData,
				},
				nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerBillingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-billing"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
