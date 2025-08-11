package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	apphttphandler "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerStripeAppDataResponse = api.StripeCustomerAppData
	GetCustomerStripeAppDataHandler  httptransport.HandlerWithArgs[GetCustomerStripeAppDataRequest, GetCustomerStripeAppDataResponse, GetCustomerStripeAppDataParams]
)

type GetCustomerStripeAppDataRequest struct {
	CustomerID customer.CustomerID
}

type GetCustomerStripeAppDataParams struct {
	CustomerIdOrKey string
}

// GetCustomerStripeAppData returns a handler for listing customers app data.
func (h *handler) GetCustomerStripeAppData() GetCustomerStripeAppDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerStripeAppDataParams) (GetCustomerStripeAppDataRequest, error) {
			// Resolve the namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerStripeAppDataRequest{}, err
			}

			// Resolve the customer by id or key
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return GetCustomerStripeAppDataRequest{}, err
			}

			// Construct the request
			req := GetCustomerStripeAppDataRequest{
				CustomerID: cus.GetID(),
			}

			return req, nil
		},
		func(ctx context.Context, request GetCustomerStripeAppDataRequest) (GetCustomerStripeAppDataResponse, error) {
			return h.getAPIStripeCustomerAppData(ctx, request.CustomerID)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerStripeAppDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getCustomerStripeAppData"),
		)...,
	)
}

type UpsertCustomerStripeAppDataRequest struct {
	CustomerId customer.CustomerID
	Data       api.StripeCustomerAppDataBase
}

type UpsertCustomerStripeAppDataParams struct {
	CustomerIdOrKey string
}

type (
	UpsertCustomerStripeAppDataResponse = api.StripeCustomerAppData
	UpsertCustomerStripeAppDataHandler  httptransport.HandlerWithArgs[UpsertCustomerStripeAppDataRequest, UpsertCustomerStripeAppDataResponse, UpsertCustomerStripeAppDataParams]
)

// UpsertCustomerStripeAppData returns a new httptransport.Handler for creating a customer.
func (h *handler) UpsertCustomerStripeAppData() UpsertCustomerStripeAppDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpsertCustomerStripeAppDataParams) (UpsertCustomerStripeAppDataRequest, error) {
			// Parse the request body
			body := api.StripeCustomerAppDataBase{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpsertCustomerStripeAppDataRequest{}, fmt.Errorf("field to decode upsert customer data request: %w", err)
			}

			// Resolve the namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertCustomerStripeAppDataRequest{}, err
			}

			// Resolve the customer by id or key
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return UpsertCustomerStripeAppDataRequest{}, err
			}

			return UpsertCustomerStripeAppDataRequest{
				CustomerId: cus.GetID(),
				Data:       body,
			}, nil
		},
		func(ctx context.Context, req UpsertCustomerStripeAppDataRequest) (UpsertCustomerStripeAppDataResponse, error) {
			// Resolve the customer app by billing profile
			stripeApp, err := h.billingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
				CustomerID: req.CustomerId,
				AppType:    app.AppTypeStripe,
			})
			if err != nil {
				return api.StripeCustomerAppData{}, err
			}

			// Upsert the customer data
			err = stripeApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
				CustomerID: req.CustomerId,
				Data:       fromAPIAppStripeCustomerDataBase(req.Data),
			})
			if err != nil {
				return api.StripeCustomerAppData{}, err
			}

			return h.getAPIStripeCustomerAppData(ctx, req.CustomerId)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpsertCustomerStripeAppDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsertCustomerStripeAppData"),
		)...,
	)
}

type (
	CreateStripeCustomerPortalSessionResponse = api.StripeCustomerPortalSession
	CreateStripeCustomerPortalSessionHandler  httptransport.HandlerWithArgs[CreateStripeCustomerPortalSessionRequest, CreateStripeCustomerPortalSessionResponse, CreateStripeCustomerPortalSessionParams]
)

type CreateStripeCustomerPortalSessionRequest struct {
	customerId customer.CustomerID
	params     api.CreateStripeCustomerPortalSessionParams
}

type CreateStripeCustomerPortalSessionParams struct {
	CustomerIdOrKey string
}

// CreateStripeCustomerPortalSession returns a handler for creating a checkout session.
func (h *handler) CreateStripeCustomerPortalSession() CreateStripeCustomerPortalSessionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CreateStripeCustomerPortalSessionParams) (CreateStripeCustomerPortalSessionRequest, error) {
			// Parse request body
			body := api.CreateStripeCustomerPortalSessionParams{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateStripeCustomerPortalSessionRequest{}, fmt.Errorf("field to decode create app stripe checkout session request: %w", err)
			}

			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateStripeCustomerPortalSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: namespace,
				},
			})
			if err != nil {
				return CreateStripeCustomerPortalSessionRequest{}, err
			}

			// Create request
			req := CreateStripeCustomerPortalSessionRequest{
				customerId: cus.GetID(),
				params:     body,
			}

			return req, nil
		},
		func(ctx context.Context, request CreateStripeCustomerPortalSessionRequest) (CreateStripeCustomerPortalSessionResponse, error) {
			// Resolve the customer app by billing profile
			genericApp, err := h.billingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
				CustomerID: request.customerId,
				AppType:    app.AppTypeStripe,
			})
			if err != nil {
				return CreateStripeCustomerPortalSessionResponse{}, err
			}

			// Enforce stripe apptype, see app type filter above
			stripeApp, ok := genericApp.(appstripeentityapp.App)
			if !ok {
				return CreateStripeCustomerPortalSessionResponse{}, fmt.Errorf("customer app is not a stripe app")
			}

			// Create the portal session
			portalSession, err := h.service.CreatePortalSession(ctx, appstripeentity.CreateStripePortalSessionInput{
				AppID:           stripeApp.GetID(),
				CustomerID:      request.customerId,
				ConfigurationID: request.params.ConfigurationId,
				ReturnURL:       request.params.ReturnUrl,
				Locale:          request.params.Locale,
			})
			if err != nil {
				return CreateStripeCustomerPortalSessionResponse{}, fmt.Errorf("failed to create portal session: %w", err)
			}

			return toAPIStripePortalSession(portalSession), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateStripeCustomerPortalSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createStripeCustomerPortalSession"),
		)...,
	)
}

// getAPIStripeCustomerAppData returns the stripe customer app data for the given customer id.
func (h *handler) getAPIStripeCustomerAppData(ctx context.Context, customerID customer.CustomerID) (api.StripeCustomerAppData, error) {
	// Resolve the customer app by billing profile
	genericApp, err := h.billingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
		CustomerID: customerID,
		AppType:    app.AppTypeStripe,
	})
	if err != nil {
		return GetCustomerStripeAppDataResponse{}, err
	}

	// Enforce stripe apptype, see app type filter above
	stripeApp, ok := genericApp.(appstripeentityapp.App)
	if !ok {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("customer app is not a stripe app")
	}

	// List customer data for the specific stripe app
	customerData, err := h.service.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      stripeApp.GetID(),
		CustomerID: customerID,
	})
	if err != nil {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("failed to get customer stripe app data: %w", err)
	}

	// Convert to API stripe customer app data
	apiStripeCustomerAppData := apphttphandler.ToAPIStripeCustomerAppData(customerData, stripeApp)

	return apiStripeCustomerAppData, nil
}
