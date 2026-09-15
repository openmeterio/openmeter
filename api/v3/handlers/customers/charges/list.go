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
	ListCustomerChargesRequest  = billingcharges.ListCustomerChargesInput
	ListCustomerChargesResponse = response.PagePaginationResponse[api.BillingCharge]
	ListCustomerChargesParams   struct {
		CustomerID api.ULID
		Params     api.ListCustomerChargesParams
	}
	ListCustomerChargesHandler = httptransport.HandlerWithArgs[ListCustomerChargesRequest, ListCustomerChargesResponse, ListCustomerChargesParams]
)

func (h *handler) ListCustomerCharges() ListCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCustomerChargesParams) (ListCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerChargesRequest{}, err
			}

			req, err := fromAPIListChargesParams(ctx, ns, args.Params.Page, args.Params.Sort, args.Params.Expand)
			if err != nil {
				return ListCustomerChargesRequest{}, err
			}

			req.CustomerIDs = []string{args.CustomerID}

			if args.Params.Filter != nil {
				if err := fromAPIListChargesParamsFilter(ctx, &api.ListChargesParamsFilter{
					Status:            args.Params.Filter.Status,
					FeatureId:         args.Params.Filter.FeatureId,
					FeatureKey:        args.Params.Filter.FeatureKey,
					ServicePeriodFrom: args.Params.Filter.ServicePeriodFrom,
					ServicePeriodTo:   args.Params.Filter.ServicePeriodTo,
				}, &req); err != nil {
					return ListCustomerChargesRequest{}, err
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerChargesRequest) (ListCustomerChargesResponse, error) {
			result, err := h.service.ListCustomerCharges(ctx, request)
			if err != nil {
				return ListCustomerChargesResponse{}, fmt.Errorf("listing charges: %w", err)
			}

			return toAPIListChargesResponse(result, request.Page)
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerChargesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
