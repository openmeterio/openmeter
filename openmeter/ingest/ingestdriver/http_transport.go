package ingestdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/internal/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	httptransport "github.com/openmeterio/openmeter/pkg/framework/transport/http"
)

// NewIngestEventsHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewIngestEventsHandler(op operation.Operation[ingest.IngestEventsRequest, bool], namespaceManager *namespace.Manager, errorHandler httptransport.ErrorHandler) http.Handler {
	return ingestdriver.NewIngestEventsHandler(op, namespaceManager, errorHandler)
}

type ErrorInvalidContentType = ingestdriver.ErrorInvalidContentType

type ErrorInvalidEvent = ingestdriver.ErrorInvalidEvent

type IngestEventsRequestDecoder = ingestdriver.IngestEventsRequestDecoder

func EncodeIngestEventsResponse(ctx context.Context, w http.ResponseWriter, response bool) error {
	return ingestdriver.EncodeIngestEventsResponse(ctx, w, response)
}

func EncodeIngestEventsError(ctx context.Context, err error, w http.ResponseWriter) bool {
	return ingestdriver.EncodeIngestEventsError(ctx, err, w)
}
