package plans

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListPlansRequest  = plan.ListPlansInput
	ListPlansResponse = response.PagePaginationResponse[api.BillingPlan]
	ListPlansParams   = api.ListPlansParams
	ListPlansHandler  httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]
)

func (h *handler) ListPlans() ListPlansHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlansParams) (ListPlansRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlansRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			return ListPlansRequest{
				Namespaces: []string{ns},
				Page:       page,
			}, nil
		},
		func(ctx context.Context, req ListPlansRequest) (ListPlansResponse, error) {
			result, err := h.service.ListPlans(ctx, req)
			if err != nil {
				return ListPlansResponse{}, err
			}

			items := make([]api.BillingPlan, 0, len(result.Items))
			for _, p := range result.Items {
				// FIXME: For now we skip plans containing price types not representable in v3 (e.g., package, dynamic). We'll add full bidirectional transform later on.
				if hasUnsupportedV3Price(p) {
					continue
				}

				billingPlan, err := FromPlan(p)
				if err != nil {
					return ListPlansResponse{}, err
				}

				items = append(items, billingPlan)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlansResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-plans"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
