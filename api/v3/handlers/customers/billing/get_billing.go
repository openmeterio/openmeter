package customersbilling

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
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
			resp := GetCustomerBillingResponse{}
			override, err := h.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: request.CustomerID,
				Expand: billing.CustomerOverrideExpand{
					Apps: true,
				},
			})
			if err != nil {
				return resp, err
			}

			// TODO: Only one app ID can be in the billing profile right now.
			// We pick the payment app for now.
			application := override.MergedProfile.Apps.Payment
			data, err := application.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
				CustomerID: request.CustomerID,
			})
			if err != nil {
				if !app.IsAppCustomerPreConditionError(err) {
					return resp, err
				}

				// The profile names this app as payment app, but the customer has no
				// data for it yet. This is a supported state; show it as unconfigured.
				data = nil // (untyped nil)
			}

			apiAppData, err := fromAPIBillingAppCustomerData(ctx, application, data)
			if err != nil {
				return resp, err
			}

			return GetCustomerBillingResponse{
				BillingProfile: &api.BillingProfileReference{
					Id: override.MergedProfile.ID,
				},
				AppData: apiAppData,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerBillingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-billing"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
