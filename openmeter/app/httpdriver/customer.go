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

type (
	UpsertCustomerDataRequest  = []app.UpsertCustomerDataInput
	UpsertCustomerDataResponse = interface{}
	UpsertCustomerDataHandler  httptransport.HandlerWithArgs[UpsertCustomerDataRequest, UpsertCustomerDataResponse, UpsertCustomerDataParams]
)

type UpsertCustomerDataParams struct {
	CustomerId string
}

// UpsertCustomerData returns a new httptransport.Handler for creating a customer.
func (h *handler) UpsertCustomerData() UpsertCustomerDataHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpsertCustomerDataParams) (UpsertCustomerDataRequest, error) {
			body := []api.CustomerAppData{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpsertCustomerDataRequest{}, fmt.Errorf("field to decode create customer request: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertCustomerDataRequest{}, err
			}

			customerId := customerentity.CustomerID{
				Namespace: ns,
				ID:        params.CustomerId,
			}

			reqs := make(UpsertCustomerDataRequest, 0, len(body))

			for _, apiCustomerData := range body {
				data, err := toCustomerData(ctx, h.service, customerId, apiCustomerData)
				if err != nil {
					return UpsertCustomerDataRequest{}, fmt.Errorf("failed to convert customer data: %w", err)
				}

				reqs = append(reqs, app.UpsertCustomerDataInput{
					AppID:      data.GetAppID(),
					CustomerID: customerId,
					Data:       data,
				})
			}

			return reqs, nil
		},
		func(ctx context.Context, reqs UpsertCustomerDataRequest) (UpsertCustomerDataResponse, error) {
			for _, req := range reqs {
				err := h.service.UpsertCustomerData(ctx, req)
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

// toCustomerData converts an API CustomerAppData to a list of CustomerData
func toCustomerData(ctx context.Context, service app.Service, customerID customerentity.CustomerID, apiApp api.CustomerAppData) (appentity.CustomerData, error) {
	// Get app type
	appType, err := apiApp.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("error getting app type: %w", err)
	}

	switch appType {
	// Sandbox app
	case string(appentitybase.AppTypeSandbox):
		sandboxCustomerData, err := apiApp.AsSandboxCustomerAppData()
		if err != nil {
			return nil, fmt.Errorf("error converting to sandbox app: %w", err)
		}

		// Get app ID from API data or get default app
		var appID appentitybase.AppID

		if sandboxCustomerData.Id != nil {
			appID = appentitybase.AppID{
				Namespace: customerID.Namespace,
				ID:        *sandboxCustomerData.Id,
			}
		} else {
			app, err := service.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
				Namespace: customerID.Namespace,
				Type:      appentitybase.AppTypeSandbox,
			})
			if err != nil {
				return nil, fmt.Errorf("error getting default sandbox app: %w", err)
			}

			appID = app.GetID()
		}

		return appsandbox.CustomerData{
			AppID:      appID,
			CustomerID: customerID,
		}, nil

	// Stripe app
	case string(appentitybase.AppTypeStripe):
		stripeCustomerData, err := apiApp.AsStripeCustomerAppData()
		if err != nil {
			return nil, fmt.Errorf("error converting to stripe app: %w", err)
		}

		// Get app ID from API data or get default app
		var appID appentitybase.AppID

		if stripeCustomerData.Id != nil {
			appID = appentitybase.AppID{
				Namespace: customerID.Namespace,
				ID:        *stripeCustomerData.Id,
			}
		} else {
			app, err := service.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
				Namespace: customerID.Namespace,
				Type:      appentitybase.AppTypeStripe,
			})
			if err != nil {
				return nil, fmt.Errorf("error getting default sandbox app: %w", err)
			}

			appID = app.GetID()
		}

		return appstripeentity.CustomerData{
			AppID:                        appID,
			CustomerID:                   customerID,
			StripeCustomerID:             stripeCustomerData.StripeCustomerId,
			StripeDefaultPaymentMethodID: stripeCustomerData.StripeDefaultPaymentMethodId,
		}, nil
	}

	return nil, fmt.Errorf("unsupported app type: %s", appType)
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
