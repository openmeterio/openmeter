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

			return UpdateCustomerBillingAppDataRequest{
				CustomerID: customer.CustomerID{
					Namespace: namespace,
					ID:        customerID,
				},
				Data: body,
			}, nil
		},
		func(ctx context.Context, request UpdateCustomerBillingAppDataRequest) (UpdateCustomerBillingAppDataResponse, error) {
			override, err := h.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: request.CustomerID,
				Expand: billing.CustomerOverrideExpand{
					Apps: true,
				},
			})
			if err != nil {
				return UpdateCustomerBillingAppDataResponse{}, err
			}

			// TODO: Only one app ID can be in the billing profile right now.
			// We pick the payment app for now.
			application := override.MergedProfile.Apps.Payment
			var appData app.CustomerData

			switch application.GetType() {
			case app.AppTypeStripe:
				if request.Data.Stripe == nil {
					return UpdateCustomerBillingAppDataResponse{}, apierrors.NewBadRequestError(ctx, fmt.Errorf("stripe data is required"), apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "stripe",
							Rule:   "required",
							Reason: "Stripe data is required",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
				if request.Data.Stripe.CustomerId == nil {
					return UpdateCustomerBillingAppDataResponse{}, apierrors.NewBadRequestError(ctx, fmt.Errorf("stripe customer id is required"), apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "stripe.customer_id",
							Rule:   "required",
							Reason: "Stripe Customer ID is required",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				appData = appstripeentity.CustomerData{
					StripeCustomerID:             *request.Data.Stripe.CustomerId,
					StripeDefaultPaymentMethodID: request.Data.Stripe.DefaultPaymentMethodId,
				}
			case app.AppTypeCustomInvoicing:
				appData = appcustominvoicing.CustomerData{}
				if request.Data.ExternalInvoicing != nil && request.Data.ExternalInvoicing.Labels != nil {
					appData = appcustominvoicing.CustomerData{
						Metadata: models.Metadata(*request.Data.ExternalInvoicing.Labels),
					}
				}
			case app.AppTypeSandbox:
				appData = appsandbox.CustomerData{}
			default:
				return UpdateCustomerBillingAppDataResponse{}, apierrors.NewInternalError(ctx, fmt.Errorf("unsupported app type: %s", application.GetType()))
			}

			err = application.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
				CustomerID: request.CustomerID,
				Data:       appData,
			})
			if err != nil {
				return UpdateCustomerBillingAppDataResponse{}, fmt.Errorf("failed to update customer data: %w", err)
			}

			return UpdateCustomerBillingAppDataResponse{
				Stripe:            request.Data.Stripe,
				ExternalInvoicing: request.Data.ExternalInvoicing,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCustomerBillingAppDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-customer-billing-app-data"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
