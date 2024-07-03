package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/debug"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DebugHandler interface {
	GetMetrics() GetMetricsHandler
}

type debugHandler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
	debugConnector   debug.DebugConnector
	grantConnector   credit.GrantConnector
	options          []httptransport.HandlerOption
}

func NewDebugHandler(
	namespaceDecoder namespacedriver.NamespaceDecoder,
	debugConnector debug.DebugConnector,
	options ...httptransport.HandlerOption,
) DebugHandler {
	return &debugHandler{
		namespaceDecoder: namespaceDecoder,
		debugConnector:   debugConnector,
		options:          options,
	}
}

type GetMetricsHandlerRequestParams struct {
	Namespace string
}

type GetMetricsHandlerRequest struct {
	params GetMetricsHandlerRequestParams
}
type GetMetricsHandlerResponse = string
type GetMetricsHandlerParams struct{}
type GetMetricsHandler httptransport.HandlerWithArgs[GetMetricsHandlerRequest, GetMetricsHandlerResponse, GetMetricsHandlerParams]

func (h *debugHandler) GetMetrics() GetMetricsHandler {
	return httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](
		func(ctx context.Context, r *http.Request, params GetMetricsHandlerParams) (GetMetricsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetMetricsHandlerRequest{}, err
			}

			return GetMetricsHandlerRequest{
				params: GetMetricsHandlerRequestParams{
					Namespace: ns,
				},
			}, nil
		},
		func(ctx context.Context, request GetMetricsHandlerRequest) (string, error) {
			return h.debugConnector.GetDebugMetrics(ctx, request.params.Namespace)
		},
		commonhttp.PlainTextResponseEncoder[string],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*models.GenericUserError); ok {
					commonhttp.NewHTTPError(
						http.StatusBadRequest,
						err,
					).EncodeError(ctx, w)
					return true
				}
				return false
			}),
		)...,
	)
}

func (h *debugHandler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error"))
	}

	return ns, nil
}
