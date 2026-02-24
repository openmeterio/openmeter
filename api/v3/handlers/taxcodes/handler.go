package taxcodes

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	// ListTaxCodes() ListTaxCodesHandler
	// GetTaxCode() GetTaxCodeHandler
	CreateTaxCode() CreateTaxCodeHandler
	// UpdateTaxCode() UpdateTaxCodeHandler
	// DeleteTaxCode() DeleteTaxCodeHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          taxcode.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service taxcode.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
