package httpdriver

import (
	"context"
	"errors"
	"fmt"
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
	QueryMeterPostCSV() QueryMeterPostCSVHandler
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
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if customerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}
	if meterService == nil {
		errs = append(errs, errors.New("meter service is required"))
	}
	if streaming == nil {
		errs = append(errs, errors.New("streaming connector is required"))
	}
	if subjectService == nil {
		errs = append(errs, errors.New("subject service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid meter handler config: %w", err)
	}

	return &handler{
		namespaceDecoder: namespaceDecoder,
		options:          options,
		customerService:  customerService,
		meterService:     meterService,
		streaming:        streaming,
		subjectService:   subjectService,
	}, nil
}
