package addons

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListAddonsRequest  = addon.ListAddonsInput
	ListAddonsResponse = response.PagePaginationResponse[apiv3.Addon]
	ListAddonsParams   = apiv3.ListAddonsParams
	ListAddonsHandler  httptransport.HandlerWithArgs[ListAddonsRequest, ListAddonsResponse, ListAddonsParams]
)

func (h *handler) ListAddons() ListAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAddonsParams) (ListAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAddonsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListAddonsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			return ListAddonsRequest{
				Namespaces: []string{ns},
				Page:       page,
			}, nil
		},
		func(ctx context.Context, request ListAddonsRequest) (ListAddonsResponse, error) {
			resp, err := h.service.ListAddons(ctx, request)
			if err != nil {
				return ListAddonsResponse{}, fmt.Errorf("failed to list add-ons: %w", err)
			}

			items := make([]apiv3.Addon, 0, len(resp.Items))
			for _, a := range resp.Items {
				apiAddon, err := ToAPIAddon(a)
				if err != nil {
					return ListAddonsResponse{}, fmt.Errorf("failed to convert add-on: %w", err)
				}
				items = append(items, apiAddon)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-addons"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
