package customersbilling

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

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

			if override.MergedProfile.Apps == nil {
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("apps are not expanded in billing profile"))
			}

			// Only one app can be in the billing profile right now; we pick
			// the payment app.
			application := override.MergedProfile.Apps.Payment

			if err := applyAppData(ctx, application, request.CustomerID, "", request.Data); err != nil {
				return resp, err
			}

			appData, err := buildAppData(ctx, application, request.CustomerID)
			if err != nil {
				return resp, err
			}

			return lo.FromPtr(appData), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingAppDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing-app-data"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
