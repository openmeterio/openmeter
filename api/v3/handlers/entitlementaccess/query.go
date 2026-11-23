package entitlementaccess

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const defaultPageSize = 100

type (
	QueryEntitlementAccessParams   = apiv3models.QueryEntitlementAccessParams
	QueryEntitlementAccessRequest  = entitlementaccess.QueryInput
	QueryEntitlementAccessResponse = apiv3models.EntitlementAccessQueryResponse
	QueryEntitlementAccessHandler  = httptransport.HandlerWithArgs[QueryEntitlementAccessRequest, QueryEntitlementAccessResponse, QueryEntitlementAccessParams]
)

func (h *handler) QueryEntitlementAccess() QueryEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryEntitlementAccessParams) (QueryEntitlementAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

			var body apiv3models.EntitlementAccessQueryRequest

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

			req := QueryEntitlementAccessRequest{
				Namespace:    ns,
				CustomerKeys: body.Customer.Keys,
			}

			if body.Feature != nil {
				req.FeatureKeys = body.Feature.Keys
			}

			if body.IncludeCredits != nil {
				req.IncludeCredits = *body.IncludeCredits
			}

			// The request validator already limits page[size] to 1..100.
			page, err := request.ParseCursorPage(ctx, (*apiv3models.CursorPaginationQueryPage)(params.Page), defaultPageSize, pagination.DecodeCursor)
			if err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

			req.PageSize = page.Size
			req.After = page.After
			req.Before = page.Before

			return req, nil
		},
		func(ctx context.Context, request QueryEntitlementAccessRequest) (QueryEntitlementAccessResponse, error) {
			res, err := h.entitlementAccessService.Query(ctx, request)
			if err != nil {
				return QueryEntitlementAccessResponse{}, err
			}

			return ToAPIEntitlementAccessQueryResponse(res, request.PageSize), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryEntitlementAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-entitlement-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
