package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
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

			// Create request
			req := CreateAppStripeCheckoutSessionRequest{
				Namespace:           namespace,
				CustomerID:          customerId,
				CustomerKey:         customerKey,
				CreateCustomerInput: createCustomerInput,
				StripeCustomerID:    body.StripeCustomerId,
				Options:             body.Options,
			}

			if body.AppId != nil {
				req.AppID = app.AppID{Namespace: namespace, ID: *body.AppId}
			} else {
				// Get the billing profiles
				billingProfileList, err := h.billingService.ListProfiles(ctx, billing.ListProfilesInput{
					Namespace: namespace,
				})
				if err != nil {
					return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("failed to get billing profile: %w", err)
				}

				// Find the billing profile with the stripe payment app
				// Prioritize the default profile
				var stripeApps []app.App
				var foundDefault bool

				for _, profile := range billingProfileList.Items {
					if foundDefault {
						break
					}

					if profile.Apps.Payment.GetType() == app.AppTypeStripe {
						req.AppID = profile.Apps.Payment.GetID()
						stripeApps = append(stripeApps, profile.Apps.Payment)

						if profile.Default {
							foundDefault = true
						}
					}
				}

				// If no default profile is found, check if there is only one stripe app and use it
				if !foundDefault {
					// If there is no stripe app, return an error
					if len(stripeApps) == 0 {
						return CreateAppStripeCheckoutSessionRequest{}, models.NewGenericNotFoundError(
							fmt.Errorf("no stripe billing profile found, please create a billing profile with a stripe app"),
						)
					} else {
						return CreateAppStripeCheckoutSessionRequest{}, models.NewGenericNotFoundError(
							fmt.Errorf("you have stripe billing profiles, but none is marked as default"),
						)
					}
				}
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
