package httphandler

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ListEventsV2Params   = api.ListEventsV2Params
	ListEventsV2Request  = meterevent.ListEventsV2Params
	ListEventsV2Response = api.IngestedEventCursorPaginatedResponse
	ListEventsV2Handler  httptransport.HandlerWithArgs[ListEventsV2Request, ListEventsV2Response, ListEventsV2Params]
)

// ListEventsV2 returns a handler for listing events.
func (h *handler) ListEventsV2() ListEventsV2Handler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEventsV2Params) (ListEventsV2Request, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEventsV2Request{}, err
			}

			p, err := convertListEventsV2Params(params, ns)
			if err != nil {
				return ListEventsV2Request{}, models.NewGenericValidationError(err)
			}

			return p, nil
		},
		func(ctx context.Context, params ListEventsV2Request) (ListEventsV2Response, error) {
			events, err := h.metereventService.ListEventsV2(ctx, params)
			if err != nil {
				return ListEventsV2Response{}, err
			}

			return convertListEventsV2Response(events)
		},
		commonhttp.JSONResponseEncoderWithStatus[ListEventsV2Response](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listEventsV2"),
		)...,
	)
}
