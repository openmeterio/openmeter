package entitlementaccess

import (
	"context"
	"fmt"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/entitlementaccess"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	defaultPageSize = 100
	maxPageSize     = 100
)

type (
	QueryEntitlementAccessParams   = apiv3.QueryEntitlementAccessParams
	QueryEntitlementAccessRequest  = entitlementaccess.QueryInput
	QueryEntitlementAccessResponse = apiv3.EntitlementAccessQueryResponse
	QueryEntitlementAccessHandler  = httptransport.HandlerWithArgs[QueryEntitlementAccessRequest, QueryEntitlementAccessResponse, QueryEntitlementAccessParams]
)

func (h *handler) QueryEntitlementAccess() QueryEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryEntitlementAccessParams) (QueryEntitlementAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

			var body apiv3.EntitlementAccessQueryRequest

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

			req := QueryEntitlementAccessRequest{
				Namespace:    ns,
				CustomerKeys: body.Customer.Keys,
				PageSize:     defaultPageSize,
			}

			if body.Feature != nil {
				req.FeatureKeys = body.Feature.Keys
			}

			if body.IncludeCredits != nil {
				req.IncludeCredits = *body.IncludeCredits
			}

			if err := applyPaging(ctx, &req, params); err != nil {
				return QueryEntitlementAccessRequest{}, err
			}

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

// applyPaging parses page[size]/page[after]/page[before] into the service input.
func applyPaging(ctx context.Context, req *QueryEntitlementAccessRequest, params QueryEntitlementAccessParams) error {
	if params.Page == nil {
		return nil
	}

	if params.Page.Size != nil {
		if *params.Page.Size < 1 || *params.Page.Size > maxPageSize {
			return apierrors.NewBadRequestError(ctx,
				fmt.Errorf("page[size] must be between 1 and %d", maxPageSize),
				apierrors.InvalidParameters{{
					Field:  "page[size]",
					Reason: fmt.Sprintf("must be between 1 and %d", maxPageSize),
					Source: apierrors.InvalidParamSourceQuery,
				}},
			)
		}

		req.PageSize = *params.Page.Size
	}

	if params.Page.After != nil && params.Page.Before != nil {
		return apierrors.NewBadRequestError(ctx,
			fmt.Errorf("page[after] and page[before] are mutually exclusive"),
			apierrors.InvalidParameters{{
				Field:  "page[after]",
				Reason: "cannot be combined with page[before]",
				Source: apierrors.InvalidParamSourceQuery,
			}},
		)
	}

	if params.Page.After != nil {
		cursor, err := decodeCursorParam(ctx, "page[after]", *params.Page.After)
		if err != nil {
			return err
		}
		req.After = cursor
	}

	if params.Page.Before != nil {
		cursor, err := decodeCursorParam(ctx, "page[before]", *params.Page.Before)
		if err != nil {
			return err
		}
		req.Before = cursor
	}

	return nil
}

func decodeCursorParam(ctx context.Context, field, raw string) (*pagination.Cursor, error) {
	cursor, err := pagination.DecodeCursor(raw)
	if err != nil {
		return nil, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{
			Field:  field,
			Reason: err.Error(),
			Source: apierrors.InvalidParamSourceQuery,
		}})
	}

	return cursor, nil
}
