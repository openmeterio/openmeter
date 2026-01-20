package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateAppStripeCheckoutSessionRequest  = appstripeentity.CreateCheckoutSessionInput
	CreateAppStripeCheckoutSessionResponse = api.CreateStripeCheckoutSessionResult
	CreateAppStripeCheckoutSessionHandler  httptransport.Handler[CreateAppStripeCheckoutSessionRequest, CreateAppStripeCheckoutSessionResponse]
)

// CreateAppStripeCheckoutSession returns a handler for creating a checkout session.
func (h *handler) CreateAppStripeCheckoutSession() CreateAppStripeCheckoutSessionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateAppStripeCheckoutSessionRequest, error) {
			body := api.CreateStripeCheckoutSessionRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("field to decode create app stripe checkout session request: %w", err)
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var createCustomerInput *customer.CreateCustomerInput
			var customerId *customer.CustomerID
			var customerKey *string

			// Try to parse as customer create first
			maybeCustomerCreate, asCustomerCreateErr := body.Customer.AsCustomerCreate()
			if asCustomerCreateErr == nil && maybeCustomerCreate.Name != "" {
				createCustomerInput = &customer.CreateCustomerInput{
					Namespace:      namespace,
					CustomerMutate: customerhttpdriver.MapCustomerCreate(maybeCustomerCreate),
				}
			}

			// Try to parse as customer ID second
			if createCustomerInput == nil {
				apiCustomerId, asCustomerIdErr := body.Customer.AsCustomerId()
				if asCustomerIdErr == nil && apiCustomerId.Id != "" {
					customerId = &customer.CustomerID{
						Namespace: namespace,
						ID:        apiCustomerId.Id,
					}
				}
			}

			// Try to parse as customer key third
			if createCustomerInput == nil && customerId == nil {
				maybeCustomerKey, asCustomerKeyErr := body.Customer.AsCustomerKey()

				if asCustomerKeyErr == nil && maybeCustomerKey.Key != "" {
					customerKey = &maybeCustomerKey.Key
				}
			}

			// One of the three must be provided
			if createCustomerInput == nil && customerId == nil && customerKey == nil {
				return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("customer is required")
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
					return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("failed to get customer by key: %w", err)
				}

				if cus != nil && cus.IsDeleted() {
					return CreateAppStripeCheckoutSessionRequest{},
						models.NewGenericPreConditionFailedError(
							fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
						)
				}

				customerId = lo.ToPtr(cus.GetID())
			}

			// Create request
			req := CreateAppStripeCheckoutSessionRequest{
				Namespace:           namespace,
				CustomerID:          customerId,
				CreateCustomerInput: createCustomerInput,
				StripeCustomerID:    body.StripeCustomerId,
				Options:             body.Options,
			}

			// Resolve app ID from request or from billing profile
			if body.AppId != nil {
				req.AppID = app.AppID{Namespace: namespace, ID: *body.AppId}
			} else {
				appId, err := h.billingService.ResolveAppIDFromBillingProfile(ctx, namespace, customerId)
				if err != nil {
					return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve app id from billing profile: %w", err)
				}

				req.AppID = appId
			}

			return req, nil
		},
		func(ctx context.Context, request CreateAppStripeCheckoutSessionRequest) (CreateAppStripeCheckoutSessionResponse, error) {
			out, err := h.service.CreateCheckoutSession(ctx, request)
			if err != nil {
				return CreateAppStripeCheckoutSessionResponse{}, fmt.Errorf("failed to create app stripe checkout session: %w", err)
			}

			response := CreateAppStripeCheckoutSessionResponse{
				CancelURL:        out.CancelURL,
				CustomerId:       out.CustomerID.ID,
				Mode:             api.StripeCheckoutSessionMode(out.Mode),
				ReturnURL:        out.ReturnURL,
				SessionId:        out.SessionID,
				SetupIntentId:    out.SetupIntentID,
				StripeCustomerId: out.StripeCustomerID,
				SuccessURL:       out.SuccessURL,
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
		commonhttp.JSONResponseEncoderWithStatus[CreateAppStripeCheckoutSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createAppStripeCheckoutSession"),
		)...,
	)
}
