package customersbilling

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
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

			resp.BillingProfile = &api.BillingProfileReference{
				Id: billingProfile.ID,
			}

			if billingProfile.Apps == nil {
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("apps are not expanded in billing profile"))
			}

			// TODO: Only one app ID can be in the billing profile right now.
			// We pick the payment app for now.
			application := billingProfile.Apps.Payment
			var appData app.CustomerData

			switch application.GetType() {
			case app.AppTypeStripe:
				if request.AppData == nil || request.AppData.Stripe == nil {
					return resp, apierrors.NewBadRequestError(ctx, fmt.Errorf("stripe data is required"), apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "app_data.stripe",
							Rule:   "required",
							Reason: "Stripe data is required",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
				if request.AppData == nil || request.AppData.Stripe.CustomerId == nil {
					return resp, apierrors.NewBadRequestError(ctx, fmt.Errorf("stripe customer id is required"), apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "stripe.customer_id",
							Rule:   "required",
							Reason: "Stripe Customer ID is required",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				resp.AppData = &api.BillingAppCustomerData{
					Stripe: request.AppData.Stripe,
				}
				appData = appstripeentity.CustomerData{
					StripeCustomerID:             *request.AppData.Stripe.CustomerId,
					StripeDefaultPaymentMethodID: request.AppData.Stripe.DefaultPaymentMethodId,
				}
			case app.AppTypeCustomInvoicing:
				resp.AppData = &api.BillingAppCustomerData{
					ExternalInvoicing: request.AppData.ExternalInvoicing,
				}
				appData = appcustominvoicing.CustomerData{}
				if request.AppData != nil && request.AppData.ExternalInvoicing != nil && request.AppData.ExternalInvoicing.Labels != nil {
					appData = appcustominvoicing.CustomerData{
						Metadata: models.Metadata(*request.AppData.ExternalInvoicing.Labels),
					}
				}
			case app.AppTypeSandbox:
				appData = appsandbox.CustomerData{}
			default:
				return resp, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
			}

			// Update the customer data for the app
			err = application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
				CustomerID: request.CustomerID,
				Data:       appData,
			})
			if err != nil {
				return resp, fmt.Errorf("failed to update customer data: %w", err)
			}

			// Override the billing profile if an ID was provided
			if request.ProfileID != nil {
				_, err = h.billingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
					Namespace:  request.CustomerID.Namespace,
					CustomerID: request.CustomerID.ID,
					ProfileID:  billingProfile.ID,
				})
				if err != nil {
					return resp, fmt.Errorf("failed to update billing profile: %w", err)
				}
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
