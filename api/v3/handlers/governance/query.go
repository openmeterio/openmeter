package governance

import (
	"context"
	"fmt"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	defaultPageSize = 100
	maxPageSize     = 100
)

type (
	QueryGovernanceAccessParams   = apiv3.QueryGovernanceAccessParams
	QueryGovernanceAccessRequest  = governance.QueryAccessInput
	QueryGovernanceAccessResponse = apiv3.GovernanceQueryResponse
	QueryGovernanceAccessHandler  = httptransport.HandlerWithArgs[QueryGovernanceAccessRequest, QueryGovernanceAccessResponse, QueryGovernanceAccessParams]
)

func (h *handler) QueryGovernanceAccess() QueryGovernanceAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryGovernanceAccessParams) (QueryGovernanceAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryGovernanceAccessRequest{}, err
			}

			var body apiv3.GovernanceQueryRequest

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return QueryGovernanceAccessRequest{}, err
			}

			req := QueryGovernanceAccessRequest{
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
				return QueryGovernanceAccessRequest{}, err
			}

			return req, nil
		},
		func(ctx context.Context, request QueryGovernanceAccessRequest) (QueryGovernanceAccessResponse, error) {
			res, err := h.governanceService.QueryAccess(ctx, request)
			if err != nil {
				return QueryGovernanceAccessResponse{}, err
			}

			return ToAPIGovernanceQueryResponse(res, request.PageSize), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryGovernanceAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("query-governance-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// applyPaging parses page[size]/page[after]/page[before] into the service input.
func applyPaging(ctx context.Context, req *QueryGovernanceAccessRequest, params QueryGovernanceAccessParams) error {
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
