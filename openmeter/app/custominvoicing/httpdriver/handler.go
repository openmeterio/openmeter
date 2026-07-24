package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	AppHandler
}

type AppHandler interface {
	DraftSyncronized() DraftSyncronizedHandler
	IssuingSyncronized() IssuingSyncronizedHandler
	UpdatePaymentStatus() UpdatePaymentStatusHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service appcustominvoicing.SyncService

	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}

func New(
	service appcustominvoicing.SyncService,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if service == nil {
		errs = append(errs, errors.New("app custom invoicing service is required"))
	}
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid app custom invoicing handler config: %w", err)
	}

	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}, nil
}
