package features

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListFeaturesRequest  = feature.ListFeaturesParams
	ListFeaturesResponse = response.PagePaginationResponse[api.Feature]
	ListFeaturesParams   = api.ListFeaturesParams
	ListFeaturesHandler  httptransport.HandlerWithArgs[ListFeaturesRequest, ListFeaturesResponse, ListFeaturesParams]
)

func (h *handler) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListFeaturesParams) (ListFeaturesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListFeaturesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListFeaturesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListFeaturesRequest{
				Namespace: ns,
				Page:      page,
			}

			if params.Filter != nil {
				meterIDs, err := filters.FromAPIFilterString(params.Filter.MeterId)
				if err != nil {
					return ListFeaturesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[meter_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.MeterIDs = meterIDs
			}

			return req, nil
		},
		func(ctx context.Context, req ListFeaturesRequest) (ListFeaturesResponse, error) {
			result, err := h.connector.ListFeatures(ctx, req)
			if err != nil {
				return ListFeaturesResponse{}, err
			}

			items := make([]api.Feature, 0, len(result.Items))
			for _, f := range result.Items {
				apiFeature, err := convertFeatureToAPI(f)
				if err != nil {
					return ListFeaturesResponse{}, err
				}
				items = append(items, apiFeature)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListFeaturesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-features"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
