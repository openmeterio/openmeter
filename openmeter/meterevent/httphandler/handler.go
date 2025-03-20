package httphandler

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	EventHandler
}

type EventHandler interface {
	ListEvents() ListEventsHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	namespaceDecoder  namespacedriver.NamespaceDecoder
	options           []httptransport.HandlerOption
	metereventService meterevent.Service
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	metereventService meterevent.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		namespaceDecoder:  namespaceDecoder,
		metereventService: metereventService,
		options:           options,
	}
}
