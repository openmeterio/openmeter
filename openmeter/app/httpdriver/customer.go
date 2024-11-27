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
			httptransport.WithOperationName("listCustomers"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
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
