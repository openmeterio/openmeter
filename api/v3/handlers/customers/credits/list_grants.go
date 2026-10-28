package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type (
	ListCreditGrantsRequest  = creditgrant.ListInput
	ListCreditGrantsResponse = response.PagePaginationResponse[api.BillingCreditGrant]
	ListCreditGrantsParams   struct {
		CustomerID api.ULID
		Params     api.ListCreditGrantsParams
	}
	ListCreditGrantsHandler httptransport.HandlerWithArgs[ListCreditGrantsRequest, ListCreditGrantsResponse, ListCreditGrantsParams]
)

func (h *handler) ListCreditGrants() ListCreditGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCreditGrantsParams) (ListCreditGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCreditGrantsRequest{}, err
			}

			req, err := fromAPIListCreditGrantsParams(ctx, args.Params)
			if err != nil {
				return ListCreditGrantsRequest{}, err
			}

			req.Namespace = ns
			req.CustomerID = args.CustomerID

			return req, nil
		},
		h.listCreditGrants,
		commonhttp.JSONResponseEncoderWithStatus[ListCreditGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-credit-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func (h *handler) listCreditGrants(ctx context.Context, request ListCreditGrantsRequest) (ListCreditGrantsResponse, error) {
	result, err := h.creditGrantService.List(ctx, request)
	if err != nil {
		return ListCreditGrantsResponse{}, fmt.Errorf("list credit grants: %w", err)
	}

	grants, err := slicesx.MapWithErr(result.Items, func(item creditpurchase.Charge) (api.BillingCreditGrant, error) {
		return toAPIBillingCreditGrant(item)
	})
	if err != nil {
		return ListCreditGrantsResponse{}, fmt.Errorf("converting credit grants: %w", err)
	}

	return response.NewPagePaginationResponse(grants, response.PageMetaPage{
		Size:   request.Page.PageSize,
		Number: request.Page.PageNumber,
		Total:  lo.ToPtr(result.TotalCount),
	}), nil
}
