package eventtypes

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

const defaultListNotificationEventTypesPageSize = 20

type ListNotificationEventTypesRequest struct {
	PageSize int
	After    *string
	Before   *string
	Type     *filter.FilterString
}

type (
	ListNotificationEventTypesResponse = response.CursorPaginationResponse[apiv3models.BillingNotificationEventTypeSummary]
	ListNotificationEventTypesParams   = apiv3models.ListBillingNotificationEventTypesParams
	ListNotificationEventTypesHandler  = httptransport.HandlerWithArgs[ListNotificationEventTypesRequest, ListNotificationEventTypesResponse, ListNotificationEventTypesParams]
)

func (h *handler) ListNotificationEventTypes() ListNotificationEventTypesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListNotificationEventTypesParams) (ListNotificationEventTypesRequest, error) {
			page, err := request.ParseCursorPage(ctx, params.Page, defaultListNotificationEventTypesPageSize, decodeEventTypeCursor)
			if err != nil {
				return ListNotificationEventTypesRequest{}, err
			}

			req := ListNotificationEventTypesRequest{PageSize: page.Size, After: page.After, Before: page.Before}

			if params.Filter != nil {
				typeFilter, err := filters.FromAPIFilterString(params.Filter.Type)
				if err != nil {
					return ListNotificationEventTypesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[type]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}

				req.Type = typeFilter
			}

			return req, nil
		},
		func(ctx context.Context, req ListNotificationEventTypesRequest) (ListNotificationEventTypesResponse, error) {
			// The registry is small and in memory, so filtering and paging happen
			// here, in its version-then-name order. A cursor is the event type at
			// the page edge; it is located in the unfiltered order, so changing the
			// filter between pages does not invalidate it.
			definitions := h.registry.EventTypes(nil)

			position := func(cursor *string, field string) (int, error) {
				index := slices.IndexFunc(definitions, func(d v4.EventTypeDefinition) bool {
					return d.EventType.String() == *cursor
				})
				if index < 0 {
					return 0, apierrors.NewBadRequestError(ctx, fmt.Errorf("unknown event type in cursor: %s", *cursor), apierrors.InvalidParameters{
						{Field: field, Reason: "the cursor does not point to a registered event type", Source: apierrors.InvalidParamSourceQuery},
					})
				}

				return index, nil
			}

			// [from, to) is the part of the order the page is taken from.
			from, to := 0, len(definitions)

			if req.After != nil {
				index, err := position(req.After, "page[after]")
				if err != nil {
					return ListNotificationEventTypesResponse{}, err
				}

				from = index + 1
			}

			if req.Before != nil {
				index, err := position(req.Before, "page[before]")
				if err != nil {
					return ListNotificationEventTypesResponse{}, err
				}

				to = index
			}

			// Filter the whole order, remembering each match's position in it, so
			// the page edges can tell whether any match lies beyond them.
			var (
				matching  []v4.EventTypeDefinition
				positions []int
			)

			for i, d := range definitions {
				if req.Type != nil {
					ok, err := req.Type.Match(d.EventType.String())
					if err != nil {
						return ListNotificationEventTypesResponse{}, err
					}

					if !ok {
						continue
					}
				}

				matching = append(matching, d)
				positions = append(positions, i)
			}

			// The matches inside [from, to) are matching[windowStart:windowEnd].
			windowStart := sort.SearchInts(positions, from)
			windowEnd := sort.SearchInts(positions, to)

			// A before page ends right before the cursor, so it takes the last
			// matches of the window; an after page or the first page takes the
			// first ones.
			start, end := windowStart, min(windowStart+req.PageSize, windowEnd)
			if req.Before != nil {
				start, end = max(windowEnd-req.PageSize, windowStart), windowEnd
			}

			page := matching[start:end]
			hasPrevious := start > 0
			hasNext := end < len(matching)

			var edges response.CursorPageEdges

			if len(page) > 0 {
				edges.First = lo.ToPtr(encodeEventTypeCursor(page[0].EventType.String()))
				edges.Last = lo.ToPtr(encodeEventTypeCursor(page[len(page)-1].EventType.String()))

				if hasNext {
					edges.Next = edges.Last
				}

				if hasPrevious {
					edges.Previous = edges.First
				}
			}

			resp := ListNotificationEventTypesResponse{
				Data: lo.Map(page, func(d v4.EventTypeDefinition, _ int) apiv3models.BillingNotificationEventTypeSummary {
					return ToAPIBillingNotificationEventTypeSummary(d)
				}),
				Meta: response.NewCursorMeta(req.PageSize, edges),
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListNotificationEventTypesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-billing-notification-event-types"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// Event type cursors are opaque to clients; they encode the event type at the
// page edge.
func encodeEventTypeCursor(eventType string) string {
	return base64.StdEncoding.EncodeToString([]byte(eventType))
}

func decodeEventTypeCursor(cursor string) (*string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil || len(decoded) == 0 {
		return nil, errors.New("invalid cursor")
	}

	return lo.ToPtr(string(decoded)), nil
}
