package ingestdriver

import (
	"net/http"

	"github.com/openmeterio/openmeter/internal/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	httptransport "github.com/openmeterio/openmeter/pkg/framework/transport/http"
)

// NewIngestEventsHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewIngestEventsHandler(
	op operation.Operation[ingest.IngestEventsRequest, bool],
	namespaceDecoder NamespaceDecoder,
	commonErrorEncoder httptransport.ErrorEncoder,
	errorHandler httptransport.ErrorHandler,
) http.Handler {
	return ingestdriver.NewIngestEventsHandler(op, namespaceDecoder, commonErrorEncoder, errorHandler)
}

// NamespaceDecoder gets the namespace from the request.
type NamespaceDecoder = ingestdriver.NamespaceDecoder
