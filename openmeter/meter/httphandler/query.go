package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListSubjectsResponse = *[]string
	ListSubjectsHandler  httptransport.HandlerWithArgs[ListSubjectsRequest, ListSubjectsResponse, ListSubjectsParams]
)

type ListSubjectsParams struct {
	IdOrSlug string
}

type ListSubjectsRequest struct {
	namespace string
	idOrSlug  string
}

// ListSubjects returns a handler for query meter.
func (h *handler) ListSubjects() ListSubjectsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubjectsParams) (ListSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubjectsRequest{}, err
			}

			return ListSubjectsRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
			}, nil
		},
		func(ctx context.Context, request ListSubjectsRequest) (ListSubjectsResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			subjectKeys, err := h.streaming.ListSubjects(ctx, streaming.ListSubjectsParams{
				Namespace: request.namespace,
				Meter:     &meter,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to list subjects: %w", err)
			}

			return &subjectKeys, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubjectsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
		)...,
	)
}

type (
	QueryMeterResponse = *api.MeterQueryResult
	QueryMeterHandler  httptransport.HandlerWithArgs[QueryMeterRequest, QueryMeterResponse, QueryMeterParams]
)

type QueryMeterParams struct {
	api.QueryMeterParams
	IdOrSlug string
}

type QueryMeterRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterParams
}

// QueryMeter returns a handler for query meter.
func (h *handler) QueryMeter() QueryMeterHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params QueryMeterParams) (QueryMeterRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterRequest{}, err
			}

			return QueryMeterRequest{
				namespace: ns,
				idOrSlug:  params.IdOrSlug,
				params:    params.QueryMeterParams,
			}, nil
		},
		func(ctx context.Context, request QueryMeterRequest) (QueryMeterResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := h.toQueryParamsFromRequest(
				ctx,
				meter,
				// Convert the POST request body to a GET request params
				ToRequestFromQueryParamsPOSTBody(request.params),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeter"),
		)...,
	)
}

type QueryMeterPostRequest struct {
	namespace string
	idOrSlug  string
	params    api.QueryMeterPostJSONRequestBody
}

type (
	QueryMeterPostParams   = string // meterIdOrSlug
	QueryMeterPostResponse = QueryMeterResponse
	QueryMeterPostHandler  httptransport.HandlerWithArgs[QueryMeterPostRequest, QueryMeterPostResponse, QueryMeterPostParams]
)

// QueryMeterPost returns a handler for query meter via POST.
func (h *handler) QueryMeterPost() QueryMeterPostHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, meterIdOrSlug QueryMeterPostParams) (QueryMeterPostRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return QueryMeterPostRequest{}, err
			}

			var request api.QueryMeterPostJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &request); err != nil {
				return QueryMeterPostRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			return QueryMeterPostRequest{
				namespace: ns,
				idOrSlug:  meterIdOrSlug,
				params:    request,
			}, nil
		},
		func(ctx context.Context, request QueryMeterPostRequest) (QueryMeterPostResponse, error) {
			meter, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
				Namespace: request.namespace,
				IDOrSlug:  request.idOrSlug,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get meter: %w", err)
			}

			params, err := h.toQueryParamsFromRequest(ctx, meter, request.params)
			if err != nil {
				return nil, fmt.Errorf("failed to construct query meter params: %w", err)
			}

			rows, err := h.streaming.QueryMeter(ctx, request.namespace, meter, params)
			if err != nil {
				return nil, fmt.Errorf("failed to query meter: %w", err)
			}

			response := ToAPIMeterQueryResult(request.params.From, request.params.To, request.params.WindowSize, rows)

			return &response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[QueryMeterPostResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("queryMeterPost"),
		)...,
	)
}
