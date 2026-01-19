package customersbilling

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateCustomerStripePortalSessionRequest = struct {
		customerId customer.CustomerID
		options    api.BillingAppStripeCreateCustomerPortalSessionOptions
	}
	CreateCustomerStripePortalSessionResponse = api.BillingAppStripeCreateCustomerPortalSessionResult
	CreateCustomerStripePortalSessionHandler  httptransport.HandlerWithArgs[CreateCustomerStripePortalSessionRequest, CreateCustomerStripePortalSessionResponse, string]
)

func (h *handler) CreateCustomerStripePortalSession() CreateCustomerStripePortalSessionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIdParam string) (CreateCustomerStripePortalSessionRequest, error) {
			// Parse request body
			body := api.BillingCustomerStripeCreateCustomerPortalSessionRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerStripePortalSessionRequest{}, fmt.Errorf("field to decode create app stripe portal session request: %w", err)
			}

			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerStripePortalSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: namespace,
					ID:        customerIdParam,
				},
			})
			if err != nil {
				return CreateCustomerStripePortalSessionRequest{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return CreateCustomerStripePortalSessionRequest{},
					models.NewGenericPreConditionFailedError(
						fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			return CreateCustomerStripePortalSessionRequest{
				customerId: cus.GetID(),
				options:    body.StripeOptions,
			}, nil
		},
		func(ctx context.Context, request CreateCustomerStripePortalSessionRequest) (CreateCustomerStripePortalSessionResponse, error) {
			// Resolve the customer app by billing profile
			genericApp, err := h.billingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
				CustomerID: request.customerId,
				AppType:    app.AppTypeStripe,
			})
			if err != nil {
				return CreateCustomerStripePortalSessionResponse{}, err
			}

			// Enforce stripe apptype, see app type filter above
			stripeApp, ok := genericApp.(appstripeentityapp.App)
			if !ok {
				return CreateCustomerStripePortalSessionResponse{}, fmt.Errorf("customer app is not a stripe app")
			}

			// Create the portal session
			portalSession, err := h.stripeService.CreatePortalSession(ctx, appstripeentity.CreateStripePortalSessionInput{
				AppID:           stripeApp.GetID(),
				CustomerID:      request.customerId,
				ConfigurationID: request.options.ConfigurationId,
				ReturnURL:       request.options.ReturnUrl,
				Locale:          request.options.Locale,
			})
			if err != nil {
				return CreateCustomerStripePortalSessionResponse{}, fmt.Errorf("failed to create portal session: %w", err)
			}

			return ConvertToApiStripePortalSession(portalSession), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerStripePortalSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-stripe-portal-session"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
