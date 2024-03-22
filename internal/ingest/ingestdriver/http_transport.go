package ingestdriver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	httptransport "github.com/openmeterio/openmeter/pkg/framework/transport/http"
	"github.com/openmeterio/openmeter/pkg/models"
)

// NewIngestEventsHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewIngestEventsHandler(op operation.Operation[ingest.IngestEventsRequest, bool], namespaceManager *namespace.Manager, errorHandler httptransport.ErrorHandler) http.Handler {
	requestDecoder := IngestEventsRequestDecoder{
		NamespaceManager: namespaceManager,
	}

	return httptransport.NewHandler(
		op,
		requestDecoder.Decode,
		EncodeIngestEventsResponse,
		EncodeIngestEventsError,
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

type IngestEventsRequestDecoder struct {
	NamespaceManager *namespace.Manager
}

func (d IngestEventsRequestDecoder) Decode(ctx context.Context, r *http.Request) (ingest.IngestEventsRequest, error) {
	var req ingest.IngestEventsRequest

	req.Namespace = d.NamespaceManager.GetDefaultNamespace()

	contentType := r.Header.Get("Content-Type")

	switch contentType {
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
		return ingest.IngestEventsRequest{}, ErrorInvalidContentType{ContentType: contentType}
	}

	return req, nil
}

func EncodeIngestEventsResponse(_ context.Context, w http.ResponseWriter, _ bool) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func EncodeIngestEventsError(ctx context.Context, err error, w http.ResponseWriter) bool {
	if e := (ErrorInvalidContentType{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w, nil)

		return true
	}

	if e := (ErrorInvalidEvent{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w, nil)

		return true
	}

	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w, nil)

	return true
}
