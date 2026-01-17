package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateStripeCheckoutSessionRequest  = appstripeentity.CreateCheckoutSessionInput
	CreateStripeCheckoutSessionResponse = api.BillingAppStripeCreateCheckoutSessionResult
	CreateStripeCheckoutSessionHandler  httptransport.Handler[CreateStripeCheckoutSessionRequest, CreateStripeCheckoutSessionResponse]
)

func (h *handler) CreateStripeCheckoutSession() CreateStripeCheckoutSessionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateStripeCheckoutSessionRequest, error) {
			body := api.BillingAppStripeCreateCheckoutSessionRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateStripeCheckoutSessionRequest{}, err
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var createCustomerInput *customer.CreateCustomerInput
			var customerId *customer.CustomerID
			var customerKey *string

			// Try to parse as customer create first
			maybeCustomerCreate, asCustomerCreateErr := body.Customer.AsBillingAppStripeCustomerCreate()
			if asCustomerCreateErr == nil && maybeCustomerCreate.Name != "" {
				createCustomerInput = &customer.CreateCustomerInput{
					Namespace:      namespace,
					CustomerMutate: customerhttpdriver.MapCustomerCreate(ConvertToBillingAppStripeCustomerCreate(maybeCustomerCreate)),
				}
			}

			// Try to parse as customer ID second
			if createCustomerInput == nil {
				apiCustomerId, asCustomerIdErr := body.Customer.AsBillingAppStripeCustomerId()
				if asCustomerIdErr == nil && apiCustomerId.Id != "" {
					customerId = &customer.CustomerID{
						Namespace: namespace,
						ID:        apiCustomerId.Id,
					}
				}
			}

			// Try to parse as customer key third
			if createCustomerInput == nil && customerId == nil {
				maybeCustomerKey, asCustomerKeyErr := body.Customer.AsBillingAppStripeCustomerKey()

				if asCustomerKeyErr == nil && maybeCustomerKey.Key != "" {
					customerKey = &maybeCustomerKey.Key
				}
			}

			// One of the three must be provided
			if createCustomerInput == nil && customerId == nil && customerKey == nil {
				return CreateStripeCheckoutSessionRequest{}, fmt.Errorf("customer is required")
			}

			// Resolve customer ID from key
			if customerKey != nil {
				cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
					CustomerKey: lo.ToPtr(
						customer.CustomerKey{
							Namespace: namespace,
							Key:       *customerKey,
						},
					),
				})
				if err != nil {
					return CreateStripeCheckoutSessionRequest{}, fmt.Errorf("failed to get customer by key: %w", err)
				}

				if cus != nil && cus.IsDeleted() {
					return CreateStripeCheckoutSessionRequest{},
						models.NewGenericPreConditionFailedError(
							fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
						)
				}

				customerId = lo.ToPtr(cus.GetID())
			}

			// Create request
			req := CreateStripeCheckoutSessionRequest{
				Namespace:           namespace,
				CustomerID:          customerId,
				CreateCustomerInput: createCustomerInput,
				StripeCustomerID:    body.StripeCustomerId,
				Options:             ConvertToCreateStripeCheckoutSessionRequestOptions(body.Options),
			}

			// Resolve app ID from request or from billing profile
			if body.AppId != nil {
				req.AppID = app.AppID{Namespace: namespace, ID: *body.AppId}
			} else {
				appId, err := appstripehttpdriver.ResolveAppIDFromBillingProfile(ctx, namespace, customerId, h.billingService)
				if err != nil {
					return CreateStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve app id from billing profile: %w", err)
				}

				req.AppID = appId
			}

			return req, nil
		},
		func(ctx context.Context, request CreateStripeCheckoutSessionRequest) (CreateStripeCheckoutSessionResponse, error) {
			out, err := h.stripeService.CreateCheckoutSession(ctx, request)
			if err != nil {
				return CreateStripeCheckoutSessionResponse{}, fmt.Errorf("failed to create app stripe checkout session: %w", err)
			}

			response := CreateStripeCheckoutSessionResponse{
				CancelUrl:        out.CancelURL,
				CustomerId:       out.CustomerID.ID,
				Mode:             api.BillingAppStripeCheckoutSessionMode(out.Mode),
				ReturnUrl:        out.ReturnURL,
				SessionId:        out.SessionID,
				SetupIntentId:    out.SetupIntentID,
				StripeCustomerId: out.StripeCustomerID,
				SuccessUrl:       out.SuccessURL,
				Url:              out.URL,

				// Add new fields from the CreateCheckoutSessionOutput
				ClientSecret:      out.ClientSecret,
				ClientReferenceId: out.ClientReferenceID,
				CustomerEmail:     out.CustomerEmail,
				Currency:          (*api.CurrencyCode)(out.Currency),
				CreatedAt:         out.CreatedAt,
				Metadata:          out.Metadata,
				Status:            (*string)(out.Status),
				ExpiresAt:         out.ExpiresAt,
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateStripeCheckoutSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-stripe-checkout-session"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
