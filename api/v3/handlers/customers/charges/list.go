package charges

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type (
	ListCustomerChargesRequest  = billingcharges.ListChargesInput
	ListCustomerChargesResponse = response.PagePaginationResponse[api.BillingCharge]
	ListCustomerChargesParams   struct {
		CustomerID api.ULID
		Params     api.ListCustomerChargesParams
	}
	ListCustomerChargesHandler = httptransport.HandlerWithArgs[ListCustomerChargesRequest, ListCustomerChargesResponse, ListCustomerChargesParams]
)

func (h *handler) ListCustomerCharges() ListCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCustomerChargesParams) (ListCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerChargesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if args.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(args.Params.Page.Number, 1),
					lo.FromPtrOr(args.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListCustomerChargesRequest{
				Page:        page,
				Namespace:   ns,
				CustomerIDs: []string{args.CustomerID},
				// Credit purchases are served by the credit grants API; exclude them here.
				ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased},
			}

			// Parse status filter
			if args.Params.Filter != nil && args.Params.Filter.Status != nil && len(args.Params.Filter.Status.Oeq) > 0 {
				statuses, err := parseChargeStatusFilterSlice(args.Params.Filter.Status.Oeq)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[status][oeq]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.StatusIn = statuses
			}

			// Parse expand
			if args.Params.Expand != nil {
				req.Expands = lo.FilterMap(*args.Params.Expand, func(exp api.BillingChargesExpand, _ int) (meta.Expand, bool) {
					if exp == api.BillingChargesExpandRealTimeUsage {
						return meta.ExpandRealizations, true
					}
					return "", false
				})
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerChargesRequest) (ListCustomerChargesResponse, error) {
			result, err := h.service.ListCharges(ctx, request)
			if err != nil {
				return ListCustomerChargesResponse{}, fmt.Errorf("listing charges: %w", err)
			}

			charges, err := slicesx.MapWithErr(result.Items, convertChargeToAPI)
			if err != nil {
				return ListCustomerChargesResponse{}, fmt.Errorf("converting charge: %w", err)
			}

			return response.NewPagePaginationResponse(charges, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerChargesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// parseChargeStatusFilterSlice converts a slice of status strings to meta.ChargeStatus values.
// Each token is validated with a type-safe switch so that unknown values are
// rejected with an explicit error message rather than caught by a generic validator.
func parseChargeStatusFilterSlice(values []string) ([]meta.ChargeStatus, error) {
	statuses := make([]meta.ChargeStatus, 0, len(values))

	for _, value := range values {
		s, err := convertAPIChargeStatus(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}

	return statuses, nil
}
