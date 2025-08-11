package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCustomerDataRequest  = app.ListCustomerInput
	ListCustomerDataResponse = api.CustomerAppDataPaginatedResponse
	ListCustomerDataHandler  httptransport.HandlerWithArgs[ListCustomerDataRequest, ListCustomerDataResponse, ListCustomerDataParams]
)

type ListCustomerDataParams struct {
	api.ListCustomerAppDataParams
	CustomerIdOrKey string
}

// ListCustomerData returns a handler for listing customers app data.
func (h *handler) ListCustomerData() ListCustomerDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerDataParams) (ListCustomerDataRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerDataRequest{}, err
			}

			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return ListCustomerDataRequest{}, err
			}

			req := ListCustomerDataRequest{
				CustomerID: cus.GetCustomerID(),

				// Pagination
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, customer.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, customer.DefaultPageNumber),
				},
			}

			if params.Type != nil {
				req.Type = lo.ToPtr(app.AppType(*params.Type))
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerDataRequest) (ListCustomerDataResponse, error) {
			resp, err := h.service.ListCustomerData(ctx, request)
			if err != nil {
				return ListCustomerDataResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			items := make([]api.CustomerAppData, 0, len(resp.Items))

			for _, customerApp := range resp.Items {
				item, err := h.toAPICustomerAppData(customerApp)
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
		)...,
	)
}

type UpsertCustomerDataRequest struct {
	CustomerId customer.CustomerID
	Data       []api.CustomerAppData
}

type UpsertCustomerDataParams struct {
	CustomerIdOrKey string
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

			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return UpsertCustomerDataRequest{}, err
			}

			return UpsertCustomerDataRequest{
				CustomerId: cus.GetCustomerID(),
				Data:       body,
			}, nil
		},
		func(ctx context.Context, req UpsertCustomerDataRequest) (UpsertCustomerDataResponse, error) {
			for _, apiCustomerData := range req.Data {
				customerApp, customerData, err := h.toCustomerData(ctx, req.CustomerId, apiCustomerData)
				if err != nil {
					return nil, err
				}

				err = customerApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
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
		)...,
	)
}

type DeleteCustomerDataParams struct {
	CustomerIdOrKey string
	AppId           string
}

type DeleteCustomerDataRequest struct {
	AppID      app.AppID
	CustomerID customer.CustomerID
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

			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					IDOrKey:   params.CustomerIdOrKey,
					Namespace: ns,
				},
			})
			if err != nil {
				return DeleteCustomerDataRequest{}, err
			}

			return DeleteCustomerDataRequest{
				CustomerID: cus.GetCustomerID(),
				AppID: app.AppID{
					Namespace: ns,
					ID:        params.AppId,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteCustomerDataRequest) (DeleteCustomerDataResponse, error) {
			// Get app
			existingApp, err := h.service.GetApp(ctx, request.AppID)
			if err != nil {
				return nil, err
			}

			// Delete customer data
			err = existingApp.DeleteCustomerData(ctx, app.DeleteAppInstanceCustomerDataInput{
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
		)...,
	)
}

// toCustomerData converts an API CustomerAppData to a CustomerData model
func (h *handler) toCustomerData(ctx context.Context, customerID customer.CustomerID, apiApp api.CustomerAppData) (app.App, app.CustomerData, error) {
	// Get app type
	appType, err := apiApp.Discriminator()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting app type: %w", err)
	}

	switch appType {
	// Sandbox app
	case string(app.AppTypeSandbox):
		// Parse as sandbox app
		apiSandboxCustomerData, err := apiApp.AsSandboxCustomerAppData()
		if err != nil {
			return nil, nil, fmt.Errorf("error converting to stripe app: %w", err)
		}

		// Resolve app
		resolvedApp, err := h.resolveCustomerApp(ctx, customerID, app.AppTypeSandbox, apiSandboxCustomerData.Id)
		if err != nil {
			return nil, nil, fmt.Errorf("error resolving sandbox app: %w", err)
		}

		// Create customer data
		sandboxCustomerData := appsandbox.CustomerData{}

		return resolvedApp, sandboxCustomerData, nil

	// Stripe app
	case string(app.AppTypeStripe):
		// Parse as stripe app
		apiStripeCustomerData, err := apiApp.AsStripeCustomerAppData()
		if err != nil {
			return nil, nil, fmt.Errorf("error converting to stripe app: %w", err)
		}

		// Resolve app
		resolvedApp, err := h.resolveCustomerApp(ctx, customerID, app.AppTypeStripe, apiStripeCustomerData.Id)
		if err != nil {
			return nil, nil, fmt.Errorf("error resolving stripe app: %w", err)
		}

		// Create customer data
		stripeCustomerData := fromAPIAppStripeCustomerData(apiStripeCustomerData)

		return resolvedApp, stripeCustomerData, nil
	case string(app.AppTypeCustomInvoicing):
		// Parse as custom invoicing app
		apiCustomInvoicingCustomerData, err := apiApp.AsCustomInvoicingCustomerAppData()
		if err != nil {
			return nil, nil, fmt.Errorf("error converting to custom invoicing app: %w", err)
		}

		// Resolve app
		resolvedApp, err := h.resolveCustomerApp(ctx, customerID, app.AppTypeCustomInvoicing, apiCustomInvoicingCustomerData.Id)
		if err != nil {
			return nil, nil, fmt.Errorf("error resolving custom invoicing app: %w", err)
		}

		// Create customer data
		customInvoicingCustomerData := appcustominvoicing.CustomerData{
			Metadata: lo.FromPtrOr(apiCustomInvoicingCustomerData.Metadata, map[string]string{}),
		}

		return resolvedApp, customInvoicingCustomerData, nil
	}

	return nil, nil, fmt.Errorf("unsupported app type: %s", appType)
}

// resolveCustomerApp resolves a customer app based on the app type or app ID.
func (h *handler) resolveCustomerApp(ctx context.Context, customerID customer.CustomerID, appType app.AppType, appID *string) (app.App, error) {
	var resolvedApp app.App
	var err error

	// Get app ID from API data or get default app for billing profile
	if appID != nil {
		return h.service.GetApp(ctx, app.GetAppInput{
			Namespace: customerID.Namespace,
			ID:        *appID,
		})
	}

	// Get the customer app by type
	resolvedApp, err = h.billingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
		CustomerID: customerID,
		AppType:    appType,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting customer app: %w", err)
	}

	return resolvedApp, nil
}

// toAPICustomerAppData converts a CustomerApp to an API CustomerAppData
func (h *handler) toAPICustomerAppData(a app.CustomerApp) (api.CustomerAppData, error) {
	apiCustomerAppData := api.CustomerAppData{}
	appId := a.App.GetID().ID

	switch customerAppData := a.CustomerData.(type) {
	case appstripeentity.CustomerData:
		stripeApp, ok := a.App.(appstripeentityapp.App)
		if !ok {
			return apiCustomerAppData, fmt.Errorf("error casting app to stripe app")
		}

		// Convert to API stripe customer app data
		apiStripeCustomerAppData := ToAPIStripeCustomerAppData(customerAppData, stripeApp)

		// Convert to API customer app data
		err := apiCustomerAppData.FromStripeCustomerAppData(apiStripeCustomerAppData)
		if err != nil {
			return apiCustomerAppData, fmt.Errorf("error converting to stripe customer app: %w", err)
		}

	case appsandbox.CustomerData:
		sandboxApp, ok := a.App.(appsandbox.App)
		if !ok {
			return apiCustomerAppData, fmt.Errorf("error casting app to sandbox app")
		}

		apiApp := mapSandboxAppToAPI(sandboxApp.Meta)

		apiSandboxCustomerAppData := api.SandboxCustomerAppData{
			Id:   &appId,
			Type: api.SandboxCustomerAppDataTypeSandbox,
			App:  &apiApp,
		}

		err := apiCustomerAppData.FromSandboxCustomerAppData(apiSandboxCustomerAppData)
		if err != nil {
			return apiCustomerAppData, fmt.Errorf("error converting to sandbox customer app: %w", err)
		}

	case appcustominvoicing.CustomerData:
		customInvoicingApp, ok := a.App.(appcustominvoicing.App)
		if !ok {
			return apiCustomerAppData, fmt.Errorf("error casting app to custom invoicing app")
		}

		apiApp := mapCustomInvoicingAppToAPI(customInvoicingApp.Meta)

		apiCustomInvoicingCustomerAppData := api.CustomInvoicingCustomerAppData{
			Id:   &appId,
			Type: api.CustomInvoicingCustomerAppDataTypeCustomInvoicing,
			App:  &apiApp,
		}

		err := apiCustomerAppData.FromCustomInvoicingCustomerAppData(apiCustomInvoicingCustomerAppData)
		if err != nil {
			return apiCustomerAppData, fmt.Errorf("error converting to custom invoicing customer app: %w", err)
		}
	default:
		return apiCustomerAppData, fmt.Errorf("unsupported customer data for app: %s", appId)
	}

	return apiCustomerAppData, nil
}
