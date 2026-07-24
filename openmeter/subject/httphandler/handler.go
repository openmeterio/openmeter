package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	SubjectHandler
}

type SubjectHandler interface {
	GetSubject() GetSubjectHandler
	ListSubjects() ListSubjectsHandler
	UpsertSubject() UpsertSubjectHandler
	DeleteSubject() DeleteSubjectHandler
}

var _ Handler = (*handler)(nil)

type handler struct {
	namespaceDecoder     namespacedriver.NamespaceDecoder
	options              []httptransport.HandlerOption
	logger               *slog.Logger
	subjectService       subject.Service
	entitlementConnector entitlement.Service
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
	logger *slog.Logger,
	subjectService subject.Service,
	entitlementConnector entitlement.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if namespaceDecoder == nil {
		errs = append(errs, errors.New("namespace decoder is required"))
	}
	if logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}
	if subjectService == nil {
		errs = append(errs, errors.New("subject service is required"))
	}
	if entitlementConnector == nil {
		errs = append(errs, errors.New("entitlement connector is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid subject handler config: %w", err)
	}

	return &handler{
		namespaceDecoder:     namespaceDecoder,
		options:              options,
		logger:               logger,
		subjectService:       subjectService,
		entitlementConnector: entitlementConnector,
	}, nil
}
