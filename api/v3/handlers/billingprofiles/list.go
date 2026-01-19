package billingprofiles

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListBillingProfilesRequest  = billing.ListProfilesInput
	ListBillingProfilesResponse = response.PagePaginationResponse[api.BillingProfile]
	ListBillingProfilesParams   = api.ListBillingProfilesParams
	ListBillingProfilesHandler  httptransport.HandlerWithArgs[ListBillingProfilesRequest, ListBillingProfilesResponse, ListBillingProfilesParams]
)

func (h *handler) ListBillingProfiles() ListBillingProfilesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListBillingProfilesParams) (ListBillingProfilesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListBillingProfilesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListBillingProfilesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListBillingProfilesRequest{
				Page:      page,
				Namespace: ns,
				Expand: billing.ProfileExpand{
					Apps: true,
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListBillingProfilesRequest) (ListBillingProfilesResponse, error) {
			resp, err := h.service.ListProfiles(ctx, request)
			if err != nil {
				return ListBillingProfilesResponse{}, fmt.Errorf("failed to list billing profiles: %w", err)
			}

			profiles, err := ConvertProfilesToBillingProfiles(resp.Items)
			if err != nil {
				return ListBillingProfilesResponse{}, fmt.Errorf("failed to convert billing profiles: %w", err)
			}

			r := response.NewPagePaginationResponse(profiles, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListBillingProfilesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-billing-profiles"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
