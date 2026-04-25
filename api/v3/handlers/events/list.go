package events

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

			sortBy, sortOrder, err := fromAPIEventSort(ctx, params.Sort)
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
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[id]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.ID = id

	source, err := filters.FromAPIFilterString(f.Source)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[source]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.Source = source

	subject, err := filters.FromAPIFilterString(f.Subject)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[subject]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.Subject = subject

	typeFilter, err := filters.FromAPIFilterString(f.Type)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[type]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.Type = typeFilter

	timeFilter, err := filters.FromAPIFilterDateTime(f.Time)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[time]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.Time = timeFilter

	ingestedAt, err := filters.FromAPIFilterDateTime(f.IngestedAt)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[ingested_at]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.IngestedAt = ingestedAt

	storedAt, err := filters.FromAPIFilterDateTime(f.StoredAt)
	if err != nil {
		return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[stored_at]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}
	req.StoredAt = storedAt

	customerID, err := fromAPICustomerIDFilter(ctx, f.CustomerId)
	if err != nil {
		return err
	}
	req.CustomerID = customerID

	return nil
}
