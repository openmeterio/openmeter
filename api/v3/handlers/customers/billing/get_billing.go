package customersbilling

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
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

			if override.MergedProfile.Apps == nil {
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("apps are not expanded in billing profile"))
			}

			// Only one app can be in the billing profile right now; we pick
			// the payment app.
			application := override.MergedProfile.Apps.Payment

			appData, err := buildAppData(ctx, application, request.CustomerID)
			if err != nil {
				return resp, err
			}

			resp = GetCustomerBillingResponse{
				BillingProfile: &api.BillingProfileReference{
					Id: override.MergedProfile.ID,
				},
				AppData: appData,
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerBillingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-billing"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
