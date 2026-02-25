package taxcodes

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	taxcode "github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListTaxCodesRequest  = taxcode.ListTaxCodesInput
	ListTaxCodesResponse = response.PagePaginationResponse[api.BillingTaxCode]
	ListTaxCodesParams   = api.ListTaxCodesParams
	ListTaxCodesHandler  = httptransport.HandlerWithArgs[ListTaxCodesRequest, ListTaxCodesResponse, ListTaxCodesParams]
)

func (h *handler) ListTaxCodes() ListTaxCodesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListTaxCodesParams) (ListTaxCodesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListTaxCodesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListTaxCodesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListTaxCodesRequest{
				Namespace:      ns,
				Page:           page,
				IncludeDeleted: lo.FromPtrOr(params.IncludeDeleted, false),
			}

			return req, nil
		},
		func(ctx context.Context, request ListTaxCodesRequest) (ListTaxCodesResponse, error) {
			resp, err := h.service.ListTaxCodes(ctx, taxcode.ListTaxCodesInput{
				Namespace:      request.Namespace,
				Page:           request.Page,
				IncludeDeleted: request.IncludeDeleted,
			})
			if err != nil {
				return ListTaxCodesResponse{}, err
			}

			taxcodes := lo.Map(resp.Items, func(item taxcode.TaxCode, _ int) api.BillingTaxCode {
				apiTaxCode, err := ConvertTaxCodeToAPITaxCode(item)
				if err != nil {
					// This should never happen, but if it does, we can skip the problematic tax code
					return api.BillingTaxCode{}
				}
				return apiTaxCode
			})

			r := response.NewPagePaginationResponse(taxcodes, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListTaxCodesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-tax-codes"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
