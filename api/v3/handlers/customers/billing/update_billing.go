package customersbilling

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/oapi-codegen/nullable"
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
	UpdateCustomerBillingRequest struct {
		CustomerID     customer.CustomerID
		BillingProfile nullable.Nullable[api.BillingProfileReference]
		AppData        *api.UpsertAppCustomerDataRequest
	}
	UpdateCustomerBillingResponse = api.BillingCustomerData
	UpdateCustomerBillingParams   = string
	UpdateCustomerBillingHandler  httptransport.HandlerWithArgs[UpdateCustomerBillingRequest, UpdateCustomerBillingResponse, UpdateCustomerBillingParams]
)

func (h *handler) UpdateCustomerBilling() UpdateCustomerBillingHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID UpdateCustomerBillingParams) (UpdateCustomerBillingRequest, error) {
			body := api.UpsertCustomerBillingDataRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCustomerBillingRequest{}, err
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerBillingRequest{}, err
			}

			// Validate customer exists and is not deleted
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
			})
			if err != nil {
				return UpdateCustomerBillingRequest{}, err
			}

			if cus.IsDeleted() {
				return UpdateCustomerBillingRequest{},
					apierrors.NewGoneError(
						ctx,
						errors.New("customer is deleted"),
					)
			}

			return UpdateCustomerBillingRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
				BillingProfile: body.BillingProfile,
				AppData:        body.AppData,
			}, nil
		},
		func(ctx context.Context, request UpdateCustomerBillingRequest) (UpdateCustomerBillingResponse, error) {
			resp := UpdateCustomerBillingResponse{}

			// The request replaces the customer's billing data: a provided
			// billing profile pins the customer to it, while an omitted or
			// null profile removes the existing pin so the default profile
			// applies. App data is validated against the target profile
			// before any mutation happens below.
			pinProfile := request.BillingProfile.IsSpecified() && !request.BillingProfile.IsNull()

			var billingProfile *billing.Profile
			var err error

			if pinProfile {
				profileRef, err := request.BillingProfile.Get()
				if err != nil {
					return resp, err
				}

				billingProfile, err = h.billingService.GetProfile(ctx, billing.GetProfileInput{
					Profile: billing.ProfileID{
						Namespace: request.CustomerID.Namespace,
						ID:        profileRef.Id,
					},
					Expand: billing.ProfileExpand{
						Apps: true,
					},
				})
				if err != nil {
					return resp, err
				}
			} else {
				billingProfile, err = h.billingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
					Namespace: request.CustomerID.Namespace,
				})
				if err != nil {
					return resp, err
				}
			}

			if billingProfile.Apps == nil {
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("apps are not expanded in billing profile"))
			}

			// Only one app can be in the billing profile right now; we pick
			// the payment app.
			application := billingProfile.Apps.Payment

			if err := applyAppData(ctx, application, request.CustomerID, "app_data.", lo.FromPtr(request.AppData)); err != nil {
				return resp, err
			}

			if pinProfile {
				_, err := h.billingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
					Namespace:  request.CustomerID.Namespace,
					CustomerID: request.CustomerID.ID,
					ProfileID:  billingProfile.ID,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to update billing profile: %w", err)
				}
			} else {
				err := h.billingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
					Customer: request.CustomerID,
				})
				if err != nil {
					// Removing a pin that does not exist is a no-op
					var notFoundError billing.NotFoundError
					if !errors.As(err, &notFoundError) {
						return resp, err
					}
				}
			}

			resp.BillingProfile = &api.BillingProfileReference{
				Id: billingProfile.ID,
			}

			respAppData, err := buildAppData(ctx, application, request.CustomerID)
			if err != nil {
				return resp, err
			}

			resp.AppData = respAppData

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
