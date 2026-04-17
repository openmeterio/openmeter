package meters

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListMeters() ListMetersHandler
	GetMeter() GetMeterHandler
	CreateMeter() CreateMeterHandler
	UpdateMeter() UpdateMeterHandler
	DeleteMeter() DeleteMeterHandler
	QueryMeter() QueryMeterHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          meter.ManageService
	streaming        streaming.Connector
	customerService  customer.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service meter.ManageService,
	streaming streaming.Connector,
	customerService customer.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		streaming:        streaming,
		customerService:  customerService,
		options:          options,
	}
}
