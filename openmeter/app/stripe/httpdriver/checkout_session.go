package httpdriver

import (
	"context"
	"fmt"
	"net/http"

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

			// Try to parse customer field as customer ID first
			apiCustomerId, err := body.Customer.AsCustomerId()
			if err == nil && apiCustomerId.Id != "" {
				customerId = &customer.CustomerID{
					Namespace: namespace,
					ID:        apiCustomerId.Id,
				}
			}

			// If no customerId found try to parse customer field as customer key
			if customerId == nil {
				maybeCustomerKey, err := body.Customer.AsCustomerKey()

				if err == nil && maybeCustomerKey.Key != "" {
					customerKey = &maybeCustomerKey.Key
				}
			}

			// If no customerKey found try to parse customer field as customer input
			if customerId == nil && customerKey == nil {
				// If err try to parse customer field as customer input
				customerCreate, err := body.Customer.AsCustomerCreate()
				if err != nil {
					return CreateAppStripeCheckoutSessionRequest{}, models.NewGenericValidationError(
						fmt.Errorf("failed to decode customer: %w", err),
					)
				}

				createCustomerInput = &customer.CreateCustomerInput{
					Namespace:      namespace,
					CustomerMutate: customerhttpdriver.MapCustomerCreate(customerCreate),
				}
			}

			req := CreateAppStripeCheckoutSessionRequest{
				Namespace:           namespace,
				CustomerID:          customerId,
				CustomerKey:         customerKey,
				CreateCustomerInput: createCustomerInput,
				StripeCustomerID:    body.StripeCustomerId,
				Options:             body.Options,
			}

			if body.AppId != nil {
				req.AppID = &app.AppID{Namespace: namespace, ID: *body.AppId}
			}

			return req, nil
		},
		func(ctx context.Context, request CreateAppStripeCheckoutSessionRequest) (CreateAppStripeCheckoutSessionResponse, error) {
			out, err := h.service.CreateCheckoutSession(ctx, request)
			if err != nil {
				return CreateAppStripeCheckoutSessionResponse{}, fmt.Errorf("failed to create app stripe checkout session: %w", err)
			}

			return CreateAppStripeCheckoutSessionResponse{
				CancelURL:        out.CancelURL,
				CustomerId:       out.CustomerID.ID,
				Mode:             api.StripeCheckoutSessionMode(out.Mode),
				ReturnURL:        out.ReturnURL,
				SessionId:        out.SessionID,
				SetupIntentId:    out.SetupIntentID,
				StripeCustomerId: out.StripeCustomerID,
				SuccessURL:       out.SuccessURL,
				Url:              out.URL,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateAppStripeCheckoutSessionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createAppStripeCheckoutSession"),
		)...,
	)
}
