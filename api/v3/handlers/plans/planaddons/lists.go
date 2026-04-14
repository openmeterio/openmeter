package planaddons

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListPlanAddonsRequest = planaddon.ListPlanAddonsInput
	ListPlanAddonsParams  struct {
		PlanID string
		Params api.ListPlanAddonsParams
	}
	ListPlanAddonsResponse = response.PagePaginationResponse[api.PlanAddon]
	ListPlanAddonsHandler  httptransport.HandlerWithArgs[ListPlanAddonsRequest, ListPlanAddonsResponse, ListPlanAddonsParams]
)

func (h *handler) ListPlanAddons() ListPlanAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlanAddonsParams) (ListPlanAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlanAddonsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Params.Page.Number, 1),
					lo.FromPtrOr(params.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListPlanAddonsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			return ListPlanAddonsRequest{
				Namespaces: []string{ns},
				PlanIDs:    []string{params.PlanID},
				Page:       page,
			}, nil
		},
		func(ctx context.Context, req ListPlanAddonsRequest) (ListPlanAddonsResponse, error) {
			result, err := h.addonService.ListPlanAddons(ctx, req)
			if err != nil {
				return ListPlanAddonsResponse{}, err
			}

			items := make([]api.PlanAddon, 0, len(result.Items))
			for _, a := range result.Items {
				planAddon, err := toAPIPlanAddon(a)
				if err != nil {
					return ListPlanAddonsResponse{}, err
				}

				items = append(items, planAddon)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlanAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-plan-addons"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
