package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type EventsHandler interface {
	IngestEvents() IngestEventsHandler
}

type eventsHandler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          ingest.Service
	options          []httptransport.HandlerOption
}

func NewEventsHandler(
	resolveNamespace func(ctx context.Context) (string, error),
	service ingest.Service,
	options ...httptransport.HandlerOption,
) EventsHandler {
	return &eventsHandler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}

type (
	IngestEventsRequest  = ingest.IngestEventsRequest
	IngestEventsResponse = *struct{}
	IngestEventsHandler  httptransport.Handler[ingest.IngestEventsRequest, IngestEventsResponse]
)

func (h *eventsHandler) IngestEvents() IngestEventsHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ingest.IngestEventsRequest, error) {
			req := ingest.IngestEventsRequest{}

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
					return req, apierrors.NewBadRequestError(ctx, err, nil)
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
					return req, apierrors.NewBadRequestError(ctx, errors.New("no events found"), nil)
				}
			case "application/cloudevents+json":
				var apiRequest api.IngestEventsApplicationCloudeventsPlusJSONRequestBody

				err := json.NewDecoder(r.Body).Decode(&apiRequest)
				if err != nil {
					return req, apierrors.NewBadRequestError(ctx, err, nil)
				}

				req.Events = []event.Event{apiRequest}
			case "application/cloudevents-batch+json":
				var apiRequest api.IngestEventsApplicationCloudeventsBatchPlusJSONBody

				err := json.NewDecoder(r.Body).Decode(&apiRequest)
				if err != nil {
					return req, apierrors.NewBadRequestError(ctx, err, nil)
				}

				req.Events = apiRequest
			default:
				return req, apierrors.NewBadRequestError(ctx, errors.New("invalid content type"), nil)
			}

			return req, nil
		},
		func(ctx context.Context, request ingest.IngestEventsRequest) (IngestEventsResponse, error) {
			_, err := h.service.IngestEvents(ctx, request)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[IngestEventsResponse](http.StatusAccepted),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("ingest-metering-events"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
