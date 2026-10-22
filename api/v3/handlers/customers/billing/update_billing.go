package customersbilling

import (
	"context"
	"errors"
	"fmt"
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
	UpdateCustomerBillingRequest struct {
		CustomerID customer.CustomerID
		ProfileID  *billing.ProfileID
		AppData    *api.BillingAppCustomerData
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

			var profileID *billing.ProfileID
			if body.BillingProfile != nil {
				profileID = &billing.ProfileID{
					Namespace: namespace,
					ID:        body.BillingProfile.Id,
				}
			}

			return UpdateCustomerBillingRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
				ProfileID: profileID,
				AppData:   body.AppData,
			}, nil
		},
		func(ctx context.Context, request UpdateCustomerBillingRequest) (UpdateCustomerBillingResponse, error) {
			resp := UpdateCustomerBillingResponse{}

			var billingProfile *billing.Profile
			var err error
			// If the profile ID is not provided, we use the default profile
			if request.ProfileID == nil {
				billingProfile, err = h.billingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
					Namespace: request.CustomerID.Namespace,
				})
				if err != nil {
					return resp, err
				}
			} else {
				// Get the billing profile by the provided profile ID
				billingProfile, err = h.billingService.GetProfile(ctx, billing.GetProfileInput{
					Profile: *request.ProfileID,
					Expand: billing.ProfileExpand{
						Apps: true,
					},
				})
				if err != nil {
					return resp, err
				}
			}

			if billingProfile.Apps == nil {
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("apps are not expanded in billing profile"))
			}

			// If app data is omitted, only the billing profile is updated and the
			// customer can provide app data later.
			if request.AppData != nil {
				// TODO: Only one app ID can be in the billing profile right now.
				// We pick the payment app for now.
				appData, err := h.applyAppCustomerData(ctx, billingProfile.Apps.Payment, request.CustomerID, *request.AppData)
				if err != nil {
					return resp, err
				}

				resp.AppData = appData
			}

			var customerOverride billing.CustomerOverrideWithDetails
			// Override the billing profile if an ID was provided
			if request.ProfileID != nil {
				customerOverride, err = h.billingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
					Namespace:  request.CustomerID.Namespace,
					CustomerID: request.CustomerID.ID,
					ProfileID:  billingProfile.ID,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to update billing profile: %w", err)
				}

				if customerOverride.CustomerOverride == nil || customerOverride.CustomerOverride.Profile == nil {
					return resp, apierrors.NewInternalError(ctx, errors.New("customer override is not found"))
				}

				resp.BillingProfile = &api.BillingProfileReference{
					Id: customerOverride.CustomerOverride.Profile.ID,
				}
			} else {
				customerOverride, err = h.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
					Customer: request.CustomerID,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to resolve effective billing profile: %w", err)
				}

				if customerOverride.CustomerOverride != nil && customerOverride.CustomerOverride.Profile != nil {
					resp.BillingProfile = &api.BillingProfileReference{
						Id: customerOverride.CustomerOverride.Profile.ID,
					}
				}
			}

			resp.EffectiveBillingProfile = api.BillingProfileReference{
				Id: customerOverride.MergedProfile.ID,
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
