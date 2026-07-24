package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	TokenHandler
}

type TokenHandler interface {
	CreateToken() CreateTokenHandler
	ListTokens() ListTokensHandler
	InvalidateToken() InvalidateTokenHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	portalService    portal.Service
	meterService     meter.Service
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
	portalService portal.Service,
	meterService meter.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if portalService == nil {
		errs = append(errs, errors.New("portal service is required"))
	}
	if meterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid portal handler config: %w", err)
	}

	return &handler{
		namespaceDecoder: namespaceDecoder,
		portalService:    portalService,
		meterService:     meterService,
		options:          options,
	}, nil
}
