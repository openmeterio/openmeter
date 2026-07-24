package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	PlanAddonHandler
}

type PlanAddonHandler interface {
	ListPlanAddons() ListPlanAddonsHandler
	CreatePlanAddon() CreatePlanAddonHandler
	DeletePlanAddon() DeletePlanAddonHandler
	GetPlanAddon() GetPlanAddonHandler
	UpdatePlanAddon() UpdatePlanAddonHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	service          planaddon.Service
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
	namespaceDecoder namespacedriver.NamespaceDecoder,
	service planaddon.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if service == nil {
		errs = append(errs, errors.New("plan add-on service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid plan add-on handler config: %w", err)
	}

	return &handler{
		service:          service,
		namespaceDecoder: namespaceDecoder,
		options:          options,
	}, nil
}
