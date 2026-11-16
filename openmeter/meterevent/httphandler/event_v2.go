package httphandler

import (
	"context"
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ListEventsV2Params   = apiv1models.ListEventsV2Params
	ListEventsV2Request  = meterevent.ListEventsV2Params
	ListEventsV2Response = apiv1models.IngestedEventCursorPaginatedResponse
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
