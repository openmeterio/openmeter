package events

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const defaultListEventSubjectsPageSize = 20

type (
	ListEventSubjectsRequest  = meterevent.ListSubjectsParams
	ListEventSubjectsResponse = response.CursorPaginationResponse[api.MeteringEventSubject]
	ListEventSubjectsParams   = api.ListEventSubjectsParams
	ListEventSubjectsHandler  httptransport.HandlerWithArgs[ListEventSubjectsRequest, ListEventSubjectsResponse, ListEventSubjectsParams]
)

func (h *handler) ListEventSubjects() ListEventSubjectsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventSubjectsParams) (ListEventSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventSubjectsRequest{}, err
			}

			req := ListEventSubjectsRequest{
				Namespace: ns,
			}

			pageSize := defaultListEventSubjectsPageSize
			if params.Page != nil {
				if params.Page.Before != nil {
					return ListEventSubjectsRequest{}, apierrors.NewBadRequestError(ctx, errors.New("page[before] is not supported"), apierrors.InvalidParameters{
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
						return ListEventSubjectsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
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
				return ListEventSubjectsRequest{}, apierrors.NewBadRequestError(ctx, fmt.Errorf("page[size] must be between 1 and %d", meterevent.MaximumLimit), apierrors.InvalidParameters{
					{
						Field:  "page[size]",
						Reason: fmt.Sprintf("must be between 1 and %d", meterevent.MaximumLimit),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}
			req.Limit = lo.ToPtr(pageSize)

			if params.Filter != nil {
				key, err := filters.FromAPIFilterString(params.Filter.Key)
				if err != nil {
					return ListEventSubjectsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[key]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.Key = key

				attributed, err := filters.FromAPIFilterBoolean(params.Filter.Attributed)
				if err != nil {
					return ListEventSubjectsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[attributed]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				if attributed != nil {
					req.Attributed = attributed.Eq
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListEventSubjectsRequest) (ListEventSubjectsResponse, error) {
			result, err := h.metereventService.ListSubjects(ctx, req)
			if err != nil {
				return ListEventSubjectsResponse{}, err
			}

			items := lo.Map(result.Items, func(s meterevent.Subject, _ int) api.MeteringEventSubject {
				return api.MeteringEventSubject{
					Key: s.Key,
				}
			})

			meta := response.CursorMeta{
				Page: response.CursorMetaPage{
					Next:     nullable.NewNullNullable[string](),
					Previous: nullable.NewNullNullable[string](),
					Size:     lo.FromPtr(req.Limit),
				},
			}

			if len(result.Items) > 0 {
				meta.Page.First = lo.ToPtr(result.Items[0].Cursor().Encode())
				meta.Page.Last = lo.ToPtr(result.Items[len(result.Items)-1].Cursor().Encode())
			}

			// The attributed filter is applied after pagination, so a full page does
			// not imply more data and a short page does not imply exhaustion. The
			// service-computed cursor is the only reliable continuation signal.
			if result.NextCursor != nil {
				meta.Page.Next = nullable.NewNullableWithValue(result.NextCursor.Encode())
			}

			return ListEventSubjectsResponse{
				Data: items,
				Meta: meta,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListEventSubjectsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-event-subjects"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
