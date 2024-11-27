package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCustomerDataRequest  = app.ListCustomerDataInput
	ListCustomerDataResponse = api.CustomerAppDataPaginatedResponse
	ListCustomerDataHandler  httptransport.HandlerWithArgs[ListCustomerDataRequest, ListCustomerDataResponse, ListCustomerDataParams]
)

type ListCustomerDataParams struct {
	api.ListCustomerAppDataParams
	CustomerId string
}

// ListCustomerData returns a handler for listing customers app data.
func (h *handler) ListCustomerData() ListCustomerDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerDataParams) (ListCustomerDataRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerDataRequest{}, err
			}

			req := ListCustomerDataRequest{
				CustomerID: customerentity.CustomerID{
					Namespace: ns,
					ID:        params.CustomerId,
				},

				// Pagination
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, customer.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, customer.DefaultPageNumber),
				},
			}

			if params.Type != nil {
				req.Type = lo.ToPtr(appentitybase.AppType(*params.Type))
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerDataRequest) (ListCustomerDataResponse, error) {
			resp, err := h.service.ListCustomerData(ctx, request)
			if err != nil {
				return ListCustomerDataResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			items := make([]api.CustomerAppData, 0, len(resp.Items))

			for _, customerData := range resp.Items {
				item, err := customerDataToAPI(customerData)
				if err != nil {
					return ListCustomerDataResponse{}, fmt.Errorf("failed to cast app customer data: %w", err)
				}

				items = append(items, item)
			}

			return ListCustomerDataResponse{
				Items:      items,
				Page:       resp.Page.PageNumber,
				PageSize:   resp.Page.PageSize,
				TotalCount: resp.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomerData"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type UpsertCustomerDataRequest struct {
	CustomerId customerentity.CustomerID
	Data       []api.CustomerAppData
}

type UpsertCustomerDataParams struct {
	CustomerId string
}

type (
	UpsertCustomerDataResponse = interface{}
	UpsertCustomerDataHandler  httptransport.HandlerWithArgs[UpsertCustomerDataRequest, UpsertCustomerDataResponse, UpsertCustomerDataParams]
)

// UpsertCustomerData returns a new httptransport.Handler for creating a customer.
func (h *handler) UpsertCustomerData() UpsertCustomerDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpsertCustomerDataParams) (UpsertCustomerDataRequest, error) {
			body := []api.CustomerAppData{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpsertCustomerDataRequest{}, fmt.Errorf("field to decode upsert customer data request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertCustomerDataRequest{}, err
			}

			customerId := customerentity.CustomerID{
				Namespace: ns,
				ID:        params.CustomerId,
			}

			return UpsertCustomerDataRequest{
				CustomerId: customerId,
				Data:       body,
			}, nil
		},
		func(ctx context.Context, req UpsertCustomerDataRequest) (UpsertCustomerDataResponse, error) {
			for _, apiCustomerData := range req.Data {
				app, customerData, err := h.getCustomerData(ctx, req.CustomerId.Namespace, apiCustomerData)
				if err != nil {
					return nil, err
				}

				err = app.UpsertCustomerData(ctx, appentity.UpsertAppInstanceCustomerDataInput{
					CustomerID: req.CustomerId,
					Data:       customerData,
				})
				if err != nil {
					return nil, err
				}
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[UpsertCustomerDataResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsertCustomerData"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type DeleteCustomerDataParams struct {
	CustomerId string
	AppId      string
}

type DeleteCustomerDataRequest struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID
}

type (
	DeleteCustomerDataResponse = interface{}
	DeleteCustomerDataHandler  httptransport.HandlerWithArgs[DeleteCustomerDataRequest, DeleteCustomerDataResponse, DeleteCustomerDataParams]
)

// DeleteCustomerData returns a handler for deleting a customer data.
func (h *handler) DeleteCustomerData() DeleteCustomerDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteCustomerDataParams) (DeleteCustomerDataRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerDataRequest{}, err
			}

			return DeleteCustomerDataRequest{
				CustomerID: customerentity.CustomerID{
					Namespace: ns,
					ID:        params.CustomerId,
				},
				AppID: appentitybase.AppID{
					Namespace: ns,
					ID:        params.AppId,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerDataRequest) (DeleteCustomerDataResponse, error) {
			// Get app
			app, err := h.service.GetApp(ctx, request.AppID)
			if err != nil {
				return nil, err
			}

			// Delete customer data
			err = app.DeleteCustomerData(ctx, appentity.DeleteAppInstanceCustomerDataInput{
				CustomerID: request.CustomerID,
			})
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteCustomerDataResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("deleteCustomerData"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// getCustomerData converts an API CustomerAppData to a list of CustomerData
func (h *handler) getCustomerData(ctx context.Context, namespace string, apiApp api.CustomerAppData) (appentity.App, appentity.CustomerData, error) {
	// Get app type
	appType, err := apiApp.Discriminator()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting app type: %w", err)
	}

	switch appType {
	// Sandbox app
	case string(appentitybase.AppTypeSandbox):
		// Parse as sandbox app
		apiSandboxCustomerData, err := apiApp.AsSandboxCustomerAppData()
		if err != nil {
			return nil, nil, fmt.Errorf("error converting to sandbox app: %w", err)
		}

		// Get app ID from API data or get default app
		app, err := h.getApp(ctx, namespace, apiSandboxCustomerData.Id, appentitybase.AppTypeSandbox)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting sandbox app: %w", err)
		}

		sandboxCustomerData := appsandbox.CustomerData{
			AppID: app.GetID(),
		}

		return app, sandboxCustomerData, nil

	// Stripe app
	case string(appentitybase.AppTypeStripe):
		// Parse as stripe app
		apiStripeCustomerData, err := apiApp.AsStripeCustomerAppData()
		if err != nil {
			return nil, nil, fmt.Errorf("error converting to stripe app: %w", err)
		}

		// Get app ID from API data or get default app
		app, err := h.getApp(ctx, namespace, apiStripeCustomerData.Id, appentitybase.AppTypeStripe)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting stripe app: %w", err)
		}

		stripeCustomerData := appstripeentity.CustomerData{
			AppID:                        app.GetID(),
			StripeCustomerID:             apiStripeCustomerData.StripeCustomerId,
			StripeDefaultPaymentMethodID: apiStripeCustomerData.StripeDefaultPaymentMethodId,
		}

		return app, stripeCustomerData, nil
	}

	return nil, nil, fmt.Errorf("unsupported app type: %s", appType)
}

// getApp gets an app by ID or gets the default app by type
func (h *handler) getApp(ctx context.Context, namespace string, appID *string, appType appentitybase.AppType) (appentity.App, error) {
	if appID != nil {
		app, err := h.service.GetApp(ctx, appentitybase.AppID{
			Namespace: namespace,
			ID:        *appID,
		})
		if err != nil {
			return nil, fmt.Errorf("error getting app by id: %w", err)
		}

		return app, nil
	}
	app, err := h.service.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
		Namespace: namespace,
		Type:      appType,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting default %s app: %w", appType, err)
	}

	return app, nil
}

// customerDataToAPI converts a CustomerData to an API CustomerAppData
func customerDataToAPI(a appentity.CustomerData) (api.CustomerAppData, error) {
	apiCustomerAppData := api.CustomerAppData{}

	appId := a.GetAppID().ID

	switch customerApp := a.(type) {
	case appstripeentity.CustomerData:
		apiStripeCustomerAppData := api.StripeCustomerAppData{
			Id:                           &appId,
			Type:                         api.StripeCustomerAppDataTypeStripe,
			StripeCustomerId:             customerApp.StripeCustomerID,
			StripeDefaultPaymentMethodId: customerApp.StripeDefaultPaymentMethodID,
		}

		err := apiCustomerAppData.FromStripeCustomerAppData(apiStripeCustomerAppData)
		if err != nil {
			return apiCustomerAppData, fmt.Errorf("error converting to stripe customer app: %w", err)
		}

	case appsandbox.CustomerData:
		apiSandboxCustomerAppData := api.SandboxCustomerAppData{
			Id:   &appId,
			Type: api.SandboxCustomerAppDataTypeSandbox,
		}

		err := apiCustomerAppData.FromSandboxCustomerAppData(apiSandboxCustomerAppData)
		if err != nil {
			return apiCustomerAppData, fmt.Errorf("error converting to sandbox customer app: %w", err)
		}

	default:
		return apiCustomerAppData, fmt.Errorf("unsupported customer data for app: %s", a.GetAppID().ID)
	}

	return apiCustomerAppData, nil
}
