package charges

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListChargesRequest  = billingcharges.ListCustomerChargesInput
	ListChargesResponse = response.PagePaginationResponse[api.BillingCharge]
	ListChargesParams   = api.ListChargesParams
	ListChargesHandler  = httptransport.HandlerWithArgs[ListChargesRequest, ListChargesResponse, ListChargesParams]
)

func (h *handler) ListCharges() ListChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListChargesParams) (ListChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListChargesRequest{}, err
			}

			req, err := fromAPIListChargesParams(ctx, ns, params.Page, params.Sort, params.Expand)
			if err != nil {
				return ListChargesRequest{}, err
			}

			if params.Filter != nil {
				if err := fromAPIListChargesParamsFilter(ctx, params.Filter, &req); err != nil {
					return ListChargesRequest{}, err
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListChargesRequest) (ListChargesResponse, error) {
			result, err := h.service.ListCustomerCharges(ctx, request)
			if err != nil {
				return ListChargesResponse{}, fmt.Errorf("listing charges: %w", err)
			}

			return toAPIListChargesResponse(result, request.Page)
		},
		commonhttp.JSONResponseEncoderWithStatus[ListChargesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
