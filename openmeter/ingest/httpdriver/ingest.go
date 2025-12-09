package httpdriver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	IngestEventsRequest  = ingest.IngestEventsRequest
	IngestEventsResponse = struct{}
	IngestEventsHandler  httptransport.Handler[IngestEventsRequest, IngestEventsResponse]
)

func (h *handler) IngestEvents() IngestEventsHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (IngestEventsRequest, error) {
			var req ingest.IngestEventsRequest

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return req, err
			}

			req.Namespace = namespace

			contentType := r.Header.Get("Content-Type")

			switch contentType {
			case "application/json":
				var apiRequest api.IngestEventsBody

				err := json.NewDecoder(r.Body).Decode(&apiRequest)
				if err != nil {
					return req, ErrorInvalidEvent{
						Err: err,
					}
				}

				// Try to parse as a single event
				e, err := apiRequest.AsEvent()
				if err == nil {
					req.Events = []event.Event{e}
				} else {
					// Try to parse as a batch of events
					e, err := apiRequest.AsIngestEventsBody1()
					if err == nil {
						req.Events = e
					}
				}

				// If we still don't have any events, return an error
				if len(req.Events) == 0 {
					return req, ErrorInvalidEvent{
						Err: errors.New("no events found"),
					}
				}
			case "application/cloudevents+json":
				var apiRequest api.IngestEventsApplicationCloudeventsPlusJSONRequestBody

				err := json.NewDecoder(r.Body).Decode(&apiRequest)
				if err != nil {
					return req, ErrorInvalidEvent{
						Err: err,
					}
				}

				req.Events = []event.Event{apiRequest}
			case "application/cloudevents-batch+json":
				var apiRequest api.IngestEventsApplicationCloudeventsBatchPlusJSONBody

				err := json.NewDecoder(r.Body).Decode(&apiRequest)
				if err != nil {
					return req, ErrorInvalidEvent{
						Err: err,
					}
				}

				req.Events = apiRequest
			default:
				return req, ErrorInvalidContentType{ContentType: contentType}
			}

			return req, nil
		},
		func(ctx context.Context, params IngestEventsRequest) (IngestEventsResponse, error) {
			_, err := h.service.IngestEvents(ctx, params)
			if err != nil {
				return IngestEventsResponse{}, err
			}

			return IngestEventsResponse{}, nil
		},
		commonhttp.EmptyResponseEncoder[IngestEventsResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("IngestEvents"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
