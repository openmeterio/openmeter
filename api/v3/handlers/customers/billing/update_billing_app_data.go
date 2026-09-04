package customersbilling

import (
	"context"
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateCustomerBillingAppDataRequest struct {
		CustomerID customer.CustomerID
		Data       api.UpsertAppCustomerDataRequest
	}
	UpdateCustomerBillingAppDataResponse = api.BillingAppCustomerData
	UpdateCustomerBillingAppDataParams   = string
	UpdateCustomerBillingAppDataHandler  httptransport.HandlerWithArgs[UpdateCustomerBillingAppDataRequest, UpdateCustomerBillingAppDataResponse, UpdateCustomerBillingAppDataParams]
)

func (h *handler) UpdateCustomerBillingAppData() UpdateCustomerBillingAppDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID UpdateCustomerBillingAppDataParams) (UpdateCustomerBillingAppDataRequest, error) {
			body := api.UpsertAppCustomerDataRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCustomerBillingAppDataRequest{}, err
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerBillingAppDataRequest{}, err
			}

			// Validate customer exists and is not deleted
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
			})
			if err != nil {
				return UpdateCustomerBillingAppDataRequest{}, err
			}

			if cus.IsDeleted() {
				return UpdateCustomerBillingAppDataRequest{},
					apierrors.NewGoneError(
						ctx,
						errors.New("customer is deleted"),
					)
			}

			return UpdateCustomerBillingAppDataRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
				Data: body,
			}, nil
		},
		func(ctx context.Context, request UpdateCustomerBillingAppDataRequest) (UpdateCustomerBillingAppDataResponse, error) {
			resp := UpdateCustomerBillingAppDataResponse{}
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
			appData, err := h.applyAppCustomerData(ctx, override.MergedProfile.Apps.Payment, request.CustomerID, api.BillingAppCustomerData(request.Data))
			if err != nil {
				return resp, err
			}

			return *appData, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingAppDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing-app-data"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
