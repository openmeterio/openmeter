package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	MeterHandler
}

type MeterHandler interface {
	ListMeters() ListMetersHandler
	GetMeter() GetMeterHandler
	CreateMeter() CreateMeterHandler
	UpdateMeter() UpdateMeterHandler
	DeleteMeter() DeleteMeterHandler
	QueryMeter() QueryMeterHandler
	QueryMeterPost() QueryMeterPostHandler
	QueryMeterCSV() QueryMeterCSVHandler
	ListSubjects() ListSubjectsHandler
	ListGroupByValues() ListGroupByValuesHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	options          []httptransport.HandlerOption
	customerService  customer.Service
	meterService     meter.ManageService
	streaming        streaming.Connector
	subjectService   subject.Service
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
	customerService customer.Service,
	meterService meter.ManageService,
	streaming streaming.Connector,
	subjectService subject.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		customerService:  customerService,
		meterService:     meterService,
		streaming:        streaming,
		subjectService:   subjectService,
	}
}
