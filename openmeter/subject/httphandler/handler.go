package httpdriver

import (
	"context"
	"errors"
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
	entitlementConnector entitlement.Connector
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
	entitlementConnector entitlement.Connector,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		namespaceDecoder:     namespaceDecoder,
		options:              options,
		logger:               logger,
		subjectService:       subjectService,
		entitlementConnector: entitlementConnector,
	}
}
