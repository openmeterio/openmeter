package meters

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListMeters() ListMetersHandler
	GetMeter() GetMeterHandler
	CreateMeter() CreateMeterHandler
	DeleteMeter() DeleteMeterHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          meter.ManageService
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service meter.ManageService,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
