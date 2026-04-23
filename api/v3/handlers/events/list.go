package events

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const defaultListMeteringEventsPageSize = 20

type (
	ListMeteringEventsRequest  = meterevent.ListEventsV2Params
	ListMeteringEventsResponse = response.CursorPaginationResponse[api.MeteringIngestedEvent]
	ListMeteringEventsParams   = api.ListMeteringEventsParams
	ListMeteringEventsHandler  httptransport.HandlerWithArgs[ListMeteringEventsRequest, ListMeteringEventsResponse, ListMeteringEventsParams]
)

func (h *handler) ListMeteringEvents() ListMeteringEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListMeteringEventsParams) (ListMeteringEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListMeteringEventsRequest{}, err
			}

			req := ListMeteringEventsRequest{
				Namespace: ns,
			}

			pageSize := defaultListMeteringEventsPageSize
			if params.Page != nil {
				if params.Page.Before != nil {
					return ListMeteringEventsRequest{}, apierrors.NewBadRequestError(ctx, errors.New("page[before] is not supported"), apierrors.InvalidParameters{
						{
							Field:  "page[before]",
							Reason: "backward pagination is not supported",
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				if params.Page.After != nil {
					cursor, err := pagination.DecodeCursor(*params.Page.After)
					if err != nil {
						return ListMeteringEventsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{
								Field:  "page[after]",
								Reason: err.Error(),
								Source: apierrors.InvalidParamSourceQuery,
							},
						})
					}
					req.Cursor = cursor
				}

				if params.Page.Size != nil {
					pageSize = *params.Page.Size
				}
			}

			if pageSize < 1 || pageSize > meterevent.MaximumLimit {
				return ListMeteringEventsRequest{}, apierrors.NewBadRequestError(ctx, fmt.Errorf("page[size] must be between 1 and %d", meterevent.MaximumLimit), apierrors.InvalidParameters{
					{
						Field:  "page[size]",
						Reason: fmt.Sprintf("must be between 1 and %d", meterevent.MaximumLimit),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}
			req.Limit = lo.ToPtr(pageSize)

			if params.Filter != nil {
				if err := applyFilters(ctx, &req, params.Filter); err != nil {
					return ListMeteringEventsRequest{}, err
				}
			}

			sortBy, sortOrder, err := parseEventSort(ctx, params.Sort)
			if err != nil {
				return ListMeteringEventsRequest{}, err
			}
			req.SortBy = sortBy
			req.SortOrder = sortOrder

			return req, nil
		},
		func(ctx context.Context, req ListMeteringEventsRequest) (ListMeteringEventsResponse, error) {
			result, err := h.metereventService.ListEventsV2(ctx, req)
			if err != nil {
				return ListMeteringEventsResponse{}, err
			}

			items, err := slicesx.MapWithErr(result.Items, toAPIMeteringIngestedEvent)
			if err != nil {
				return ListMeteringEventsResponse{}, fmt.Errorf("convert events: %w", err)
			}

			pageSize := lo.FromPtrOr(req.Limit, defaultListMeteringEventsPageSize)
			resp := response.NewCursorPaginationResponse(result.Items, pageSize)

			return ListMeteringEventsResponse{
				Data: items,
				Meta: resp.Meta,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMeteringEventsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-metering-events"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func applyFilters(ctx context.Context, req *ListMeteringEventsRequest, f *api.ListEventsParamsFilter) error {
	id, err := filters.FromAPIFilterString(f.Id)
	if err != nil {
		return filterError(ctx, "filter[id]", err)
	}
	req.ID = id

	source, err := filters.FromAPIFilterString(f.Source)
	if err != nil {
		return filterError(ctx, "filter[source]", err)
	}
	req.Source = source

	subject, err := filters.FromAPIFilterString(f.Subject)
	if err != nil {
		return filterError(ctx, "filter[subject]", err)
	}
	req.Subject = subject

	typeFilter, err := filters.FromAPIFilterString(f.Type)
	if err != nil {
		return filterError(ctx, "filter[type]", err)
	}
	req.Type = typeFilter

	timeFilter, err := filters.FromAPIFilterDateTime(f.Time)
	if err != nil {
		return filterError(ctx, "filter[time]", err)
	}
	req.Time = timeFilter

	ingestedAt, err := filters.FromAPIFilterDateTime(f.IngestedAt)
	if err != nil {
		return filterError(ctx, "filter[ingested_at]", err)
	}
	req.IngestedAt = ingestedAt

	storedAt, err := filters.FromAPIFilterDateTime(f.StoredAt)
	if err != nil {
		return filterError(ctx, "filter[stored_at]", err)
	}
	req.StoredAt = storedAt

	customerID, err := fromAPICustomerIDFilter(ctx, f.CustomerId)
	if err != nil {
		return err
	}
	req.CustomerID = customerID

	return nil
}

// fromAPICustomerIDFilter maps the v3 customer_id filter to the backend filter,
// rejecting every operator that the underlying service cannot evaluate. Only
// `eq` and `oeq` are supported because ListEventsV2Params requires a concrete
// IN set.
func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}

	if f.Neq != nil || f.Contains != nil || len(f.Ocontains) > 0 || f.Exists != nil {
		return nil, filterError(ctx, "filter[customer_id]", errors.New("only eq and oeq operators are supported"))
	}

	var values []string
	if f.Eq != nil {
		values = append(values, *f.Eq)
	}
	if len(f.Oeq) > 0 {
		values = append(values, f.Oeq...)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return &filter.FilterString{In: &values}, nil
}

func filterError(ctx context.Context, field string, err error) error {
	return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
		{
			Field:  field,
			Reason: err.Error(),
			Source: apierrors.InvalidParamSourceQuery,
		},
	})
}

// parseEventSort resolves the public sort query into a backend sort field and direction.
func parseEventSort(ctx context.Context, sort *api.SortQuery) (streaming.EventSortField, sortx.Order, error) {
	if lo.FromPtr(sort) == "" {
		return "", "", nil
	}

	parsed, err := request.ParseSortBy(*sort)
	if err != nil {
		return "", "", apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "sort",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	var field streaming.EventSortField
	switch parsed.Field {
	case string(streaming.EventSortFieldTime):
		field = streaming.EventSortFieldTime
	case string(streaming.EventSortFieldIngestedAt):
		field = streaming.EventSortFieldIngestedAt
	case string(streaming.EventSortFieldStoredAt):
		field = streaming.EventSortFieldStoredAt
	default:
		err := fmt.Errorf("unsupported sort field: %q", parsed.Field)
		return "", "", apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "sort",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	// If the caller did not supply an explicit asc/desc suffix, default to
	// descending so `sort=time` behaves the same as omitting the parameter
	// (most recent first).
	order := parsed.Order.ToSortxOrder()
	if len(strings.Fields(string(*sort))) == 1 {
		order = sortx.OrderDesc
	}

	return field, order, nil
}
