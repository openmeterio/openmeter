package httpdriver

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

type ListEventsRequest struct {
	namespace string
	ListEventsParams
}

// ListEvents returns a handler for listing events.
func (h *handler) ListEvents() ListEventsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsParams) (ListEventsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsRequest{}, err
			}

			return ListEventsRequest{
				namespace:        ns,
				ListEventsParams: params,
			}, nil
		},
		func(ctx context.Context, request ListEventsRequest) (ListEventsResponse, error) {
			// We add a second to avoid validation issues
			minimumFrom := time.Now().Add(-meterevent.MaximumFromDuration).Add(time.Second)

			result, err := h.metereventService.ListEvents(ctx, meterevent.ListEventsInput{
				Namespace:      request.namespace,
				ClientID:       request.ClientId,
				IngestedAtFrom: request.IngestedAtFrom,
				IngestedAtTo:   request.IngestedAtTo,
				From:           lo.FromPtrOr(request.From, minimumFrom),
				To:             request.To,
				ID:             request.Id,
				Subject:        request.Subject,
				Limit:          lo.FromPtrOr(request.Limit, meterevent.MaximumLimit),
			})
			if err != nil {
				return ListEventsResponse{}, fmt.Errorf("failed to list events: %w", err)
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
