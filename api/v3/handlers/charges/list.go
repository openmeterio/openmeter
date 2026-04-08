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
			if args.Params.Filter != nil && args.Params.Filter.Status != nil {
				statuses, err := parseChargeStatusFilter(args.Params.Filter.Status.Oeq)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[status][oeq]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				// The service uses StatusNotIn, so we need to compute the complement:
				// all valid statuses minus the requested ones.
				req.StatusNotIn = chargeStatusComplement(statuses)
			}

			// Parse expand
			if args.Params.Expand != nil {
				for _, exp := range *args.Params.Expand {
					if exp == api.BillingChargesExpandRealTimeUsage {
						req.Expands = append(req.Expands, meta.ExpandRealizations)
					}
				}
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

// parseChargeStatusFilter parses a comma-separated list of charge statuses.
func parseChargeStatusFilter(oeq string) ([]meta.ChargeStatus, error) {
	if oeq == "" {
		return nil, fmt.Errorf("empty status filter")
	}

	parts := strings.Split(oeq, ",")
	statuses := make([]meta.ChargeStatus, 0, len(parts))

	for _, part := range parts {
		s := meta.ChargeStatus(strings.TrimSpace(part))
		if err := s.Validate(); err != nil {
			return nil, fmt.Errorf("invalid status %q: %w", part, err)
		}
		statuses = append(statuses, s)
	}

	return statuses, nil
}

// chargeStatusComplement returns all valid statuses not in the given set.
func chargeStatusComplement(include []meta.ChargeStatus) []meta.ChargeStatus {
	rawValues := meta.ChargeStatus("").Values()
	all := make([]meta.ChargeStatus, 0, len(rawValues))
	for _, v := range rawValues {
		all = append(all, meta.ChargeStatus(v))
	}

	includeSet := make(map[meta.ChargeStatus]bool, len(include))
	for _, s := range include {
		includeSet[s] = true
	}

	var complement []meta.ChargeStatus
	for _, s := range all {
		if !includeSet[s] {
			complement = append(complement, s)
		}
	}

	return complement
}
