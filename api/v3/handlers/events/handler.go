package events

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	IngestEvents() IngestEventsHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          ingest.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service ingest.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
