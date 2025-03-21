package httphandler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListEventsParams   = api.ListEventsParams
	ListEventsResponse = []api.IngestedEvent
	ListEventsHandler  httptransport.HandlerWithArgs[ListEventsRequest, ListEventsResponse, ListEventsParams]
)

type ListEventsRequest = meterevent.ListEventsParams

// ListEvents returns a handler for listing events.
func (h *handler) ListEvents() ListEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsParams) (ListEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsRequest{}, err
			}

			// We add a second to avoid validation issues
			minimumFrom := time.Now().Add(-meterevent.MaximumFromDuration).Add(time.Second)

			return ListEventsRequest{
				Namespace:      ns,
				ClientID:       params.ClientId,
				From:           lo.FromPtrOr(params.From, minimumFrom),
				To:             params.To,
				IngestedAtFrom: params.IngestedAtFrom,
				IngestedAtTo:   params.IngestedAtTo,
				ID:             params.Id,
				Subject:        params.Subject,
				Limit:          lo.FromPtrOr(params.Limit, meterevent.MaximumLimit),
			}, nil
		},
		func(ctx context.Context, request ListEventsRequest) (ListEventsResponse, error) {
			events, err := h.metereventService.ListEvents(ctx, request)
			if err != nil {
				return ListEventsResponse{}, fmt.Errorf("failed to list events: %w", err)
			}

			result := make(ListEventsResponse, len(events))
			for i, event := range events {
				result[i], err = convertEvent(event)
				if err != nil {
					return ListEventsResponse{}, fmt.Errorf("failed to convert event: %w", err)
				}
			}

			return result, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListEventsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEvents"),
		)...,
	)
}
