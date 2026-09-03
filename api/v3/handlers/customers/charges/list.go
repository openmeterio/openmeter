package charges

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// maxListCustomerChargesPageSize bounds a single page: every listed charge
// loads its full run history, and expands add per-row live rating and full
// invoice hydration on top.
const maxListCustomerChargesPageSize = 1000

type (
	ListCustomerChargesRequest  = billingcharges.ListCustomerChargesInput
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

			// See maxListCustomerChargesPageSize.
			if page.PageSize > maxListCustomerChargesPageSize {
				return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx,
					fmt.Errorf("page size must not exceed %d", maxListCustomerChargesPageSize),
					apierrors.InvalidParameters{
						{
							Field:  "page[size]",
							Reason: fmt.Sprintf("page size must not exceed %d", maxListCustomerChargesPageSize),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
			}

			// Realization runs are always required to compute booked totals;
			// the facade adds that expand itself. The request only carries the
			// expands the caller asked for.
			expands := meta.ExpandNone
			if args.Params.Expand != nil {
				chargesExpands, err := lo.MapErr(*args.Params.Expand, slicesx.WrapMapFn(convertAPIChargesExpand))
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "expand",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				expands = expands.With(chargesExpands...)
			}

			req := ListCustomerChargesRequest{
				ListChargesInput: billingcharges.ListChargesInput{
					Page:        page,
					Namespace:   ns,
					CustomerIDs: []string{args.CustomerID},
					// Credit purchases are served by the credit grants API; exclude them here.
					ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased},
					Expands:     expands,
				},
			}

			// Parse sort. When omitted, the service defaults to created_at ascending
			// with id as a tie-breaker (AIP-132 deterministic default order).
			if args.Params.Sort != nil {
				sort, err := request.ParseSortBy(*args.Params.Sort)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				orderBy, err := FromAPICustomerChargesSortField(ctx, sort.Field)
				if err != nil {
					return ListCustomerChargesRequest{}, err
				}
				req.OrderBy = orderBy
				req.Order = sort.Order.ToSortxOrder()
			}

			// Parse the filters. Each one is optional and validated
			// independently; the service-period filters are plain per-column
			// time filters the caller composes as needed.
			if args.Params.Filter != nil {
				status, err := filters.FromAPIFilterStringExact(args.Params.Filter.Status)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[status]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.Status = status

				// The search adapter hides deleted charges unless the request
				// opts in, so a status filter positively selecting "deleted"
				// (eq/oeq) must lift that guard; neq never unhides them.
				if f := args.Params.Filter.Status; f != nil {
					deleted := string(meta.ChargeStatusDeleted)
					// TODO: find a better solution to handle this
					req.IncludeDeleted = lo.FromPtr(f.Eq) == deleted || lo.Contains(f.Oeq, deleted)
				}

				featureID, err := filters.FromAPIFilterULID(args.Params.Filter.FeatureId)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[feature_id]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.FeatureID = featureID

				featureKey, err := filters.FromAPIFilterStringExact(args.Params.Filter.FeatureKey)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[feature_key]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.FeatureKey = featureKey

				servicePeriodFrom, err := filters.FromAPIFilterDateTime(args.Params.Filter.ServicePeriodFrom)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[service_period_from]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.ServicePeriodFrom = servicePeriodFrom

				servicePeriodTo, err := filters.FromAPIFilterDateTime(args.Params.Filter.ServicePeriodTo)
				if err != nil {
					return ListCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[service_period_to]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.ServicePeriodTo = servicePeriodTo
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerChargesRequest) (ListCustomerChargesResponse, error) {
			result, err := h.service.ListCustomerCharges(ctx, request)
			if err != nil {
				return ListCustomerChargesResponse{}, fmt.Errorf("listing charges: %w", err)
			}

			charges, err := slicesx.MapWithErr(result.Charges.Items, func(charge billingcharges.CustomerCharge) (api.BillingCharge, error) {
				return convertChargeToAPI(charge, result.Expands)
			})
			if err != nil {
				return ListCustomerChargesResponse{}, fmt.Errorf("converting charge: %w", err)
			}

			return response.NewPagePaginationResponse(charges, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.Charges.TotalCount),
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
