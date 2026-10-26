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
	UpdateCustomerBillingProfileRequest struct {
		Customer  customer.CustomerID
		ProfileID *string
	}
	UpdateCustomerBillingProfileResponse = api.BillingCustomerProfile
	UpdateCustomerBillingProfileParams   = string
	UpdateCustomerBillingProfileHandler  = httptransport.HandlerWithArgs[UpdateCustomerBillingProfileRequest, UpdateCustomerBillingProfileResponse, UpdateCustomerBillingProfileParams]
)

func (h *handler) UpdateCustomerBillingProfile() UpdateCustomerBillingProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID UpdateCustomerBillingProfileParams) (UpdateCustomerBillingProfileRequest, error) {
			var body api.UpsertCustomerBillingProfileRequest
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCustomerBillingProfileRequest{}, err
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCustomerBillingProfileRequest{}, err
			}

			input := UpdateCustomerBillingProfileRequest{
				Customer: customer.CustomerID{Namespace: namespace, ID: customerID},
			}
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &input.Customer})
			if err != nil {
				return input, err
			}

			if cus.IsDeleted() {
				return input, apierrors.NewGoneError(ctx, errors.New("customer is deleted"))
			}

			if body.BillingProfile.IsSpecified() && !body.BillingProfile.IsNull() {
				reference, err := body.BillingProfile.Get()
				if err != nil {
					return input, fmt.Errorf("read billing profile reference: %w", err)
				}

				input.ProfileID = lo.ToPtr(reference.Id)
			}

			return input, nil
		},
		func(ctx context.Context, input UpdateCustomerBillingProfileRequest) (UpdateCustomerBillingProfileResponse, error) {
			var customerProfile billing.CustomerBillingProfile
			err := h.billingService.WithLock(ctx, input.Customer, func(ctx context.Context) error {
				if input.ProfileID != nil {
					assignedProfile, err := h.billingService.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
						Customer:  input.Customer,
						ProfileID: *input.ProfileID,
					})
					if err != nil {
						return fmt.Errorf("set customer billing profile: %w", err)
					}

					customerProfile = assignedProfile

					return nil
				}

				if err := h.billingService.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{
					Customer: input.Customer,
				}); err != nil && !errors.Is(err, billing.ErrCustomerOverrideNotFound) {
					return fmt.Errorf("clear customer billing profile: %w", err)
				}

				// Resolve before committing so a missing default cannot leave the customer
				// unpinned after a failed request.
				resolvedProfile, err := h.billingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
					Customer: input.Customer,
				})
				if err != nil {
					return fmt.Errorf("resolve customer billing profile: %w", err)
				}

				customerProfile = resolvedProfile

				return nil
			})
			if err != nil {
				return UpdateCustomerBillingProfileResponse{}, err
			}

			return mapCustomerBillingProfileToAPI(customerProfile), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing-profile"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
