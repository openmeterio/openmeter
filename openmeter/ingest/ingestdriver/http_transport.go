package ingestdriver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

// NewIngestEventsHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewIngestEventsHandler(
	op operation.Operation[ingest.IngestEventsRequest, bool],
	namespaceDecoder namespacedriver.NamespaceDecoder,
	commonErrorEncoder httptransport.ErrorEncoder,
	errorHandler httptransport.ErrorHandler,
) httptransport.Handler[ingest.IngestEventsRequest, bool] {
	return httptransport.NewHandler(
		(ingestEventsRequestDecoder{
			NamespaceDecoder: namespaceDecoder,
		}).decode,
		op,
		encodeIngestEventsResponse,
		httptransport.WithErrorEncoder((ingestEventsErrorEncoder{
			CommonErrorEncoder: commonErrorEncoder,
		}).encode),
		httptransport.WithErrorHandler(errorHandler),
		httptransport.WithOperationName("ingestEvents"),
	)
}

type ErrorInvalidContentType struct {
	ContentType string
}

func (e ErrorInvalidContentType) Error() string {
	// return "invalid content type"

	return "invalid content type: " + e.ContentType
}

func (e ErrorInvalidContentType) Message() string {
	return "invalid content type: " + e.ContentType
}

func (e ErrorInvalidContentType) Details() map[string]any {
	return map[string]any{
		"contentType": e.ContentType,
	}
}

type ErrorInvalidEvent struct {
	Err error
}

func (e ErrorInvalidEvent) Error() string {
	// return "invalid event"

	return "invalid event: " + e.Err.Error()
}

func (e ErrorInvalidEvent) Message() string {
	return "invalid event: " + e.Err.Error()
}

type ingestEventsRequestDecoder struct {
	NamespaceDecoder namespacedriver.NamespaceDecoder
}

func (d ingestEventsRequestDecoder) decode(ctx context.Context, r *http.Request) (ingest.IngestEventsRequest, error) {
	var req ingest.IngestEventsRequest

	namespace, ok := d.NamespaceDecoder.GetNamespace(ctx)
	if !ok {
		return req, errors.New("namespace not found")
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

	// Validate the data content type of each event.
	for _, ev := range req.Events {
		contentType := ev.DataContentType()

		// In some event formats the datacontenttype attribute MAY be omitted.
		// For example, if a JSON format event has no datacontenttype attribute,
		// then it is implied that the data is a JSON value conforming to the "application/json" media type.
		// In other words: a JSON-format event with no datacontenttype is exactly equivalent to one with datacontenttype="application/json".
		// https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md#datacontenttype
		if contentType == "" {
			continue
		}

		if contentType != "application/json" {
			return req, ErrorInvalidEvent{
				Err: fmt.Errorf("unsupported cloudevents data content type with id %s: %s", ev.ID(), contentType),
			}
		}
	}

	return req, nil
}

func encodeIngestEventsResponse(_ context.Context, w http.ResponseWriter, _ bool) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}

type ingestEventsErrorEncoder struct {
	CommonErrorEncoder httptransport.ErrorEncoder
}

func (e ingestEventsErrorEncoder) encode(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
	if e := (ErrorInvalidContentType{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w)

		return true
	}

	if e := (ErrorInvalidEvent{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w)

		return true
	}

	if e.CommonErrorEncoder != nil {
		return e.CommonErrorEncoder(ctx, err, w, r)
	}

	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w)

	return false
}
