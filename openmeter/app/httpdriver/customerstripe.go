package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
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
			stripeApp, err := h.resolveCustomerApp(ctx, req.CustomerId, app.AppTypeStripe, nil)
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

// getAPIStripeCustomerAppData returns the stripe customer app data for the given customer id.
func (h *handler) getAPIStripeCustomerAppData(ctx context.Context, customerID customer.CustomerID) (api.StripeCustomerAppData, error) {
	// Resolve the customer app by billing profile
	genericApp, err := h.resolveCustomerApp(ctx, customerID, app.AppTypeStripe, nil)
	if err != nil {
		return GetCustomerStripeAppDataResponse{}, err
	}

	// Enforce stripe apptype, see app type filter above
	stripeApp, ok := genericApp.(appstripeentityapp.App)
	if !ok {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("customer app is not a stripe app")
	}

	// List customer data for the specific stripe app
	resp, err := h.service.ListCustomerData(ctx, app.ListCustomerInput{
		AppID:      lo.ToPtr(stripeApp.GetID()),
		CustomerID: customerID,
	})
	if err != nil {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("failed to get customer stripe app data: %w", err)
	}

	// We need to check length of array because we use the list method to get the customer data
	if len(resp.Items) == 0 {
		return GetCustomerStripeAppDataResponse{}, models.NewGenericNotFoundError(
			fmt.Errorf("no customer stripe app data found"),
		)
	}

	// Enforce stripe customer data type
	stripeCustomerAppData, ok := resp.Items[0].CustomerData.(appstripeentity.CustomerData)
	if !ok {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("customer app data is not a stripe app data: %w", err)
	}

	// Convert to API stripe customer app data
	apiStripeCustomerAppData, err := h.toAPIStripeCustomerAppData(stripeCustomerAppData, stripeApp)
	if err != nil {
		return GetCustomerStripeAppDataResponse{}, fmt.Errorf("error converting to stripe customer app: %w", err)
	}

	return apiStripeCustomerAppData, nil
}
