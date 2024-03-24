package ingestdriver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	httptransport "github.com/openmeterio/openmeter/pkg/framework/transport/http"
	"github.com/openmeterio/openmeter/pkg/models"
)

// NewIngestEventsHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewIngestEventsHandler(
	op operation.Operation[ingest.IngestEventsRequest, bool],
	namespaceDecoder NamespaceDecoder,
	commonErrorEncoder httptransport.ErrorEncoder,
	errorHandler httptransport.ErrorHandler,
) http.Handler {
	return httptransport.NewHandler(
		op,
		(ingestEventsRequestDecoder{
			NamespaceDecoder: namespaceDecoder,
		}).decode,
		encodeIngestEventsResponse,
		(ingestEventsErrorEncoder{
			CommonErrorEncoder: commonErrorEncoder,
		}).encode,
		httptransport.WithErrorHandler(errorHandler),
		httptransport.WithOperationName("ingestEvents"),
	)
}

// NamespaceDecoder gets the namespace from the request.
type NamespaceDecoder interface {
	GetNamespace(ctx context.Context) (string, bool)
}

type StaticNamespaceDecoder string

func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) {
	return string(d), true
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
	NamespaceDecoder NamespaceDecoder
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
}

func encodeIngestEventsResponse(_ context.Context, w http.ResponseWriter, _ bool) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}

type ingestEventsErrorEncoder struct {
	CommonErrorEncoder httptransport.ErrorEncoder
}

func (e ingestEventsErrorEncoder) encode(ctx context.Context, err error, w http.ResponseWriter) bool {
	if e := (ErrorInvalidContentType{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w, nil)

		return true
	}

	if e := (ErrorInvalidEvent{}); errors.As(err, &e) {
		models.NewStatusProblem(ctx, e, http.StatusBadRequest).Respond(w, nil)

		return true
	}

	if e.CommonErrorEncoder != nil {
		return e.CommonErrorEncoder(ctx, err, w)
	}

	models.NewStatusProblem(ctx, errors.New("something went wrong"), http.StatusInternalServerError).Respond(w, nil)

	return false
}
