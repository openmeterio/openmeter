// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/debug"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
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
type (
	GetMetricsHandlerResponse = string
	GetMetricsHandlerParams   struct{}
	GetMetricsHandler         httptransport.HandlerWithArgs[GetMetricsHandlerRequest, GetMetricsHandlerResponse, GetMetricsHandlerParams]
)

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
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
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
